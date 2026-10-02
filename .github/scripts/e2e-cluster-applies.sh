#!/usr/bin/env bash
# Decide whether the e2e-cluster workflow's job does its cluster-backed work.
#
# Usage: e2e-cluster-applies.sh <event-name> <head-ref> <labels-file> <files-file>
#
#   event-name   github.event_name (workflow_dispatch always applies)
#   head-ref     the pull request's head branch (empty for workflow_dispatch)
#   labels-file  one label name per line, read live from the pull request
#   files-file   one changed path per line, every page of the pull request's
#                file list (a renamed file contributes both its old and new path)
#
# Prints a report of what it checked and why it decided, and writes
# applies=true|false to $GITHUB_OUTPUT when that is set, else to stdout. The
# decision depends only on these inputs, so runs for the same head commit,
# head branch and labels always decide the same way: an unrelated label event
# can repeat a verdict, never overturn it.
set -euo pipefail

[ $# -eq 4 ] || { echo "usage: $0 <event-name> <head-ref> <labels-file> <files-file>" >&2; exit 2; }
event=$1 head_ref=$2 labels_file=$3 files_file=$4
[ -r "$labels_file" ] || { echo "e2e-cluster: cannot read labels file $labels_file" >&2; exit 2; }
[ -r "$files_file" ] || { echo "e2e-cluster: cannot read files file $files_file" >&2; exit 2; }

cascade_branch=deps/cascade
cascade_label=deps-cascade
release_prefix=release-please--

# Paths whose change makes the job apply: the operator embed and what it
# serves, plus the job's own inputs (a change to them is tested where it is
# made, not at the next release pull request). Extended regular expressions,
# matched against the whole repo-relative path.
apply_paths=(
  '^internal/operator/'
  '^internal/cmd/operator/'
  '^templates/'
  '^hack/platform/'
  '^\.github/workflows/e2e-cluster\.yml$'
  '^\.github/scripts/e2e-cluster-applies\.sh$'
  '^Taskfile\.yml$'
  '^hack/fixtures\.sh$'
  '^hack/kind-(config|platform|operator-rbac)\.yaml$'
  '^hack/opm-config\.cue$'
  '^tests/e2e/[^/]+\.go$'
  '^tests/e2e/testdata/operator-owned/'
)

labels=$(grep -v '^[[:space:]]*$' "$labels_file" || true)
files=$(grep -v '^[[:space:]]*$' "$files_file" || true)
file_count=$(printf '%s' "$files" | grep -c '' || true)

reasons=()
[ "$event" = workflow_dispatch ] && reasons+=("started by workflow_dispatch")
case "$head_ref" in "$release_prefix"*) reasons+=("release pull request (head branch $head_ref)") ;; esac
[ "$head_ref" = "$cascade_branch" ] && reasons+=("cascade branch $cascade_branch")
if printf '%s\n' "$labels" | grep -qx -- "$cascade_label"; then
  reasons+=("label $cascade_label")
fi
matched=$(printf '%s\n' "$files" | grep -E "$(IFS='|'; echo "${apply_paths[*]}")" || true)
if [ -n "$matched" ]; then
  reasons+=("changes $(printf '%s' "$matched" | grep -c '') listed path(s), first: $(printf '%s\n' "$matched" | head -1)")
fi

joined_labels=$(printf '%s' "$labels" | paste -sd, - | sed 's/,/, /g')
if [ ${#reasons[@]} -gt 0 ]; then
  applies=true
  echo "e2e-cluster: applies"
  echo "  head branch: ${head_ref:-<none>}"
  echo "  labels: ${joined_labels:-<none>}"
  for r in "${reasons[@]}"; do echo "  because: $r"; done
else
  applies=false
  echo "e2e-cluster: not applicable"
  echo "  head branch: ${head_ref:-<none>}"
  echo "  labels: ${joined_labels:-<none>}"
  echo "  changed files: $file_count, none under internal/operator/, internal/cmd/operator/, templates/, hack/platform/ or the job's own inputs"
  echo "  nothing to do; passing"
fi

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  echo "applies=$applies" >>"$GITHUB_OUTPUT"
else
  echo "applies=$applies"
fi
