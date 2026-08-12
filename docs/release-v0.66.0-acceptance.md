# KPanel v0.66.0 发布验收记录

日期：2026-08-12

发布级别：L3

正式提交 / 标签：`94ac4674332cbbf83a1207a209568248711a0f20` / `v0.66.0`

上一稳定版本 / 回滚点：`v0.65.0` / `f6c78c4d70473cd393e02e427cf01281968132b1` / OCI index `sha256:8947e7aa281df47612d03a29cd78556b7c60b00a59214fd542a80b58eff57c22`

## 发布画像

- 桌面模式新增独立轻量安全阅读内核。Panel 只签发 10 分钟短时会话，第三方公网 HTTP(S) 内容由独立 Relay 获取、净化并在隔离 Origin 中渲染。
- 新增 Relay SSRF/DNS 重绑定防护、精确 CORS/CSP、并发与流量预算、空闲超时、图片代理、健康检查和可理解错误状态。
- 正式 Compose、安装器和应用市场契约新增非特权 Relay、Panel/Relay 共享随机密钥、双 Origin、相邻独立端口、健康检查、更新事务和失败回滚。
- 变更涉及 Web、Panel API、Relay 协议和部署契约，不新增 Host Agent 权限，不修改 `kejilion.sh` 协议或 Panel 数据模型。
- 本版本是 Phase 1 安全阅读内核，不是 Chromium。复杂 SPA、第三方登录、视频、WebSocket、DRM 和完整下载管理继续使用系统浏览器回退。

## 候选范围与冻结

- 候选基线为 `7f5735ad50998a0fc6ec13bc0d082c6993bcbdc1`，正式候选包含 8 个连续提交，冻结 HEAD 为 `94ac467`。
- 净差异为 49 个文件、3386 行新增、117 行删除；包含浏览内核、Panel 会话、Relay、部署/应用市场事务、测试、设计、验收和版本元数据。
- `go.mod`、Go 依赖和 npm 依赖图未变化；`package-lock.json` 仅同步版本号。没有混入无关依赖升级或其他工作树内容。
- bundle：`C:\GitHub\_release-artifacts\v0.66.0\kpanel-v0.66.0-94ac467.bundle`，SHA-256 `3aabb0cf305fd7c63e80586b0b37c0600a4650db7b1b206fc6b65a8ee0155d13`，`git bundle verify` 确认完整历史。

## 自动门禁与 L3

- 精确候选完成完整 L3：Go 全量测试、核心包 race、`go vet`、Linux amd64/arm64 Panel/Agent/Node/Relay 构建、Web 87 个测试文件/624 项测试、i18n、typecheck、生产构建、安装安全和应用配置生命周期全部通过。
- `npm audit` 为 0；`govulncheck` 可达漏洞为 0；Trivy 源码和最终镜像 HIGH/CRITICAL、Secret、Misconfiguration 均为 0。
- 固定 `kejilion.sh` 提交为 `28f89c1b34df4b25e6ef9b144c328fdea75dbac9`，raw SHA-256 `0583f7cd5be1f0bb6ec48d92e2cf224bfabfafada5788658bda4414ba9561229`，L3 和生产再次核对通过。
- L3 日志：`C:\GitHub\_release-artifacts\v0.66.0\l3-94ac467.log`，SHA-256 `322429fb9dfa24f0c55173dd58678e65c2c9d76f6df50f6a459b1ef2131fb82c`，最终标记 `L3 release verification completed`。
- 候选 CI `31548682130`、候选 Dependency freshness `31548682128`、主线 CI `31548876344`、主线 Dependency freshness `31548876364`、Release run `31549078161` 全部成功。

## 安全、性能与稳定性

