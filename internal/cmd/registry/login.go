package registrycmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ociauth"
	"cuelang.org/go/mod/modconfig"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/dockercfg"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/publish"
)

// target is the registry host a login addresses, with its scheme choice.
type target struct {
	host     string
	insecure bool
}

// arg renders the target back as a runnable command argument.
func (t target) arg() string {
	if t.insecure {
		return t.host + "+insecure"
	}
	return t.host
}

// errBadCredential marks a credential the registry judged and rejected: a
// refusal that leaves the file untouched, never a connectivity failure.
var errBadCredential = errors.New("the registry rejected the credential")

// stdinFd is stdin's descriptor for the terminal checks.
//
//nolint:gosec // G115: stdin's fd is a small non-negative number on every supported platform
func stdinFd() int {
	return int(os.Stdin.Fd())
}

// NewRegistryLoginCmd creates the registry login command.
func NewRegistryLoginCmd(cfg *config.GlobalConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "login [host]",
		Short: "Log in to an OCI registry",
		Long: `Store a credential for an OCI registry in the standard docker credential
file — the store OPM's push and pull both read, and the same file
'docker login' writes. The credential is verified against the registry
before anything is written; a rejected credential leaves the file
untouched, and everything in the file outside the one host entry is
preserved.

	With a host argument the host is taken literally (docker-login semantics);
	append +insecure to a host served over plain HTTP. Without one, the
	configured registry mapping (--registry > OPM_REGISTRY > config.registry)
	is resolved to its host set: a single host proceeds; several refuse,
	listing each as a runnable command.

	This command is interactive by design and prompts for the username and
	secret. In CI, use 'docker login' — it writes the same file.

	Exit codes: 0 written, 2 refusal, 3 registry unreachable.

	Examples:
	  # Log in to the host the configured registry resolves to
	  opm registry login

	  # Log in to a named host
	  opm registry login ghcr.io

	  # A local registry served over plain HTTP
	  opm registry login localhost:5000+insecure`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			return runLogin(c, cfg, args)
		},
	}
	return c
}

// runLogin resolves the target, prompts, probes, and writes — refusing
// before any of it when stdin is not a terminal.
func runLogin(c *cobra.Command, cfg *config.GlobalConfig, args []string) error {
	var tgt target
	if len(args) == 1 {
		tgt = parseHostArg(args[0])
	} else {
		reportResolution(cfg)
		var refusal *publish.Refusal
		tgt, refusal = resolveTarget(cfg.Registry)
		if refusal != nil {
			return refuse(*refusal)
		}
	}

	if os.Getenv("DOCKER_AUTH_CONFIG") != "" {
		output.Warn("DOCKER_AUTH_CONFIG is set; it shadows the credential file this command writes — unset it for the stored login to be read")
	}

	if !term.IsTerminal(stdinFd()) {
		return refuse(publish.Refusal{
			Headline:    "standard input is not a terminal",
			Consequence: "This command prompts for a username and secret; it takes no credential\nflags and cannot read them from a pipe.",
			Action:      fmt.Sprintf("For scripted use:  docker login %s", tgt.host),
		})
	}

	user, secret, err := promptCredential(tgt.host)
	if err != nil {
		return err
	}

	path, err := dockercfg.Path()
	if err != nil {
		return err
	}

	if err := login(c.Context(), path, tgt, user, secret); err != nil {
		if errors.Is(err, errBadCredential) {
			return refuse(publish.Refusal{
				Headline:    fmt.Sprintf("%s rejected the credential for %s", tgt.host, user),
				Consequence: "Nothing was written: a stored bad credential would move this failure to\nthe next publish, blamed on the wrong command.",
				Action:      fmt.Sprintf("Check the username and secret, then retry:  opm registry login %s", tgt.arg()),
			})
		}
		var connErr *publish.ConnectivityError
		if errors.As(err, &connErr) {
			return &opmexit.ExitError{Code: opmexit.ExitConnectivityError, Err: err}
		}
		return err
	}

	output.Info(output.FormatCheckmark(fmt.Sprintf("Credential for %s written to %s", tgt.host, path)))
	return nil
}

// login is the verify-then-write core: the /v2/ probe with the entered
// credential, then the single-entry upsert. Split from the prompts so the
// hermetic tests drive it directly.
func login(ctx context.Context, path string, tgt target, user, secret string) error {
	if err := probeCredential(ctx, tgt, user, secret); err != nil {
		return err
	}
	return dockercfg.Upsert(path, tgt.host, user, secret)
}

// parseHostArg takes the host literally, docker-login style; the one piece
// of syntax honored is the trailing +insecure suffix, selecting plain HTTP
// the same way the mapping syntax does.
func parseHostArg(arg string) target {
	host, insecure := strings.CutSuffix(arg, "+insecure")
	return target{host: host, insecure: insecure}
}

// reportResolution prints the resolution line for the no-arg form: the
// winning source and any value it shadows — ResolveRegistry.Shadowed's
// first consumer (D11: an override is reportable rather than silent).
func reportResolution(cfg *config.GlobalConfig) {
	res := cfg.RegistryResolution
	if res.Registry == "" {
		return
	}
	keyvals := []any{"registry", res.Registry, "source", string(res.Source)}
	if v, ok := res.Shadowed[config.SourceEnv]; ok {
		keyvals = append(keyvals, "shadows OPM_REGISTRY", v)
	}
	if v, ok := res.Shadowed[config.SourceConfig]; ok {
		keyvals = append(keyvals, "shadows config.registry", v)
	}
	output.Info("resolved registry", keyvals...)
}

