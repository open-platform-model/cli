---
title: "Install the CLI"
description: "Download the opm CLI for Linux, macOS or Windows, verify it and put it on your PATH."
type: how-to
weight: 15
---

The `opm` CLI ships as a prebuilt binary on each [GitHub release](https://github.com/open-platform-model/cli/releases) of `open-platform-model/cli`. This page installs [v1.0.0-beta.5](https://github.com/open-platform-model/cli/releases/tag/v1.0.0-beta.5) on Linux or macOS, then covers Windows, building from source and shell completion.

> [!WARNING]
> **GitHub's Latest release is not the current CLI**
>
> GitHub's Latest label points at the old v0.6.0, which has no binaries, and every v1.0.0 beta is marked Pre-release. A `releases/latest/download/...` URL returns 404. Use a tagged URL, as on this page. Newer releases appear at the top of the [releases page](https://github.com/open-platform-model/cli/releases) with the Pre-release label.

## Before you begin

- A shell with `curl`, `tar` and `sha256sum` (Linux) or `shasum` (macOS).
- Your operating system and CPU architecture. Each release carries one archive per pair:

  | OS      | Architecture                | Archive                    |
  | ------- | --------------------------- | -------------------------- |
  | Linux   | x86-64 (`amd64`)            | `opm-linux-amd64.tar.gz`   |
  | Linux   | ARM64 (`arm64`)             | `opm-linux-arm64.tar.gz`   |
  | macOS   | Intel (`amd64`)             | `opm-darwin-amd64.tar.gz`  |
  | macOS   | Apple silicon (`arm64`)     | `opm-darwin-arm64.tar.gz`  |
  | Windows | x86-64 (`amd64`)            | `opm-windows-amd64.tar.gz` |

  There is no Windows ARM64 archive.
- On Linux x86-64, a glibc-based distribution. The `linux-amd64` binary links against glibc and does not start on musl-based systems such as Alpine; [build from source](/docs/start/install-the-cli/#build-from-source) there instead.

The CLI is not published to Homebrew or any other package manager.

## 1. Download the release

Set the version and your system, then download the archive and the release's checksum file:

```sh
OPM_VERSION=v1.0.0-beta.5
OS=linux      # linux or darwin
ARCH=amd64    # amd64 or arm64
curl -fsSLO "https://github.com/open-platform-model/cli/releases/download/${OPM_VERSION}/opm-${OS}-${ARCH}.tar.gz"
curl -fsSLO "https://github.com/open-platform-model/cli/releases/download/${OPM_VERSION}/checksums.txt"
```

## 2. Verify the download

Check the archive against its SHA-256 sum in `checksums.txt`. On Linux:

```sh
grep " opm-${OS}-${ARCH}.tar.gz$" checksums.txt | sha256sum --check
```

On macOS:

```sh
grep " opm-${OS}-${ARCH}.tar.gz$" checksums.txt | shasum -a 256 --check
```

The output should look similar to this:

```text
opm-linux-amd64.tar.gz: OK
```

Any other result means the download is damaged or is not the released file. Delete it and download again.

> [!TIP]
> **Check the release attestation**
>
> CLI releases are immutable, and GitHub attests the assets of each one. With the [GitHub CLI](https://cli.github.com/) you can check that the archive is the asset the release published:
>
> ```sh
> gh release verify-asset "${OPM_VERSION}" "opm-${OS}-${ARCH}.tar.gz" -R open-platform-model/cli
> ```
>
> ```text
> Calculated digest for opm-linux-amd64.tar.gz: sha256:1debc482cf1a432f7183993dda637b5c0a6af123e1089644574b4b9efee11ec1
> Resolved tag v1.0.0-beta.5 to sha1:7c10dfb7d4c01cce3029f92596888c0474da77d4
> Loaded attestation from GitHub API
>
> ✓ Verification succeeded! opm-linux-amd64.tar.gz is present in release v1.0.0-beta.5
> ```
>
> The releases carry no other signature or build provenance.

## 3. Put opm on your PATH

Extract the binary and install it into a directory on your `PATH`:

```sh
tar -xzf "opm-${OS}-${ARCH}.tar.gz" opm
sudo install -m 0755 opm /usr/local/bin/opm
```

To install without `sudo`, use a directory you own that is on your `PATH`, such as `~/.local/bin`:

```sh
install -m 0755 opm ~/.local/bin/opm
```

> [!NOTE]
> **macOS and browser downloads**
>
> The macOS binaries are not signed or notarized by Apple. A file downloaded with a browser is quarantined, and macOS refuses to run it. Download with `curl` as in step 1, or clear the quarantine flag on the extracted binary with `xattr -d com.apple.quarantine opm`.

## 4. Check that it worked

```sh
opm version
```

The output should look similar to this:

```text
opm version 1.0.0-beta.5 (7c10dfb7d4c01cce3029f92596888c0474da77d4) built 2026-10-01T20:02:54Z with go1.26.0
CUE SDK: v0.17.1
```

The first line names the release, the commit it was built from, the build time and the Go version. The second names the CUE SDK built into the CLI. If your shell reports `opm: command not found`, the install directory from step 3 is not on your `PATH`.

## Install on Windows

The Windows archive holds `opm.exe`. In PowerShell, download it and the checksum file, and compare the sums:

```powershell
$Version = "v1.0.0-beta.5"
$Base = "https://github.com/open-platform-model/cli/releases/download/$Version"
Invoke-WebRequest "$Base/opm-windows-amd64.tar.gz" -OutFile opm-windows-amd64.tar.gz
Invoke-WebRequest "$Base/checksums.txt" -OutFile checksums.txt
$Expected = (Select-String -Path checksums.txt -Pattern " opm-windows-amd64.tar.gz$").Line.Split(" ")[0]
$Actual = (Get-FileHash opm-windows-amd64.tar.gz -Algorithm SHA256).Hash.ToLower()
if ($Actual -eq $Expected) { "OK" } else { "MISMATCH" }
```

On `OK`, extract the archive with the `tar` that ships with Windows, move `opm.exe` into a folder on your `Path`, and run `opm version` in a new terminal:

```powershell
tar -xzf opm-windows-amd64.tar.gz
```

## Build from source

With Go 1.26 or newer, `go install` builds the same release from its tag:

```sh
go install github.com/open-platform-model/cli/cmd/opm@v1.0.0-beta.5
```

The binary lands in `$(go env GOPATH)/bin`, or in `$GOBIN` when it is set. Put that directory on your `PATH`. A binary built this way reports `opm version dev (unknown) built unknown`, because only the release build stamps the version in.

## Turn on shell completion

`opm completion` prints a completion script for `bash`, `zsh`, `fish` or `powershell`. To load it in the current shell, run the line for your shell.

Bash, with the `bash-completion` package installed:

```bash
source <(opm completion bash)
```

Zsh, with `compinit` loaded:

```zsh
source <(opm completion zsh)
```

Fish:

```fish
opm completion fish | source
```

To load completion in every new shell, `opm completion <shell> --help` prints where to write the script for that shell.

## Next steps

- [Quickstart](/docs/start/quickstart/), which starts with `opm config init` to configure the CLI
- [Install the operator](/docs/start/install-the-operator/)

<!-- Tested on 2026-10-03 against the v1.0.0-beta.5 release (Pre-release, immutable; GitHub's Latest resolves to v0.6.0, which has no assets, so releases/latest/download/opm-linux-amd64.tar.gz returns 404). Linux amd64: steps 1 to 4 run as written on Fedora (install into a scratch directory instead of /usr/local/bin), the checksum and `gh release verify-asset` outputs are from that run; the checksum line for darwin-arm64 and windows-amd64 also verified with sha256sum. The linux-amd64 binary is dynamically linked against glibc and fails to start on alpine:3 (missing dynamic library); linux-arm64 is statically linked. The PowerShell block ran in the mcr.microsoft.com/powershell container on Linux (prints OK; tar extracts opm.exe), not on Windows. macOS steps, the shasum line and the Gatekeeper note were not run on a Mac; .goreleaser.yml has no signing or notarization. `go install ...@v1.0.0-beta.5` built with go1.26.5 and printed `opm version dev (unknown) built unknown with go1.26.5`; the same install in golang:1.26-alpine builds and runs. bash, zsh (with compinit) and fish completion load (bash `complete -p opm` shows `__start_opm`, zsh `_comps[opm]` is `_opm`). Re-verified on 2026-10-03 at v1.0.0-beta.5: Linux amd64 steps 1 to 4 (checksum, `gh release verify-asset`, `opm version`) and `go install` ran and the outputs above come from that run; the windows archive holds `opm.exe`. The Alpine, PowerShell, macOS and completion checks are from the beta.4 run and were not repeated. Re-run every step and update the outputs when the named release changes. -->
