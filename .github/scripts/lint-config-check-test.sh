#!/usr/bin/env bash
# Scenario test of lint-config-check.sh: the pass case on a copy of this
# tree, then one defect per scenario, each of which the check must refuse
# with the named message. Offline; needs golangci-lint on PATH (the Lint job
# runs it after the action; locally: task lint:config:test).
#
# Usage: lint-config-check-test.sh
set -euo pipefail

repo=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
check=$repo/.github/scripts/lint-config-check.sh
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

failed=0
count=0

# fresh NAME: a copy of the files the check reads, in $work/NAME.
fresh() {
  local dir=$work/$1
  mkdir -p "$dir/.github/workflows" "$dir/.github/scripts"
  cp "$repo/.golangci.yml" "$repo/.golangci-lint-version" "$dir/"
  cp -r "$repo/.github/golangci-lint" "$dir/.github/"
  cp "$repo/.github/workflows/pr.yml" "$repo/.github/workflows/ci.yml" "$dir/.github/workflows/"
  echo "$dir"
}

# expect NAME STATUS PATTERN: the check on $work/NAME exits STATUS and its
# output holds PATTERN (a fixed string; empty for none).
expect() {
  local name=$1 status=$2 pattern=$3 out rc=0
  count=$((count + 1))
  out=$(bash "$check" --root "$work/$name" 2>&1) || rc=$?
  if [ "$rc" -ne "$status" ]; then
    echo "FAIL $name: exit $rc, want $status"; echo "$out"; failed=$((failed + 1)); return
  fi
  if [ -n "$pattern" ] && ! grep -Fq -- "$pattern" <<<"$out"; then
    echo "FAIL $name: output lacks '$pattern'"; echo "$out"; failed=$((failed + 1)); return
  fi
  echo "ok   $name"
}

d=$(fresh pass)
expect pass 0 "lint-config: ok"

# The step as it was before the check existed: the action names its own
# version and runs its own, downloading, configuration check.
d=$(fresh action-verifies)
sed -i '/^ *verify: false$/d' "$d/.github/workflows/pr.yml"
expect action-verifies 1 "pr.yml:"
expect action-verifies 1 "needs 'verify: false'"

d=$(fresh verify-true)
sed -i 's/^\( *\)verify: false$/\1verify: true/' "$d/.github/workflows/ci.yml"
expect verify-true 1 "ci.yml:"

d=$(fresh own-version)
sed -i 's/^\( *\)version-file: .*$/\1version: v2.11.4/' "$d/.github/workflows/ci.yml"
expect own-version 1 "sets 'version: v2.11.4'"
expect own-version 1 "needs 'version-file: .golangci-lint-version'"

d=$(fresh both-version-inputs)
sed -i 's/^\( *\)verify: false$/\1verify: false\n\1version: v2.10.0/' "$d/.github/workflows/pr.yml"
expect both-version-inputs 1 "sets 'version: v2.10.0'"

d=$(fresh split-pins)
sed -i 's|\(golangci/golangci-lint-action@\)[0-9a-f]\{40\}|\10123456789abcdef0123456789abcdef01234567|' "$d/.github/workflows/ci.yml"
expect split-pins 1 "pinned to more than one commit"

d=$(fresh tag-pin)
sed -i 's|\(golangci/golangci-lint-action@\)[0-9a-f]\{40\}|\1v9|' "$d/.github/workflows/pr.yml"
expect tag-pin 1 "is not pinned to a commit SHA"

d=$(fresh no-check-step)
sed -i '/run: bash \.github\/scripts\/lint-config-check\.sh$/d' "$d/.github/workflows/ci.yml"
expect no-check-step 1 "ci.yml uses golangci/golangci-lint-action but has no step"

d=$(fresh no-action)
sed -i '/golangci\/golangci-lint-action@/d' "$d/.github/workflows/pr.yml" "$d/.github/workflows/ci.yml"
expect no-action 1 "no workflow uses golangci/golangci-lint-action"

d=$(fresh minor-without-schema)
echo v2.12.0 >"$d/.golangci-lint-version"
expect minor-without-schema 1 "golangci.v2.12.jsonschema.json is missing"
expect minor-without-schema 1 "golangci.v2.11.jsonschema.json is not the schema of v2.12.0"

d=$(fresh bad-version)
echo latest >"$d/.golangci-lint-version"
expect bad-version 1 "want vX.Y.Z"

d=$(fresh no-version-file)
rm "$d/.golangci-lint-version"
expect no-version-file 1 ".golangci-lint-version is missing"

d=$(fresh edited-schema)
for f in "$d"/.github/golangci-lint/*.jsonschema.json; do echo >>"$f"; done
expect edited-schema 1 "does not match .github/golangci-lint/SHA256SUMS"

d=$(fresh no-checksum)
rm "$d/.github/golangci-lint/SHA256SUMS"
expect no-checksum 1 "SHA256SUMS is missing"

d=$(fresh stale-schema)
cp "$d"/.github/golangci-lint/golangci.v*.jsonschema.json "$d/.github/golangci-lint/golangci.v2.0.jsonschema.json"
expect stale-schema 1 "golangci.v2.0.jsonschema.json is not the schema of"

d=$(fresh invalid-config)
printf '\nnot-a-golangci-key: true\n' >>"$d/.golangci.yml"
expect invalid-config 1 "does not pass golangci-lint config verify"
expect invalid-config 1 "not-a-golangci-key"

count=$((count + 1))
if out=$(bash "$check" --no-such-flag 2>&1); then rc=0; else rc=$?; fi
if [ "$rc" -eq 2 ]; then echo "ok   usage"; else echo "FAIL usage: exit $rc, want 2: $out"; failed=$((failed + 1)); fi

if [ "$failed" -gt 0 ]; then
  echo "lint-config test: $failed of $count failed"
  exit 1
fi
echo "lint-config test: $count passed"
