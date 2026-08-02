#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

version="$(tr -d '\r\n' < VERSION)"
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] || {
  echo "VERSION is not a supported semantic version: $version" >&2
  exit 1
}

go_default="$(
  sed -n 's/^var Version = "\(.*\)-dev"$/\1/p' internal/version/version.go
)"
web_version="$(node -p "require('./web/package.json').version")"
lock_version="$(node -p "require('./web/package-lock.json').version")"
lock_root_version="$(node -p "require('./web/package-lock.json').packages[''].version")"

for entry in \
  "internal/version/version.go:$go_default" \
  "web/package.json:$web_version" \
  "web/package-lock.json:$lock_version" \
  "web/package-lock.json root package:$lock_root_version"; do
  location="${entry%%:*}"
  actual="${entry#*:}"
  [[ "$actual" == "$version" ]] || {
    echo "$location version $actual does not match VERSION $version" >&2
    exit 1
  }
done

echo "Version metadata is consistent: $version"
