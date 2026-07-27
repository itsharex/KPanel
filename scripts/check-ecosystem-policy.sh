#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

required_files=(
  "PROJECT_RULES.md"
  "docs/ecosystem-parity.md"
  "docs/external-config-sources.md"
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
grep -Fq '`kejilion.sh` 已有的外联配置必须直接复用，禁止 KPanel 自行编写替代模板' PROJECT_RULES.md
grep -Fq '没有稳定脚本接口时登记为适配缺口，不得先做一套 KPanel 专属实现' docs/ecosystem-parity.md
grep -Fq '<!-- external-config-debt:website-nginx:blocked -->' docs/external-config-sources.md
grep -Fq '<!-- external-config-debt:wordpress-flow:blocked -->' docs/external-config-sources.md

policy_base="${ECOSYSTEM_POLICY_BASE_REF:-}"
if [[ -z "$policy_base" ]]; then
  if [[ "${CI:-}" == "true" ]] && git rev-parse --verify HEAD^ >/dev/null 2>&1; then
    policy_base="HEAD^"
  else
    policy_base="HEAD"
  fi
fi
if ! git cat-file -e "${policy_base}^{commit}" 2>/dev/null; then
  policy_base="HEAD"
fi
mapfile -t changed_policy_files < <(
  {
    git diff --name-only --diff-filter=ACMRT "$policy_base" --
    git ls-files --others --exclude-standard
  } | sed '/^$/d' | sort -u
)

for path in "${changed_policy_files[@]}"; do
  case "$path" in
    internal/sites/managed_template.go)
      if grep -Fq '<!-- external-config-debt:website-nginx:blocked -->' docs/external-config-sources.md; then
        echo "Blocked external-config debt changed: $path" >&2
        echo "Migrate website Nginx generation to the kejilion.sh source before modifying this file." >&2
        exit 1
      fi
      ;;
    internal/sites/wordpress.go|internal/dockerx/wordpress.go)
      if grep -Fq '<!-- external-config-debt:wordpress-flow:blocked -->' docs/external-config-sources.md; then
        echo "Blocked external-config debt changed: $path" >&2
        echo "Migrate WordPress configuration to the kejilion.sh source before modifying this file." >&2
        exit 1
      fi
      ;;
  esac
done

echo "Ecosystem policy check passed."
