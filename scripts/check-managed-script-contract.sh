#!/usr/bin/env bash
set -euo pipefail

repo_root="${KPANEL_REPO_ROOT:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
cd "$repo_root"

mapfile -t script_revisions < <(
  grep -oE 'io\.kejilion\.script\.revision="[0-9a-f]{40}"' Dockerfile |
    cut -d '"' -f 2
)
mapfile -t script_digests < <(
  grep -oE 'io\.kejilion\.script\.sha256="[0-9a-f]{64}"' Dockerfile |
    cut -d '"' -f 2
)

if [[ ${#script_revisions[@]} -ne 1 || ${#script_digests[@]} -ne 1 ]]; then
  echo "Dockerfile must declare exactly one managed script revision and SHA-256 label." >&2
  exit 1
fi

script_revision="${script_revisions[0]}"
script_sha256="${script_digests[0]}"

grep -Fq "ADD --checksum=sha256:${script_sha256} \\" Dockerfile
grep -Fq "https://raw.githubusercontent.com/kejilion/sh/${script_revision}/kejilion.sh \\" Dockerfile
grep -Fq "kejilion/sh@${script_revision}" docs/external-config-sources.md
grep -Fq "SHA-256 \`${script_sha256}\`" docs/external-config-sources.md

image="${1:-}"
if [[ -n "$image" ]]; then
  image_revision="$(docker image inspect --format \
    '{{index .Config.Labels "io.kejilion.script.revision"}}' "$image")"
  image_sha256="$(docker image inspect --format \
    '{{index .Config.Labels "io.kejilion.script.sha256"}}' "$image")"
  [[ "$image_revision" == "$script_revision" ]]
  [[ "$image_sha256" == "$script_sha256" ]]

  temporary="$(mktemp -d /tmp/kpanel-managed-script-contract.XXXXXX)"
  container="kpanel-managed-script-contract-$$-$RANDOM"
  cleanup() {
    docker rm -f "$container" >/dev/null 2>&1 || true
    case "$temporary" in
      /tmp/kpanel-managed-script-contract.*) rm -rf -- "$temporary" ;;
    esac
  }
  trap cleanup EXIT

  docker create --name "$container" "$image" >/dev/null
  docker cp "$container:/release/kejilion.sh" "$temporary/kejilion.sh"
  actual_sha256="$(sha256sum "$temporary/kejilion.sh" | awk '{print $1}')"
  [[ "$actual_sha256" == "$script_sha256" ]]
  cleanup
  trap - EXIT
fi

printf 'managed_script_contract=pass revision=%s sha256=%s\n' \
  "$script_revision" "$script_sha256"
