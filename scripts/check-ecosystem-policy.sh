#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

required_files=(
  "PROJECT_RULES.md"
  "docs/ecosystem-parity.md"
  "docs/operational-boundary-audit.md"
)

for path in "${required_files[@]}"; do
  if [[ ! -s "$path" ]]; then
    echo "Missing permanent ecosystem policy file: $path" >&2
    exit 1
  fi
done

production_pathspec=(
  "internal/**/*.go"
  "cmd/**/*.go"
  "web/src/**/*.ts"
  "web/src/**/*.vue"
  ":(exclude)**/*_test.go"
  ":(exclude)web/src/**/*.test.ts"
)

forbidden_patterns=(
  'Confirmation[[:space:]]*(==|!=)'
  'ConfirmDomain[[:space:]]*(==|!=)'
  'confirmation:[[:space:]]*['"'"'"](REBOOT|PRUNE|RESTORE|MIGRATE)['"'"'"]'
  'summary\.Ownership[[:space:]]*(==|!=)'
  'container\.Ownership[[:space:]]*(==|!=)'
  'managedNginxLabels'
  'unsafeNginxReason'
  'cUnsafeReason'
  'unsafeWordPressMySQLReason'
  'safeExistingWordPressContainer'
  'singleContainerIPv4'
  'ErrProtectedDockerResource'
  'ErrReadOnlyContainer'
  'ErrUnsafeOrInvalidAction'
  'Manual edits make it read-only'
)

failed=false
for pattern in "${forbidden_patterns[@]}"; do
  if matches="$(git grep -n -E "$pattern" -- "${production_pathspec[@]}" 2>/dev/null)"; then
    echo "Ecosystem policy regression matched /$pattern/:" >&2
    echo "$matches" >&2
    failed=true
  fi
done

if [[ "$failed" == true ]]; then
  echo "Resource origin, fixed confirmation words and KPanel self-protection cannot authorize business actions." >&2
  exit 1
fi

grep -Fq '`kejilion.sh` 是 KPanel 业务行为、资源布局和兼容方式的首要真源' PROJECT_RULES.md
grep -Fq '不得设置用户操作护栏' PROJECT_RULES.md
grep -Fq '必须保留的攻击面与完整性防护' PROJECT_RULES.md
grep -Fq '核验分级' PROJECT_RULES.md

echo "Ecosystem policy check passed."