// resolveTarget resolves the no-arg form: the public resolver over the
// configured mapping, then its host set. Exactly one host proceeds; picking
// one out of several silently would authenticate the user against a host
// they never named, so several refuse with each listed as a runnable
// command, and none refuses pointing at configuration.
func resolveTarget(registry string) (target, *publish.Refusal) {
	if registry == "" {
		return target{}, &publish.Refusal{
			Headline:    "no registry is configured",
			Consequence: "Without a configured registry there is no host to log in to.",
			Action:      "Configure one:  opm config init\nOr name the host directly:  opm registry login <host>",
		}
	}

	resolver, err := modconfig.NewResolver(&modconfig.Config{CUERegistry: registry})
	if err != nil {
		return target{}, &publish.Refusal{
			Headline: fmt.Sprintf("the configured registry %q does not parse", registry),
			Err:      err,
		}
	}

	hosts := resolver.AllHosts()
	switch len(hosts) {
	case 1:
		return target{host: hosts[0].Name, insecure: hosts[0].Insecure}, nil
	case 0:
		return target{}, &publish.Refusal{
			Headline:    fmt.Sprintf("the configured registry %q resolves to no host", registry),
			Consequence: "There is no host to log in to.",
			Action:      "Name the host directly:  opm registry login <host>",
		}
	default:
		evidence := make([][]string, 0, len(hosts))
		actions := make([]string, 0, len(hosts))
		for _, h := range hosts {
			t := target{host: h.Name, insecure: h.Insecure}
			scheme := "https"
			if h.Insecure {
				scheme = "http"
			}
			evidence = append(evidence, []string{h.Name, scheme})
			actions = append(actions, "opm registry login "+t.arg())
		}
		return target{}, &publish.Refusal{
			Headline:    fmt.Sprintf("the configured registry maps to %d hosts; name the one to log in to", len(hosts)),
			Evidence:    evidence,
			Consequence: "A credential is stored per host. Logging in to one of these silently\nwould authenticate you against a host you never named.",
			Action:      strings.Join(actions, "\n"),
		}
	}
}

// promptCredential asks for the username on stderr and reads the secret
// with terminal echo disabled.
func promptCredential(host string) (user, secret string, err error) {
	output.Prompt(fmt.Sprintf("Username for %s: ", host))
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", "", fmt.Errorf("reading username: %w", err)
	}
	user = strings.TrimSpace(line)
	if user == "" {
		return "", "", &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: errors.New("username must not be empty")}
	}

	output.Prompt("Secret (token or password): ")
	raw, err := term.ReadPassword(stdinFd())
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", "", fmt.Errorf("reading secret: %w", err)
	}
	secret = string(raw)
	if secret == "" {
		return "", "", &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: errors.New("secret must not be empty")}
	}
	return user, secret, nil
}

// staticAuth supplies the entered credential to ociauth's transport for
// exactly the probed host.
type staticAuth struct {
	host  string
	entry ociauth.ConfigEntry
}

func (s staticAuth) EntryForRegistry(host string) (ociauth.ConfigEntry, error) {
	if host != s.host {
		return ociauth.ConfigEntry{}, nil
	}
	return s.entry, nil
}

// probeCredential verifies the credential with the request the resolver's
// transport will make anyway: GET /v2/ through CUE's own auth transport, so
// basic-auth registries and bearer-challenge registries (GHCR) both verify
// for real. 401/403 mean the registry judged and rejected the credential;
// failing to get a verdict at all is connectivity, not a judgment.
func probeCredential(ctx context.Context, tgt target, user, secret string) error {
	scheme := "https"
	if tgt.insecure {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s/v2/", scheme, tgt.host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("building credential probe for %s: %w", tgt.host, err)
	}

	client := &http.Client{
		Transport: ociauth.NewStdTransport(ociauth.StdTransportParams{
			Config: staticAuth{host: tgt.host, entry: ociauth.ConfigEntry{Username: user, Password: secret}},
		}),
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		// A bearer-challenge token endpoint rejecting the credential surfaces
		// as an HTTP-status error from the auth transport, not a response.
		var herr ociregistry.HTTPError
		if errors.As(err, &herr) && rejectedStatus(herr.StatusCode()) {
			return fmt.Errorf("%s: %w", tgt.host, errBadCredential)
		}
		return &publish.ConnectivityError{Op: fmt.Sprintf("verifying credential against %s", url), Err: err}
	}
	defer resp.Body.Close()

	if rejectedStatus(resp.StatusCode) {
		return fmt.Errorf("%s answered HTTP %d: %w", tgt.host, resp.StatusCode, errBadCredential)
	}
	// Anything else — 200, or e.g. 404 from a registry that does not serve
	// /v2/ but accepted the auth — is a pass, mirroring the transport.
	return nil
}

// rejectedStatus reports whether an HTTP status is the registry rejecting
// the credential.
func rejectedStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

// refuse prints one refusal through the house funnel and exits 2.
func refuse(r publish.Refusal) error {
	cmdutil.PrintRefusals([]publish.Refusal{r})
	return &opmexit.ExitError{
		Code:    opmexit.ExitValidationError,
		Err:     errors.New(r.Headline),
		Printed: true,
	}
}
