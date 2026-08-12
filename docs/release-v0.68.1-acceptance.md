# KPanel v0.68.1 发布验收记录

日期：2026-08-13

发布级别：L3 / Browser Reader 补丁

正式提交 / 标签：`9641d7cab152f664648f65744440c0127a1f27bf` / `v0.68.1`

功能提交：`3e48c6c`，来源冻结提交 `773eef83462791797e453395ade89a01d50af890`

上一稳定版本 / 回滚点：`v0.68.0` / `07b424cd46e0f43ed2ab6e65289b3608334142c6`

## 根因与修复范围

- v0.68.0 的应用市场契约将新安装和未标记升级默认写为 `disabled`，但桌面浏览器入口仍然可见，导致创建浏览器会话时返回 `browser_beta_disabled`。
- 本补丁新增安全 `reader` 模式作为新安装默认；保留显式 `beta` 模式以及 `disabled` kill switch，不把需要双 HTTPS Origin 和 Secure Cookie 的 Beta 在普通 HTTP/IP 环境中强制开启。
- Reader 仅允许 Relay 发起 `GET`/`HEAD`，使用短期 reader scope token；不执行目标页面脚本、表单和下载。Panel 从固定内网 Relay 获取有界内容，经私有 `MessageChannel` 传入 `sandbox="allow-scripts"` 且无 `allow-same-origin` 的 iframe，Reader CSP 禁止联网。
- 补充 Azure WireServer `168.63.129.16/32` SSRF 阻断及回归测试。
- 新安装写入 `reader` 与 `reader-v1` marker；尚无 marker 的 `disabled` 安装仅在交互确认或一次性 `KPANEL_BROWSER_READER_MIGRATION=reader` 时迁移；写入 marker 后，后续更新保留管理员选择。

## 能力边界

- Reader 用于安全阅读普通 HTML；复杂 SPA、登录、脚本挑战、媒体、下载等功能仍应使用“用系统浏览器打开”。
- `beta` 仍是需要双 HTTPS Origin 的实验模式；本补丁不降低 TLS、Origin、Secure Cookie、SSRF、DNS 重绑定、Token、CSP、大小、并发和资源限制。
- 本版本不宣称 Reader 或 Browser Beta 等同于完整 Chromium 浏览器。

## 候选冻结与 L3 门禁

- 候选基线为 `e483be9492621e17bda158e1a2a4e2b1da2cd646`；功能与版本提交之后形成正式提交 `9641d7c`，工作树 clean，未从分叉 `main` 或其他脏工作树发布。
- 完整 bundle：`C:\GitHub\_release-artifacts\v0.68.1-reader\kpanel-v0.68.1-9641d7c-full.bundle`，SHA-256 `0584b40e351d8ae18194558b8f52f5987b3189c8e8d79cf7b09790b6a4a66c53`。
- 154 L3 全量通过：Go 全量测试、`go vet`、核心包 race；Web 90 个文件 / 646 项测试；i18n 2144 条；typecheck、生产构建；Linux amd64/arm64 全部二进制；正式 Dockerfile；managed script contract；应用安装/更新/卸载 lifecycle。
- `govulncheck` 可达漏洞 0；`npm audit` 0；Trivy 源码、镜像、Dockerfile 的已知漏洞、secret 与 misconfiguration 门禁均通过。
- Reader 真实 Chromium harness 验证：opaque sandbox 与私有 MessageChannel 正常；恶意脚本 0 执行、表单 0 执行、目标图片 0 直连、父页面不可读取 iframe 文档、链接仅回传受控导航消息。
- L3 日志：`C:\GitHub\_release-artifacts\v0.68.1-reader\l3-154.log`，SHA-256 `5f5170a897d3a8e3fd9918478cfb7240d54190ec89e1a200969c9cd5bba14b49`。
- 候选 CI `31631157033`、候选依赖新鲜度 `31631157066`、主线 CI `31631465200`、主线依赖新鲜度 `31631465232`、Release workflow `31631824747` 全部成功。

## Release、OCI 与 apps

