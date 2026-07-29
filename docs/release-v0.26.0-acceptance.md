# KPanel v0.26.0 验收记录

## 交付范围

- KPanel 交互式终端自动识别 `http://` 和 `https://` URL，点击后使用隔离的新浏览器标签页打开。
- 交互式终端增加外接预输入栏：用户可先完整输入或粘贴内容，再按 `Enter` 整行发送；原实时终端输入继续保留。
- 修复应用停止或状态刷新后详情弹窗不可见、页面滚动仍被锁定的问题。
- 修复 Release 重跑无法复用草稿版本的问题，并为 Docker Hub 摘要读取及 `latest` 提升增加有限重试。
- KPanel 继续固定使用
  `kejilion/sh@f031d1206224de3743845d2fc81c4801ecda32f4`，脚本 SHA-256 为
  `278526cee183cdc826c25e113a399fcac72484f8f2af2fd17a8f75a1cd6a40c1`。

## 发布提交

- 应用详情修复：`341befe453ad7d9a920306176f53223613289886`
- 终端 URL 与预输入栏：`e2b546eefc5f230395b6b930132539b9c507a825`
- 版本准备：`8efe1f1f4105561cc463b81de1c0ed37c9442857`
- 发布链路修复及最终标签提交：`0fcc4ca2161b030cc19de786c485893eaf4a7082`
- 标签：`v0.26.0`

## 自动与本地验收

- 主分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30428186476>
- Release：
  <https://github.com/kejilion/KPanel/actions/runs/30428273602>
- 本地通过 14 个前端测试文件、76 项 Vitest、类型检查和生产构建。
- Go 单元测试、`go vet`、Linux amd64/arm64 的 Panel、Agent、kpctl
  交叉编译全部通过。
- Shell 语法、安装安全测试、生态规则检查和 `npm audit --audit-level=high`
  通过，Node 高危漏洞为 0。
- GitHub CI 与 Release 均通过 `govulncheck v1.6.0`、
  `kejilion.sh` 应用生命周期测试及运行时镜像契约验证。
- 发布镜像以只读根文件系统、非 root、`cap-drop ALL` 和
  `no-new-privileges` 条件完成健康检查；许可证、版本、源码提交、
  固定脚本提交及摘要均通过复核。
- 首次发布在 Docker Hub 授权接口瞬时断连后停在 `latest` 提升阶段；
  第二次执行暴露草稿查询不幂等。两处链路问题修复后，完整发布流程重新执行并成功。

## 线上产物

- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.26.0>
- 生产镜像：
  `docker.io/kjlion/kejilion-panel@sha256:703a56b9ab8e5e6fd0a1e7ee21f16b060d647de150efe1d4ac18e3b836a9a9ed`
- `0.26.0` 与 `latest` 为同一 manifest digest，包含 linux/amd64、
  linux/arm64 及对应 SBOM/Provenance 证明清单。
- 发布附件已重新下载并按 `SHA256SUMS` 独立校验：
  - `kejilion-agent-linux-amd64`：
    `2c996d6b4134d8bbb4d4f9d8fd838dde1237863c0b0520fcddd34f23d452026e`
  - `kejilion-agent-linux-arm64`：
    `073164693957cc681fd1d360d28f63205c5d75c60f1d38200acbb6f188fcbb47`
  - `kejilion-panel-deploy-0.26.0.tar.gz`：
    `eb4316aba5b81a98c875843faebdee83cf61a54adbb20b3aa09d0d45118429cb`

## 回滚与边界

- 代码回滚点：标签 `v0.25.0`。
- 镜像回滚点：
  `docker.io/kjlion/kejilion-panel@sha256:303ba7e5820194c9f15d7b9fcb4fc7309d7c5e46e6f425155dc0f7fc2e175e49`。
- 本版本没有数据库格式迁移，也没有更新 `kejilion.sh`；回滚
  KPanel/Agent 不会删除网站、数据库、应用容器、域名配置或环境备份。
- 发布镜像已上线到 Docker Hub，但不会主动替换用户主机。用户从
  `kejilion.sh` 应用市场更新 KPanel 时仍执行原位升级和失败回滚。
