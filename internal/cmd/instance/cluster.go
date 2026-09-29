package instance

import (
	"context"
	"time"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
)

// clusterLookupTimeout bounds the whole cluster Platform lookup of instance
// build and vet: an API server that does not answer costs at most this long
// before the render falls back to the instance's own deps.
const clusterLookupTimeout = 10 * time.Second

// clusterLookup carries the flags instance build and vet read the cluster
// Platform through.
type clusterLookup struct {
	k8s     cmdutil.K8sFlags
	offline bool
}

// offlineFlagHelp is the --offline help shared by instance build and vet.
const offlineFlagHelp = "Never contact a cluster; render against --platform or the instance's own deps"

// optionalClusterGetter returns the cluster Platform getter instance build
// and vet resolve through, or nil when the cluster step is skipped. Building
// it never fails the command:
//
//   - --offline (or --platform, which wins anyway) skips the cluster and
//     loads no kubeconfig;
//   - no resolvable kubeconfig context (an empty or missing kubeconfig) skips
//     it silently;
//   - a client that cannot be built otherwise warns and skips it.
//
// The returned getter bounds the lookup with clusterLookupTimeout; the
// resolver is told the cluster is optional, so any read failure warns and
// falls back to the deps.
func optionalClusterGetter(cfg *config.GlobalConfig, kf cmdutil.K8sFlags, platformFlag string, offline bool) platform.ClusterPlatformGetter {
	if offline || platformFlag != "" {
		return nil
	}
	k8sConfig, err := config.ResolveKubernetes(config.ResolveKubernetesOptions{
		Config:         cfg,
		KubeconfigFlag: kf.Kubeconfig,
		ContextFlag:    kf.Context,
	})
	if err != nil {
		output.Warn("cluster Platform not used (resolving kubeconfig: " + err.Error() + ")")
		return nil
	}
	client, err := cmdutil.NewK8sClient(k8sConfig, cfg.Log.Kubernetes.APIWarnings)
	if err != nil {
		if !cmdutil.IsNoKubeContext(err) {
			output.Warn("cluster Platform not used (" + err.Error() + ")")
		}
		return nil
	}
	read := platform.ClusterPlatformGetterFor(client.Dynamic)
	return func(ctx context.Context) (*platform.ClusterPlatform, string, error) {
		ctx, cancel := context.WithTimeout(ctx, clusterLookupTimeout)
		defer cancel()
		return read(ctx)
	}
}
