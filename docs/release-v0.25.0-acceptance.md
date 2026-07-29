# KPanel v0.25.0 验收记录

## 交付范围

- 静态站、PHP 站、域名反代、负载均衡和重定向模板统一调用 `kejilion.sh`
  原生交互建站流程；面板只预填首个域名，其余参数由脚本终端继续收集。
- Bitwarden 和 Halo 补充为可选的一键建站模板，继续复用脚本现有业务实现。
- 建站任务使用独立后台 PTY；关闭弹窗、刷新页面或离开网站页不会中断任务，可从任务横幅
  重新进入终端。
- KPanel 固定使用 `kejilion/sh@f031d1206224de3743845d2fc81c4801ecda32f4`，
  GitHub Raw 字节 SHA-256 为
  `278526cee183cdc826c25e113a399fcac72484f8f2af2fd17a8f75a1cd6a40c1`。
- KPanel 源代码及发行物采用 `AGPL-3.0-only`；部署包和镜像继续携带第三方许可与品牌说明。

## 发布提交

- KPanel 功能提交：`0b3dfc605e5657945d34cc11209ec385b9f7384a`
- KPanel 发布提交：`e84fa8cc6e6b5c465ee5af8b9ef9a1bdb70077f3`
- 标签：`v0.25.0`
- `kejilion.sh`：`f031d1206224de3743845d2fc81c4801ecda32f4`

## 自动与本地验收

- 发布候选分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30424489297>
- 主分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30424626609>
- Release：
  <https://github.com/kejilion/KPanel/actions/runs/30424742325>
- 两轮 Linux CI 和 Release 均通过 Go 变更验证、`go vet`、`govulncheck v1.6.0`、
  Web 类型检查、71 项 Vitest、生产构建、`npm audit --audit-level=high`、安装安全检查及
  `kejilion.sh` 应用生命周期测试。
- 本地相关 Go 单元测试、Panel 路由测试、Linux amd64/arm64 的 Panel、Agent、kpctl
  交叉编译全部通过。
- `kejilion.sh` 通过语法检查和 9 组 smoke tests；由线上固定脚本重新生成的 115 个内置
  应用目录与仓库内容逐字节一致。
- 生态规则检查通过；发布镜像在只读根文件系统、非 root、`cap-drop ALL` 和
  `no-new-privileges` 条件下通过运行时健康检查，OCI 版本、源码、许可、脚本提交与摘要
  均通过工作流复核。

## 线上产物

- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.25.0>
- 生产镜像：
  `docker.io/kjlion/kejilion-panel@sha256:303ba7e5820194c9f15d7b9fcb4fc7309d7c5e46e6f425155dc0f7fc2e175e49`
- `0.25.0` 与 `latest` 为同一 manifest digest，包含 linux/amd64、linux/arm64 以及
  对应 SBOM/Provenance 证明清单。
- 发布附件已重新下载，并按附件中的 `SHA256SUMS` 独立校验：
  - `kejilion-agent-linux-amd64`：
    `5985797d1536a0f6443b323ae91894fa47067e984eff3ca90d5029e6fe13cfcb`
  - `kejilion-agent-linux-arm64`：
    `4830c531f81d92c66f0c9c11f9057218a8e2f4dfcd4ba54392a9e2fe32c4c291`
  - `kejilion-panel-deploy-0.25.0.tar.gz`：
    `b081151d30f7b60eedbf45617694bdc9ab23d483a4ccccc3c4ed4b96199112c8`

## 回滚与边界

- 代码回滚点：标签 `v0.24.4`。
- 镜像回滚点：
  `docker.io/kjlion/kejilion-panel@sha256:4de5e75c3fc43331de7a4f4aa8907c960a271cf38a62282bd5569ad9e5aaf395`。
- 本版本没有数据库格式迁移；回滚 KPanel/Agent 不会删除网站、数据库、应用容器、域名配置
  或环境备份。
- 发布不会自动替换用户生产主机。用户从 `kejilion.sh` 应用市场更新 KPanel 时继续执行
  原位升级，保留当前访问端口，并在失败时恢复原镜像、Agent、脚本和服务配置。
- Windows 本地环境未模拟 systemd 与 Linux 专属安装安全测试；这些路径已由两轮主机
  Linux CI 和 Release 工作流覆盖。
