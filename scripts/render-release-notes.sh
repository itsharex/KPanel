#!/usr/bin/env bash

set -euo pipefail

if (( $# != 4 )); then
  echo "usage: $0 <version> <image> <digest> <output>" >&2
  exit 2
fi

version="$1"
image="$2"
digest="$3"
output="$4"
changelog="${CHANGELOG_FILE:-CHANGELOG.md}"

[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?$ ]] || {
  echo "release notes version must be an exact semantic version" >&2
  exit 1
}
test -f "$changelog" || {
  echo "release notes changelog not found: $changelog" >&2
  exit 1
}

section="$({
  awk -v version="$version" '
    $0 == "## [" version "]" || index($0, "## [" version "] - ") == 1 {
      found = 1
      next
    }
    found && /^## \[/ { exit }
    found && !started && /^[[:space:]]*$/ { next }
    found {
      started = 1
      if ($0 == "### Added") print "#### 新增"
      else if ($0 == "### Changed") print "#### 变更"
      else if ($0 == "### Fixed") print "#### 修复"
      else if ($0 == "### Security") print "#### 安全"
      else if ($0 == "### Performance") print "#### 性能"
      else if ($0 == "### Documentation") print "#### 文档"
      else if ($0 == "### Upgrade Notes") print "#### 升级注意事项"
      else if ($0 == "### Deprecated") print "#### 弃用"
      else if ($0 == "### Removed") print "#### 移除"
      else if ($0 ~ /^### /) {
        sub(/^### /, "#### ")
        print
      } else print
    }
    END { if (!found) exit 3 }
  ' "$changelog"
} || true)"

test -n "${section//[[:space:]]/}" || {
  echo "CHANGELOG.md is missing the [$version] release section" >&2
  exit 1
}
grep -Eq '^- ' <<<"$section" || {
  echo "CHANGELOG.md [$version] must contain at least one explicit update item" >&2
  exit 1
}

mkdir -p "$(dirname "$output")"
{
  echo "## KPanel $version"
  echo
  echo "### 版本更新内容"
  echo
  printf '%s\n' "$section"
  echo
  echo "### 升级说明"
  echo
  echo "- 已安装用户可继续通过 kejilion.sh 应用市场的 KPanel 更新入口原位升级。"
  echo "- 默认兼容升级并保留现有数据目录、端口与反向代理配置；如上方列有升级注意事项，以该版本的明确说明为准。"
  echo "- 更新前建议记录当前版本与镜像摘要；失败时按原应用市场回滚流程恢复上一镜像。"
  echo
  echo "### 发布产物与完整性"
  echo
  echo "- 生产镜像：\`$image@$digest\`"
  echo "- Agent、轻量节点和部署归档须使用附件 \`SHA256SUMS\` 校验后再安装。"
  echo "- Release 流水线已经执行测试、漏洞扫描、双架构构建、镜像运行契约及摘要一致性检查。"
  echo
  echo "完整版本记录：[CHANGELOG.md](https://github.com/kejilion/KPanel/blob/v$version/CHANGELOG.md)"
} > "$output"