- 隔离真机验证非法 Origin 403、无效 Token 401、10 类特殊/SSRF URL 400、17 MiB 超限 413、上游 Cookie 不泄漏、重定向不自动跟随、慢 Header/Body 关闭和精确 CSP。
- Relay 运行约束为非 root、只读根、`cap-drop ALL`、`no-new-privileges`、128 MiB、0.5 CPU、64 PIDs；Panel 保持 256 MiB、1 CPU、128 PIDs。
- 30 分钟稳定性采样为 900/900，错误 0、Panel/Relay 重启 0、OOM false；Relay 平均/峰值内存 11.01/14.94 MiB，平均 CPU 0.154%。结果文件 SHA-256 `401491fc3bcb00e38010cc3ff71e18d933f8875bba5818a1104c3566fe20bd8d`。
- 健康接口 2000 次/24 并发和 IANA 重写 120 次/6 并发均无失败；详细性能与兼容性证据保存在本版本发布产物目录和 154 隔离证据中。

## Chrome 用户旅程

- 使用 Google Chrome `151.0.7922.76` 与 ChatGPT 扩展，于 2026-08-12 07:54-07:57（Asia/Shanghai）完成真实桌面交互。
- 登录、桌面模式、浏览器双击启动、最大化/最小化/任务栏恢复、刷新、多标签切换与关闭全部通过。
- IANA Reserved Domains 与 `example.com` 经真实 Relay 内核渲染成功；`http://127.0.0.1/` 被 HTTP 400 拒绝并显示可理解错误；“用系统浏览器打开”精确打开 IANA URL。
- 业务页面控制台错误为 0；隔离候选没有生产 Agent 权限，页面显示 Agent 离线属于预期边界。
- Chrome 证据：`C:\GitHub\_release-artifacts\v0.66.0\chrome-acceptance.md`，SHA-256 `4a52584a337af8b47421fb8e2dec51a43ac852aad5cdf8d74ac6362a293e22cd`。

## Release、镜像与 apps

