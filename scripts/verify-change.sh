#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

requested_level="${VERIFY_LEVEL:-auto}"
base_ref="${VERIFY_BASE_REF:-${1:-}}"

if [[ -n "$base_ref" ]] && git cat-file -e "${base_ref}^{commit}" 2>/dev/null; then
  mapfile -t changed_files < <(
    {
      git diff --name-only --diff-filter=ACMRT "$base_ref" --
      git ls-files --others --exclude-standard
    } | sed '/^$/d' | sort -u
  )
elif [[ "${CI:-}" == "true" ]] && git rev-parse --verify HEAD^ >/dev/null 2>&1; then
  mapfile -t changed_files < <(
    git diff --name-only --diff-filter=ACMRT HEAD^ HEAD -- |
      sed '/^$/d' | sort -u
  )
elif git rev-parse --verify HEAD >/dev/null 2>&1; then
  mapfile -t changed_files < <(
    {
      git diff --name-only --diff-filter=ACMRT HEAD --
      git ls-files --others --exclude-standard
    } | sed '/^$/d' | sort -u
  )
else
  mapfile -t changed_files < <(
    {
      git ls-files
      git ls-files --others --exclude-standard
    } | sed '/^$/d' | sort -u
  )
fi

git diff --check
git diff --cached --check
if [[ -n "$base_ref" ]] && git cat-file -e "${base_ref}^{commit}" 2>/dev/null; then
  git diff --check "$base_ref" --
fi
ECOSYSTEM_POLICY_BASE_REF="$base_ref" bash scripts/check-ecosystem-policy.sh

if [[ ${#changed_files[@]} -eq 0 ]]; then
  echo "No changes require verification."
  exit 0
fi

printf 'Changed files (%d):\n' "${#changed_files[@]}"
printf '  %s\n' "${changed_files[@]}"

docs_only=true
needs_web=false
needs_go=false
needs_full_go=false
needs_deploy=false
needs_image=false
needs_shell_syntax=false
go_domains=()

for path in "${changed_files[@]}"; do
  case "$path" in
    *.md|docs/*|LICENSE)
      ;;
    web/*)
      docs_only=false
      needs_web=true
      ;;
    internal/dockerx/*)
      docs_only=false
      needs_go=true
      go_domains+=("dockerx")
      ;;
    internal/sites/*)
      docs_only=false
      needs_go=true
      go_domains+=("sites")
      ;;
    internal/appmarket/*)
      docs_only=false
      needs_go=true
      go_domains+=("appmarket")
      ;;
    internal/systemmanage/*)
      docs_only=false
      needs_go=true
      go_domains+=("systemmanage")
      ;;
    internal/agent/*|internal/panel/*|internal/contract/*|cmd/*|go.mod|go.sum)
      docs_only=false
      needs_go=true
      needs_full_go=true
      ;;
    *.go)
      docs_only=false
      needs_go=true
      ;;
    Dockerfile|docker-compose*.yml|deploy/*|VERSION)
      docs_only=false
      needs_deploy=true
      needs_shell_syntax=true
      if [[ "$path" == Dockerfile || "$path" == docker-compose*.yml || "$path" == VERSION ]]; then
        needs_image=true
      fi
      ;;
    .github/workflows/*)
      docs_only=false
      ;;
    *.sh|Makefile|scripts/*)
      docs_only=false
      needs_shell_syntax=true
      ;;
    *)
      docs_only=false
      needs_go=true
      needs_web=true
      needs_deploy=true
      ;;
  esac
done

if [[ ${#go_domains[@]} -gt 0 ]]; then
  mapfile -t go_domains < <(printf '%s\n' "${go_domains[@]}" | sort -u)
  if [[ ${#go_domains[@]} -gt 1 ]]; then
    needs_full_go=true
  fi
fi

if [[ "$requested_level" == "0" || "$requested_level" == "l0" || "$docs_only" == true ]]; then
  echo "L0 verification completed."
  exit 0
fi

install_web_dependencies() {
  if [[ "${CI:-}" == "true" || ! -d web/node_modules ]]; then
    (cd web && npm ci)
  fi
}

verify_web() {
  install_web_dependencies
  (
    cd web
    npm run typecheck
    npm test
    npm run build
  )
}

verify_targeted_go() {
  local -a packages=()
  local path directory
  for path in "${changed_files[@]}"; do
    [[ "$path" == *.go ]] || continue
    directory="${path%/*}"
    [[ "$directory" == "$path" ]] && directory="."
    packages+=("./$directory")
  done
  if [[ ${#packages[@]} -eq 0 ]]; then
    packages=("./...")
  else
    mapfile -t packages < <(printf '%s\n' "${packages[@]}" | sort -u)
  fi
  go test "${packages[@]}"
  go vet "${packages[@]}"
}

verify_deploy() {
  if command -v bash >/dev/null 2>&1; then
    while IFS= read -r script; do
      bash -n "$script"
    done < <(git ls-files '*.sh')
  fi
  sh deploy/tests/install-safety.sh
}

if [[ "$requested_level" == "3" || "$requested_level" == "l3" || "$requested_level" == "release" ]]; then
  make test
  go vet ./...
  make build-linux
  docker build --build-arg "VERSION=$(tr -d '\r\n' < VERSION)" -t kejilion-panel:verify .
  echo "L3 release verification completed."
  exit 0
fi

if [[ "$requested_level" == "2" || "$requested_level" == "l2" ]]; then
  needs_go=true
  needs_full_go=true
  needs_web=true
  needs_deploy=true
  needs_linux_build=true
else
  needs_linux_build=false
fi

if [[ "$needs_shell_syntax" == true ]]; then
  while IFS= read -r script; do
    bash -n "$script"
  done < <(git ls-files '*.sh')
fi

if [[ "$needs_go" == true ]]; then
  if [[ "$needs_full_go" == true ]]; then
    go test ./...
    go vet ./...
  else
    verify_targeted_go
  fi
fi

if [[ "$needs_web" == true ]]; then
  verify_web
fi

if [[ "$needs_deploy" == true ]]; then
  verify_deploy
fi

if [[ "$needs_linux_build" == true ]]; then
  make build-linux-binaries
fi

if [[ "$needs_image" == true ]] && command -v docker >/dev/null 2>&1; then
  docker build --build-arg "VERSION=$(tr -d '\r\n' < VERSION)" -t kejilion-panel:verify .
fi

echo "Change-aware verification completed."
