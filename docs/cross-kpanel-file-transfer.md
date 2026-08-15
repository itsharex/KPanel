# KPanel 跨面板文件复制

## 任务契约

- 任务 scope / 角色 / AI：`cross-kpanel-file-transfer` / development / Codex。
- 目标与用户价值：允许用户把 KPanel A 文件窗口中的一个或多个普通文件/目录拖到 KPanel B 桌面或文件管理窗口；B 将内容复制到指定目录，桌面目标在复制成功后创建入口。
- 允许路径：`internal/cluster`、`internal/filemanager`、`internal/agent`、`internal/panel`、`internal/contract`、`web/src`、本文件及针对性测试。禁止路径：安装/升级/发布脚本、生产配置、版本号、数据库迁移。
- 共享契约：Cluster v2 Noise 身份、Panel-Agent Unix socket、文件路径/resourceVersion、桌面 workspace。
- 业务真源与用户旅程：`docs/cluster-monitoring.md`、`docs/desktop-icon-workspace.md`、`docs/storage-strategy.md`；A 文件行拖拽 -> B 桌面识别 -> B 后端向已配对 A 拉取 -> B Agent 原子落盘 -> 创建桌面入口。
- worktree / branch / base / rollback：`C:\GitHub\kejilion-panel-cross-kpanel-file-transfer` / `feat/cross-kpanel-file-transfer` / `origin/main@431b9ca5ac21d5a3d1629e46e209ce718367569f` / 源实现回滚到 `431b9ca5ac21d5a3d1629e46e209ce718367569f`；正式版本继续记录上一稳定 Tag、镜像与生产备份。
- 依赖、相邻任务与冲突面：依赖现有 Cluster v2 配对；相邻 worktree 均未修改上述实现路径，治理 worktree 的脏文件不重叠。
- 风险等级：L2。新增跨前后端网络认证、流式传输与 Agent 文件写入；不发布、不改安装链。
- 验收：Go 定向测试与全量测试、Web 单测/typecheck/build、两个本地 KPanel 浏览器跨窗口拖放；长时间浏览器测试按 registered arena 工作流执行。
- 权限：用户已明确授权将文件互传及当前可上线内容提交并进入上线流程；由唯一发布任务负责候选推送、更新 main、tag、Release 与已授权生产部署。
- 交付物：可审阅差异、设计文档、测试与浏览器证据、已知限制。

## 产品规则

1. 跨面板操作永远是复制，不提供移动语义；源文件不发生修改。
2. 拖到 B 桌面时，默认目标为 B 的桌面文件目录（初始值 `/home/KPanel Desktop`）。文件复制完成后才创建桌面入口。
3. 同名目标采用 `name (1)`、`name (2)` 的可预期规则，不覆盖现有内容。
4. 传输中显示来源、目标、当前字节数并允许取消。失败不留下可见的半成品；快捷方式保存失败与文件复制失败分开报告。
5. 只有已通过 Cluster v2 配对且明确带有 `cluster.files.read` scope 的来源节点可用。旧配对不会静默升级权限，需重新配对。
6. 一次最多接收 64 个顶层文件或目录。批量传输按选择顺序逐项执行，每项独立原子提交；单项失败不阻断后续项目，并在结束时展示成功/失败明细。
7. 拖到 B 桌面时复制到桌面文件目录并批量创建入口；拖到 B 文件管理空白区时复制到当前目录，拖到目录项时复制到该目录，不创建桌面入口。
8. 跨面板拖放始终是复制；双向互传要求 A 和 B 分别持有读取对方的有效授权，任何方向都不因另一方向已配对而自动扩权。

## 协议与安全

浏览器兼容读取单项 MIME `application/x-kpanel-cross-panel-file-v1`；批量拖拽使用
`application/x-kpanel-cross-panel-files-v2`，只包含来源 `nodeId` 和最多 64 个项目的文件名、规范化绝对路径、类型及 `resourceVersion`。描述符不是授权凭据；B 后端必须为每项使用本地保存的配对密钥找到 A 并重新鉴权。旧目标仍可接收新来源的单项 v1 描述符，不会把批量误当成单项。

B Panel 通过 Cluster v2 Noise IK 向 A 的固定端点请求文件。握手响应携带经认证的文件元数据，随后复用握手产生的 Noise transport cipher，以长度帧传输加密正文和经认证的结束记录。此方式同时覆盖 HTTPS 与受允许私网中的 `http://IP:port`，且不放宽 SSRF、重放、时钟偏差、并发和空闲超时规则。

目录使用受限 TAR 流传输：拒绝 symlink、special file、路径穿越和超出既有 entry/byte budget 的内容。B Agent 解包到目标目录内的隐藏临时目录，完整结束并同步后以 no-replace rename 原子发布。

审计只记录来源节点、类型、字节数、目标目录和结果；不记录 Noise 明文、密钥或拖拽描述全文。

## 状态机

单项：`idle -> connecting -> transferring -> committing -> shortcut? -> complete`

批量：`idle -> item[n].connecting -> item[n].transferring -> item[n].committing -> next|failed -> shortcut? -> complete|partial`

取消或错误进入 `cancelled` / `error`。在 `committing` 前取消会清理临时文件；文件已提交而桌面入口失败时进入 `partial`，保留真实文件并允许用户手动重试“添加到桌面”。

## 明确非目标

- KPanel 桌面拖回 Windows/macOS 桌面。浏览器无法为远端目录提供原生文件系统对象；文件 drag-out 还依赖浏览器私有下载协议。本任务不伪装为可用能力。
- KPanel A 到 B 的移动、双向同步、断点续传、后台任务跨刷新恢复。
- 未配对面板、Cluster v1 或 light node 的文件传输。

## 开发验收记录

- Go：`internal/filemanager`、`internal/agent`、`internal/cluster` 定向测试通过；Panel 文件相关测试与编译通过；变更范围 `go vet` 通过。
- Web：Vitest 97 个测试文件、715 项测试通过；i18n 校验、`vue-tsc --noEmit`、Vite 生产构建与预压缩通过。
- 浏览器：本地生产构建在应用内真实浏览器中完成桌面壳和经典文件管理页烟测，页面结构正常且控制台无 warning/error；静态预览不具备真实 Panel Session、双 Agent 和 Cluster v2 配对，目录 API 的 `Not found` 属于环境限制，因此未把该结果记录为跨面板传输 E2E。
- 待发布前：在两台已配对 Linux KPanel 上验证单项和多选文件/目录分别拖到桌面、文件管理当前目录与目录项；同时验证双方向授权、同名副本、部分失败、取消、来源变更、旧配对拒绝、断链清理及快捷方式失败保留真实文件。
