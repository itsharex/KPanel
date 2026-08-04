# KPanel v0.43.0 发布验收

## 发布范围

v0.43.0 优化 AI 助手会话体验与静默沉淀能力：新会话自动使用默认模型并根据首条消息命名，模型选择器移入输入框，工具结果不再作为普通消息泄露，运行完成不再闪回初始页；安全的重复工作可自动沉淀为跨会话记忆或流程，受保护操作仍需确认。

## 版本与制品

- 发布提交：`2b7874484cc80132ad0f7df5d63f6ec8beb87521`
- 标签：`v0.43.0`
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.43.0>
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30881374178>
- Docker 多架构摘要：`sha256:8da3c071341faebb8356148876486726e63b1b651c403ad5bd57086c53401574`
- linux/amd64：`sha256:0b6d8c2edf93a3c6489aac635ece2d0ecc584a04f6d33db897e9858c4d0a9a38`
- linux/arm64：`sha256:93967c3e441d44e16512ae14de81cfa9db522219191fbad8938fa7e4c3a98853`
- `0.43.0` 与 `latest` 指向同一多架构摘要。
- 公开应用市场 `kpanel.conf` 提交：`61f645dd7f129aff8b4663589e6bf3b65dc44860`
- 公开应用市场 `kpanel.conf` SHA-256：`b1e23371da402ebfcb61d56a377f77b6b78c44a454e23b99119bab5aaf895d0f`

## 自动化与 L3 验收

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30881005899>，成功。
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30881205829>，成功。
- 154 上使用提交 `2b78744` 的独立归档执行 L3，未使用原脏工作树。
- 前端 i18n、typecheck、生产构建成功；Vitest `38` 个文件、`245` 项测试通过。
- Go 全量测试、核心特权包 `-race`、`go vet`、`govulncheck` 通过；调用路径已知漏洞为 `0`。
- `npm audit --audit-level=high` 为 `0`；Trivy 源码、依赖、Secret、配置和最终镜像 HIGH/CRITICAL 扫描均为 `0`。
- 应用市场安装、更新、失败回滚和卸载生命周期通过。
- amd64/arm64 共 8 个 CGO-free、静态、stripped 二进制构建成功；公开不可变镜像再次通过隔离 `image_e2e`。
- 全局最多两个并行 Run、单会话排队、审批恢复、重启中断、流开始后不重放、静默学习去重/低置信度/受保护流程拒绝均包含在通过的 Go 测试中。

## 轻量验收

- stripped `paneld`：v0.42.1 为 `12,955,810` B，v0.43.0 为 `12,972,194` B，仅增加 `16,384` B，低于 30 MiB 目标。
- 154 暖机进程 RSS：v0.43.0 为 `19,472` KiB；Docker 统计为 `12.9 MiB / 256 MiB`，未突破 Compose 256 MiB 限制。
- 升级前同机 v0.42.1 进程 RSS 为 `22,080` KiB，v0.43.0 未增加。

## 154 上线结果

- 使用公开应用市场 `docker_app_update` 完成 v0.42.1 → v0.43.0 更新，输出 `KPanel 更新完成 / Update Complete`。
- Agent：`0.43.0 v1alpha1`，服务为 `active`、`NRestarts=0`。
- Panel：`version=0.43.0`、`status=ok`、容器 `healthy`；镜像 ID 为正式多架构摘要。
- `/` 与 `/ai` 返回 `200`；未登录 `/api/v1/ai/providers` 返回 `401`。
- 域名 HTTPS 转发路径返回 `200`，无效 Host 返回 `421`，确认 v0.42.1 的 Host 校验修复未回归。
- Panel 保持非特权用户 `65532:65532`、只读根文件系统、`privileged=false`、256 MiB、128 PIDs、双网络和可信代理最小网关 `/32`。
- AI SQLite `integrity_check=ok`；`ai.db` 与 `ai-secrets.key` 权限均为 `0600`。验收机未配置 Provider，不调用真实付费 API。
- 浏览器确认线上资源和登录页加载正常且无前端控制台错误；登录后的真实 Provider 对话由管理员后续配置，发布验收使用本地 mock Provider 和 stub Agent。

## 回滚

- 公共源码与镜像回滚点：`v0.42.1`，镜像摘要 `sha256:8269a8a25693814b50ec9ca5b38d12fb60550109e91771da9d384bea99db5eae`。
- 154 升级前一致性备份：`/root/kpanel-backups/v0.43.0-preupgrade-20260804T054702Z/kpanel-home.tar.gz`。
- 备份权限 `0600`，大小 `20,564,321` B，SHA-256：`da87c3e375023ab8b65179ed742e573c5f95b82543e4f62ca8d5375e17d4c6c4`。
- 回滚时恢复旧镜像、Agent、Compose、systemd 单元与完整 KPanel 目录；`ai.db*` 必须与 `ai-secrets.key` 成对保留。

## 已知边界

- 静默学习只在可复用任务满足门槛时额外调用一次短模型评估；单次只读查询不会触发。
- 154 验收机没有 Provider 或登录浏览器会话，因此没有发送真实 API Key，也没有执行真实付费模型调用。
- 原 `feature/health-center` 脏工作树保持在 `c9a3419`，本次发布未 stash、reset、提交或修改其文件。