- [GitHub Release v0.68.1](https://github.com/kejilion/KPanel/releases/tag/v0.68.1) 已公开，非 draft、非 prerelease；Tag 精确指向正式提交，未改写历史。
- `docker.io/kjlion/kejilion-panel:0.68.1` 与 `latest` 均指向 OCI index `sha256:4ef9c602f04377638098235125b9c1c524aad6a29586a90c22975037815d2941`。
- `linux/amd64` manifest 为 `sha256:dd8edf3e2689977de94b6085a07826d5480e63c321315294aafc3cf235345b70`；`linux/arm64` manifest 为 `sha256:60c1b3d5bb05fc17bf6c11089f55457b063667bdf6f8c919a143e34e0e71829f`。两架构 revision/version 均精确为正式提交和 `0.68.1`。
- 从 Docker Hub 重新拉取公开镜像后的 154 E2E 输出 `image_e2e=pass`。
- apps 正式提交为 `08fc6a5d626f4af1c7537013be5b294fbcf4cb5a`；`kpanel.conf` 与发布源码 blob 一致，SHA-256 `8c287b5e197e89067d7a9cc62c189386889becdf86a76209e91912204882346c`，语法与生命周期门禁通过。

## 生产备份与回滚演练

- 154 停写一致性备份：`/root/kpanel-backups/v0.68.1-preupgrade-arena154-20260812T192418Z`；归档 SHA-256 `8cc77d25275d6e8bd758dd0940bc3069f9ca6b6c7a81ce586f1c43daf308f479`；旧镜像归档 SHA-256 `ca71aafbe12485e1e9435e9967ed31686482c6e44236bff7a78d6103f2fb639c`。
- 108 停写一致性备份：`/root/kpanel-backups/v0.68.1-preupgrade-prod108-20260812T192438Z`；归档 SHA-256 `006397b58db65a06b5ce52248728602204ca1a9baf2c5ee1b14cc73a8eb5f916`；旧镜像归档 SHA-256 `91c4f04eeaa2f25fb1e569344c5c28736d975ca5a6816676c07781e1b20f644a`。
- 两台备份均完成独立解包、文件摘要、SQLite/JSON/JSONL、权限及数据完整性恢复校验。
- 两台均在独立端口、独立容器和独立网络中完成 v0.68.0 成套回滚演练：同时恢复旧镜像、Compose、`.env`、数据与密钥组；154 恢复 `beta`，108 恢复 `disabled`；Panel/Relay healthy、restart 0、OOM false，未替换生产容器或生产数据。
- 首次 154 演练发现脚本未恢复 supplemental secret group，隔离实例因此未启动；生产未受影响。脚本补齐密钥组恢复与 trap 后，两台演练重新执行并通过。
- 完整回滚必须成套恢复匹配版本的镜像、Compose、`.env` 和数据；不得只换镜像或只改 mode，因为 v0.68.0 不识别 `reader`，更旧 Relay 也不支持当前参数契约。

## 生产升级结果

### arena-154

- 使用应用市场标准更新入口升级至 `0.68.1`，保留既有 `KPANEL_BROWSER_MODE=beta`、`reader-v1` marker、双 HTTPS Origin 和 `KPANEL_SECURE_COOKIE=true`。
- Panel 与 Relay healthy、restart 0、OOM false；Agent active、restart 0；镜像 revision/version 精确匹配正式提交与 `0.68.1`。
- Panel：`https://kpanel.154.36.153.9.sslip.io`；Relay：`https://kpanel-browser.154.36.153.9.sslip.io`。未配置、接管或使用 `kp.kejilion.pro`。

### prod-108

- 服务器既有 `/root/apps` 含与本轮无关的本地主线提交，发布过程未覆盖它；从 apps 精确提交 `08fc6a5` 建立隔离应用目录并调用标准更新入口。
- 升级时显式设置一次性 `KPANEL_BROWSER_READER_MIGRATION=reader`；升级后 Panel 与 Relay 的 mode 均为 `reader`，marker 为 `reader-v1`。
- 108 保持既有 HTTP Reader 部署和 `KPANEL_SECURE_COOKIE=false`，没有伪造 HTTPS，也没有强开 Beta。
- Reader 响应为 `Cache-Control: no-store`；CSP 包含 `sandbox allow-scripts`、`default-src 'none'`、`connect-src 'none'` 和 `frame-ancestors 'self'`。Relay 未鉴权访问被拒绝，Beta `/kernel/` 在 Reader 模式返回 404。
- 由于该实例启用了 Security Entrance，未携带入口凭据的受保护 Panel API 统一返回 404；这是既有隐藏策略，不是 Reader 路由缺失。生产未创建测试用户、未导出凭据，真实内容链路由候选/L3 harness 覆盖。
- Panel 与 Relay healthy、restart 0、OOM false；Agent active、restart 0；镜像 revision/version 精确匹配正式提交与 `0.68.1`。

两台 Agent 均报告 `0.68.1 v1alpha1`；Agent SHA-256 均为 `46a2fdfa9a9ed92da1f1414e68272c2afbe8b7a1ae1dd71481cfb9cfc261ee96`，托管 `kejilion.sh` SHA-256 均为 `d73231f146f7398d7b50133695faf2116134fbfe33a7b94068e277cc7b82df55`。

## 生产持续观察

- arena-154 从 `2026-08-12T19:34:02Z` 至 `20:10:56Z` 连续观察 36 分 54 秒；prod-108 从 `19:34:03Z` 至 `20:10:28Z` 连续观察 36 分 25 秒。两台均完成 121 组双容器样本，Panel/Relay restart 0、OOM false、fatal 0，Agent active、restart 0，最终数据完整性复核通过。
- arena-154：Panel 内存最小/最大/平均 `10.95/19.62/12.29 MiB`，末 20 样本增量 `+0.01 MiB`，最大 14 PIDs；Relay 为 `4.82/5.15/4.95 MiB`，末 20 样本增量 `+0.02 MiB`，最大 8 PIDs。
- prod-108：Panel 内存最小/最大/平均 `13.00/75.30/17.00 MiB`，首样本升级后暂态为 `75.28 MiB`，随后回落，末 20 样本增量 `+0.01 MiB`，最大 8 PIDs；Relay 为 `4.81/5.00/4.91 MiB`，末 20 样本增量 `0.00 MiB`，最大 8 PIDs。
- 两台尾部均未出现单调资源增长，远低于 Panel `256 MiB`、Relay `128 MiB` 的容器限额。
- 生产证据分别保存在 `/root/kpanel-release-evidence/v0.68.1/arena154-20260812T192418Z` 与 `/root/kpanel-release-evidence/v0.68.1/prod108-20260812T192438Z`；脱敏观察摘要已同步到 `C:\GitHub\_release-artifacts\v0.68.1-reader`。

## 止损与遗留风险

- Reader 异常时可切回 `disabled`；154 Browser Beta 异常时同样优先执行 kill switch，再按备份成套回滚。
- Reader 是安全阅读回退，不承诺复杂 SPA、登录、媒体或脚本挑战兼容；相关页面必须保留可理解提示和系统浏览器入口，不允许静默白屏。
- 原始 Cookie、Bearer、密钥与未脱敏 trace 未进入仓库、Release 或公开验收材料。
- 本轮完整执行已有 `release-kpanel v1.7` 和项目版本治理流程；没有新增重复工作流，本文件作为版本事实、备份和回滚记录沉淀。
