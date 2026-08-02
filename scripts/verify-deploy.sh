#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if command -v docker >/dev/null 2>&1; then
  docker run --rm \
    -v "$repo_root:/src:ro" \
    bash:5.2.37-alpine3.22@sha256:3bee76a96d86d5d2d5efc7c1c570e5a7c95db22348a26944e0e546fa174e3324 \
    sh /src/deploy/tests/install-safety.sh
else
  sh "$repo_root/deploy/tests/install-safety.sh"
fi
