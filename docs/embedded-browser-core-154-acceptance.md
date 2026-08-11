# 内置浏览器内核 154 隔离验收

## 范围与结论

- 验收机：`arena-154`，Debian 13 / Linux amd64 / AMD EPYC 7K62。
- 最终候选：`c707e240c5a72167fa405aa9e08b3510f74140e2`。
- 测试拓扑：候选 Panel 与 Relay 分别绑定远端回环端口 `18080`、`18081`，使用独立数据、密钥、Cookie 和受限容器；未替换生产 Panel/Agent。
- 结论：**安全、性能和短时稳定性门禁通过；完整替代现有网页访问链路仍为 No-Go。** 当前可以继续作为 Phase 1 隔离预览内核验证，不应直接进入生产全量切换。

## 验收中发现并修复的问题

1. Relay 未进入 `Makefile` 和最终镜像，源码可测但发布产物不可用；已在提交 `e7a23f3` 补齐。
2. Relay 空闲超时回调与 `Close()` 存在数据竞争；154 的 `-race` 实际复现，已在提交 `a0d44b1` 同步定时器初始化并复验通过。
3. Panel 仍允许 `frame-src 'self' http: https:`，不符合停止任意第三方 iframe 的目标；已在提交 `b5acb1b` 收紧为 `'self' blob: <exact-relay-origin>`，提交 `c707e24` 完成格式门禁。

## 构建与安全门禁

- Linux 全量 `go test ./...`、`go vet ./...` 通过。
- `go test -race ./internal/panel ./internal/auth ./internal/browsercore ./internal/dockerx` 通过；CSP 修正后 Panel/BrowserCore Race 再次通过。
- Web 锁文件安装、`vue-tsc`、全量 Vitest、生产构建通过；`npm audit` 为 0 漏洞。
- `govulncheck` 可达漏洞为 0；依赖模块中存在 1 个不可达漏洞，未命中调用链。
- Trivy 源码和最终镜像 HIGH/CRITICAL 漏洞、Secret、Misconfiguration 均为 0。
- 最终容器均为只读根文件系统、`cap_drop=ALL`、`no-new-privileges`；Panel 限额 `256 MiB / 1 CPU / 128 PIDs`，Relay 限额 `128 MiB / 1 CPU / 64 PIDs`。
- 动态检查通过：错误/缺失 Origin、伪造 Token、10 类私网/特殊地址/危险 URL、CONNECT、异常 Header、17 MiB 请求体、上游 Cookie 隔离、302 不自动跟随、5 秒慢 Header 关闭及精确 CSP。

## 性能与稳定性

### 真实 HTTP 链路

| 场景 | 数量 / 并发 | 错误 | P95 | P99 |
| --- | ---: | ---: | ---: | ---: |
| Relay `/healthz` | 2000 / 24 | 0 | 7.781 ms | 11.980 ms |
| IANA HTTPS Fetch | 120 / 6 | 0 | 18.240 ms | 29.298 ms |

`example.com` 高频并发曾出现上游 `403/502`，换用 IANA 同参数后 120/120 成功；判断为单一上游/WAF/连接波动，不能把外部站点响应时间视为纯 Relay 开销。

### 进程内基准

- 64 KiB Relay：`30.114–32.376 µs/op`，`2.02–2.18 GB/s`，约 `12.1 KiB/op`、`102 allocs/op`。
- Limiter：`244.5–281.3 ns/op`，`64 B/op`、`2 allocs/op`。

### 8 分钟长稳

- Relay 健康检查 240/240、真实 IANA Fetch 240/240 成功。
- 健康 P95 `2.839 ms`，Fetch P95 `18.403 ms`。
- Panel/Relay 日志错误 0、重启 0、OOM 0。
- Relay 内存峰值约为 128 MiB 限额的 `8.75%`；长稳结束后约 `11.12 MiB`。
- Panel 登录 Argon2 哈希后短时达到约 `138 MiB`，随后回落到约 `11.39 MiB`；这是认证路径既有峰值，不是 Relay 常驻开销。

## 兼容性结果与边界

- IAB 与真实 Chrome 均能加载并正确渲染候选初始化/登录页面，标题、中文表单和安全文案正常。
- 真实认证 → Browser Session → Relay Token → HTTPS Fetch → Kernel CSP 协议链通过。
- URL 抓取矩阵覆盖 Example、IANA、GitHub、Wikipedia、httpbin、Cloudflare、MDN；Relay 均正常返回上游结果，其中 Wikipedia 上游返回 403。
- 当前浏览器控制桥在连续表单输入时多次超时，因此登录后桌面窗口、标签页、前进后退和 iframe 内渲染未取得完整真实 Chrome 操作证据，不能标记为通过。
- Phase 1 仍不执行第三方 JavaScript，不保留原站 CSS、登录 Cookie、表单、WebSocket 和完整 SPA 运行时；复杂登录站、视频站和强 JS 应用会降级或不可用。这与“任意 URL、无体验损失”的最终目标仍有距离。

## 上线前阻断项

1. Compose/安装器尚未正式管理 Relay 服务、共享密钥、双 Origin、健康检查和回滚；当前只完成二进制进入最终镜像。
2. 补齐登录后的真实 Chrome 桌面模式操作闭环，并覆盖桌面/移动视口、标签页、导航、错误页和会话过期。
3. 建立按站点类型分层的兼容语料库与清晰降级提示，明确静态页、服务端渲染、SPA、登录、媒体、下载和 WebSocket 的支持等级。
4. 在正式双域名 HTTPS/反向代理拓扑下复验 CSP、Cookie、Origin、证书、缓存和跨域预检。
5. 上线前执行更长时间和多 Session 并发长稳；本轮 8 分钟只证明短时稳定，不替代 30 分钟/2 小时门禁。

## 生产保护与回滚

- 测试前后生产 `kejilion-panel` 容器 ID、镜像 ID、端口和健康状态未变化，Agent 始终 active。
- 所有非测试业务容器 ID 与测试前基线一致。
- 代码回滚点为原提交 `9d254600056c48f793cbd540bed4c846a9e9f3b0`；最终候选在其后增加发布接线、Race 修复和 CSP 收紧。
- 154 脱敏证据位于 `/root/kpanel-release-evidence/browser-core-e7a23f3`。