- [GitHub Release v0.66.0](https://github.com/kejilion/KPanel/releases/tag/v0.66.0) 已公开，非 draft、非 prerelease，共 10 个附件，增加 Relay 的 amd64/arm64 原生二进制并保留 Agent、Node、部署归档、校验和与许可文件。
- annotated tag object 为 `b19f34c993f3c031059c9c3c6ea1c9d6fa3a109c`，peel 后精确指向 `94ac4674332cbbf83a1207a209568248711a0f20`。
- `docker.io/kjlion/kejilion-panel:0.66.0` 与 `latest` OCI index 均为 `sha256:2abfb267da7a1ead720dbe3d44928a2fe76e3f3ef2efe727b7082c62718bbd99`。
- `linux/amd64` 子清单为 `sha256:7c12eece4ac626ca38fc2ad894daa858638cf94b250090d6b7e1bea5ab31edcc`；`linux/arm64` 子清单为 `sha256:fd7152b73d320771a0483fd10aa28cc2e99a9363077339801ce09e45db3c4cbb`；其余 `unknown/unknown` 为 attestation。
- 154 从公开仓库拉取版本镜像，公开镜像 E2E 输出 `image_e2e=pass`，日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- apps 提交为 `b34a3992806b65fac789a0a28cb7018b1cbec501`；`kpanel.conf` 与 KPanel 候选 blob 均为 `a86f3130d4e3c120aa53bd728900e657e70c04a2`，文件 SHA-256 `201ae7babced0e9a5efb7833c8491a81c61f8603323067b91ed9e2b1ec2890fe`。

## 生产部署与观察

- 生产真源为 `arena-154` 的 `http://154.36.153.9:8080`；Relay 为 `http://154.36.153.9:8081`。`kp.kejilion.pro` 不属于本次生产目标。
- 升级前为 `v0.65.0`，Panel healthy、0 restarts、OOM false，Agent active；生产 Relay 不存在，8081 端口空闲。
- 停写一致性备份目录：`/root/kpanel-backups/v0.66.0-preupgrade-20260812T001807Z`；归档：`/root/kpanel-backups/v0.66.0-preupgrade-20260812T001807Z.tar.gz`；SHA-256 `908eee9f29a23449fd0bd401078cb499a76e57ac399321bd542f07073c150385`。源数据、归档、独立解包后的清单/哈希、SQLite、JSON 和 JSONL 均通过恢复校验，旧版恢复 healthy 后才执行升级。
- 使用标准入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel` 更新成功，没有触发应用回滚；更新日志 SHA-256 `9aa2a055356422c888277f3d25a257e6c31056793856a9b6f610aa22c733e6c8`。
- 更新后本机与公网 Panel 均为 `0.66.0/ok/v1alpha1`，本机与公网 Relay 均为 `ok/kpanel-browser-core/v1`；Panel/Relay 均 healthy、0 restarts、OOM false；Agent `0.66.0 v1alpha1`、active。
- 生产镜像、源码标签、Relay 资源/权限/健康检查、共享密钥权限、apps 配置、固定脚本和 Agent/Relay 二进制摘要均通过核对。Agent SHA-256 `9e58903168842b11c936556bae0fa0fc96a004d85472a85ce8890ed5a2bfc3af`；Relay SHA-256 `635c2ca8f153c7d7b174fc2888c28a4204eabe54eb51715cef994d7a481859ad`。
- `agent.env`、systemd、Agent token、AI secret 和集群身份哈希与备份一致；`.env` 仅新增预期的 `KPANEL_BROWSER_RELAY_URL=http://154.36.153.9:8081`。SQLite quick check、5 个 JSON 和 14986 条 JSONL 解析通过；最近 Panel/Relay/Agent 日志没有 panic、fatal、segmentation fault 或 OOM。
- 生产验证日志 SHA-256 `b9a84eff3f356562f7bb145d998007d9ca8a078bf4f5443d47307561dda181a3`。
- 60 次、2 秒间隔持续采样从 `2026-08-12T00:19:29Z` 到 `00:21:41Z` 全部通过；采样同时覆盖本机/公网 Panel、Relay、两个容器健康/重启/OOM和 Agent。采样 SHA-256 `7736addaa8bc0537464d19fe27e5ad2c4d4c69063a5c41afd74ecc0e888a8945`。
- 生产只执行发布、健康、完整性和只读协议验收；完整登录后浏览器旅程来自隔离候选 Chrome 证据，没有读取或传输生产凭据。

## 回滚

- 源码/Tag：`v0.65.0` / `f6c78c4d70473cd393e02e427cf01281968132b1`。
- 镜像：OCI index `sha256:8947e7aa281df47612d03a29cd78556b7c60b00a59214fd542a80b58eff57c22`；amd64 `sha256:73afda66298cfa6e46d655e8c574ee15dc0e045fe6ba2299be074021a5ef2173`；arm64 `sha256:fa06659353caed0951423324c007eb2d0d74f40a47a04d08a6963a5b68d492c5`。
- 停止 Panel、Relay 与 Agent，将 Compose、`.env`、Agent/脚本、应用市场配置和 systemd 链路恢复到已验证备份，并固定上一稳定镜像摘要；删除仅由 v0.66.0 创建的 Relay 容器。执行 `systemctl daemon-reload` 后恢复 Agent 和 Panel。
- 回滚后复核 v0.65.0 版本、Panel 健康/重启/OOM、Agent、数据完整性和公网入口。历史 `v0.66.0` Tag/Release/版本镜像保持不可变；若生产回滚，公共 `latest` 和标准更新入口必须通过新的规范动作恢复上一稳定默认版本。

## 节奏、限制与沉淀

- 首个候选功能提交：2026-08-12 06:49:01+08:00；候选冻结：07:23:58；Chrome 门禁完成：07:57；Tag：08:07:23；Release 公开：约 08:12；生产持续采样完成：08:21:41。
- 功能提交到生产完成约 1 小时 33 分钟，冻结到生产完成约 58 分钟；没有应用回滚、紧急热修复或重复版本。
- 已知产品边界是 Phase 1 不执行第三方 JavaScript、不承诺复杂 SPA/登录/媒体/WebSocket/DRM；系统浏览器回退是明确产品路径，不是隐式失败。
- 本轮复用 `release-kpanel v1.7` 和现有版本治理流程，没有新增重复工作流；发布事实沉淀在本文件中。
