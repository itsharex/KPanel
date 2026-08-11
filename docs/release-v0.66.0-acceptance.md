# KPanel v0.66.0 发布验收记录

日期：2026-08-12

发布级别：L3

候选提交 / 标签：待候选冻结 / `v0.66.0`

上一稳定版本 / 回滚点：`v0.65.0` / `f6c78c4d70473cd393e02e427cf01281968132b1` / OCI index `sha256:8947e7aa281df47612d03a29cd78556b7c60b00a59214fd542a80b58eff57c22`

## 发布画像

- 业务域：桌面模式内置浏览器、Browser Relay、标准安装/更新/回滚链路。
- 变更面：Web UI、Panel API、Relay 协议、双 Origin 部署、应用市场配置；不新增 Host Agent 权限，不修改 `kejilion.sh` 协议和 Panel 数据模型。
- 用户旅程：在 KPanel 桌面内创建会话、输入 URL、前进/后退/刷新、多标签浏览；不适合重写的站点可转系统浏览器。
- 风险等级：高。涉及不受信任网页内容、服务端出网和生产更新，因此必须完成共享密钥、精确 Origin、SSRF 防护、资源上限、回滚及 154 隔离实机门禁。
- 产品边界：本版是轻量安全阅读内核 Phase 1，不宣称等同 Chromium。复杂 SPA、第三方登录、视频和 WebSocket 继续使用系统浏览器回退。

## 候选范围

- 纳入原 Phase 1 实现及修复：浏览会话、同源重写、导航控制、错误/过期处理和系统浏览器回退。
- 新增生产门禁：独立 Relay Origin、Panel/Relay 共享密钥、精确 Origin 校验、非 root/只读/无 capabilities、CPU/内存/PID 上限和健康检查。
- 新增标准应用市场安装/更新/回滚事务：相邻 Relay 端口、密钥文件权限校验、端口预检、失败恢复和孤儿 Relay 清理。
- 未纳入：完整 JavaScript 代理、Service Worker、WebSocket、媒体 DRM 和第三方登录兼容层。

## 已完成的隔离门禁

| 维度 | 状态 | 证据 |
| --- | --- | --- |
| 业务正确性 | 已验证 | 154 隔离实例完成 bootstrap、浏览会话和 IANA 页面重写端到端；Web 87 个测试文件、624 项测试通过 |
| 安全性 | 已验证 | 非法 Origin 403、无效 Token 401、10 组特殊/SSRF URL 400、17 MiB 响应 413、上游 Cookie 不泄漏、重定向不跟随；Trivy HIGH/CRITICAL/secret/misconfig 均为 0 |
| 性能 | 已验证 | 健康接口 2000 次/24 并发：p50 1.54 ms、p95 5.28 ms、p99 69.12 ms；IANA 重写 120 次/6 并发：p50 8.29 ms、p95 45.40 ms、p99 50.42 ms，均无失败 |
| 资源约束 | 已验证 | Panel 256 MiB/1 CPU/128 PID；Relay 128 MiB/0.5 CPU/64 PID；二者非 root、只读、cap-drop ALL、no-new-privileges |
| 安装/更新/回滚 | 已验证 | `packaging/tests/app-conf-lifecycle.sh` 覆盖首次安装、版本拒绝、端口预检、健康失败回滚、密钥权限和双 Origin 配置 |
| 30 分钟稳定性 | 进行中 | 154 隔离环境持续健康与重写采样，完成后补入精确统计和日志摘要 |
| Chrome 桌面交互 | 待验证 | Chrome 已安装且扩展/Native Host 正常，但当前未运行；已向用户请求启动许可 |

## 自动门禁与依赖

- Go 全量测试、`go vet`、核心包 race、Web i18n/typecheck/build、部署安全测试均已在 154 候选源上通过。
- `npm audit` 为 0；`govulncheck` 无 reachable vulnerability，报告的 1 个 required-module 漏洞不可达。
- 候选镜像基于固定 digest 的 Node `24.18.0` 与 Go `1.26.5` 构建，大小 16,019,113 bytes；本版未新增产品依赖。
- 最终 `make verify-release`、候选/main CI、Release workflow、公共多架构镜像和 SBOM/provenance 状态待候选冻结后补录。

## 154 隔离实机

- 候选运行于 `arena-154` 的独立端口 `18080/18081` 和独立 Docker 网络，不修改生产 `8080` 实例。
- Panel 与 Relay 均 healthy、0 restart、OOM false；协议、安全、性能测试均针对实际候选镜像执行。
- 隔离测试账号只用于该候选环境，测试完成后与容器、网络一并清理。

## 生产与回滚

- 生产目标：`arena-154`，当前 `v0.65.0`；用户已授权本次发布和上线。
- 发布前必须停写备份 `/home/docker/kpanel`、应用市场配置和 systemd 链路，并验证 SQLite/JSON/JSONL 完整性。
- 使用标准应用市场更新入口上线；生产 Relay 使用独立端口 `8081`，因为 `8090` 已被无关 Nginx 占用。
- 失败时恢复 `v0.65.0` 精确镜像、Compose、`.env`、Agent/脚本和应用市场配置；删除仅由本次事务创建的 Relay，复核版本、健康、重启/OOM、数据完整性及公网入口。
- 生产部署、持续采样、正式 commit/tag/Release/OCI digest 和 apps 提交均待门禁完成后补录，不得把当前隔离结果表述为生产成功。

## 遗留风险与准入结论

- 当前尚未满足正式上线条件：30 分钟稳定性与 Chrome 桌面交互仍待完成。
- 即使全部通过，本版仍保留 Phase 1 兼容边界；复杂站点回退系统浏览器是预期行为，不是全功能浏览器承诺。
- 复用项目已有 `release-kpanel v1.7` 与版本治理流程；未新增重复工作流。
