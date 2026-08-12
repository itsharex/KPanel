# KPanel 轻量内置浏览器内核

- 基线版本：`v0.68.0`；本文同时记录待发布补丁中的安全阅读模式
- 产品状态：`reader` 默认启用；完整脚本运行时仍为显式 `beta`
- 运行时：安全 Reader + Scramjet v2 Service Worker/WASM + KPanel Relay Transport
- 更新日期：2026-08-13

## 0. 运行模式

| 模式 | 默认 | 能力 | 安全要求 |
| --- | --- | --- | --- |
| `disabled` | 否 | 硬关闭内置浏览器，仅保留 Relay `/healthz` | 紧急止损和管理员显式 kill switch |
| `reader` | 是 | 净化后显示 HTML、文本、JSON 和受限栅格图片；不执行目标脚本、表单、下载或媒体 | 可用于应用市场的 HTTP/IP 拓扑；仅允许 `GET/HEAD` |
| `beta` | 否 | Scramjet v2 上下文重写与第三方脚本运行 | 必须显式启用、双独立安全 Origin 和 Secure Cookie |

Reader 的目标出网仍由独立 `browser-relay` sidecar 完成。已登录的 Panel 页面使用短期 `reader` scope 令牌调用固定内网 Relay 地址，取得有界字节后通过私有 `MessageChannel` 传入同源 reader iframe。令牌不会进入 iframe；reader iframe 同时受 `sandbox="allow-scripts"` 和响应头 `CSP: sandbox allow-scripts` 约束，不含 `allow-same-origin`，且 `connect-src 'none'`。目标内容通过新建 DOM 节点的 allowlist 净化后显示，不使用 `innerHTML` 或 `srcdoc`。

Reader 与 Beta 的令牌 scope 不可互换。Reader Relay 拒绝写方法、请求体以及 Cookie、Authorization、Origin、Referer 等上游请求头，并从返回元数据中剔除 `Set-Cookie`。Relay 对 Reader 单次响应实施 8 MiB 和 30 秒总上限；Panel 丢弃重定向 body，并把完整导航链限制为 45 秒。文档响应上限为 8 MiB，HTML 解析前另限制为 4 Mi 字符和 12,000 个标记，净化树限制 12,000 节点与 128 层；单图 2 MiB、每页最多 24 张和 12 MiB，总并发最多 4。SSRF、DNS 拨号复核、TLS、Origin、会话并发和空闲超时继续复用当前 Relay 安全边界。

下文第 1 至第 6 节主要描述完整 `beta` 运行时；Reader 的差异以本节和第 7 节为准。

## 1. 定位与非目标

KPanel 没有嵌入 Chromium、Chrome、Electron、CEF 或其他独立浏览器进程。内置浏览器复用用户现有浏览器提供的 iframe、Service Worker、WASM 和标准 Web API，在独立 Relay Origin 内重写网页，并由服务器 Relay 访问目标站点。

这里仍使用 iframe，但它只承担隔离视口：Panel iframe 加载的是 KPanel 自己控制的 Relay `/kernel/`，内层 iframe 加载的是同一 Relay Origin 下的重写结果，不会直接使用 `iframe src="https://第三方站点"`。因此它不是旧版“直接嵌入第三方 iframe”架构。

目标是兼容公开互联网中的常见 HTTP(S) 网页，同时保持轻量、可回滚和较好的日常浏览体验。它不是 Chromium 的完整替代品，也不是经过独立安全审计的高敏感业务浏览器。

“任意 URL”仅指语法有效且目标解析为公网地址的 `http://` 或 `https://` URL，不包括本机、内网、云元数据、本地文件和其他协议。

## 2. 运行链路

```text
用户现有浏览器
  ├─ HTTPS → Panel Origin（例如 https://panel.example.com）
  │    └─ POST /api/v1/browser/sessions
  │         └─ 生成 10 分钟 HMAC-SHA256 浏览器会话令牌
  └─ HTTPS → Relay Origin（例如 https://browser.example.com）
       └─ /kernel/ 外层隔离 iframe
            ├─ 注册 Relay Origin 根 Scope Service Worker
            ├─ Scramjet Controller 创建内层网页视口
            ├─ 重写 HTML、JavaScript、CSS、URL 和常用浏览器 API
            └─ KPanel Relay Transport → POST /v1/fetch
                 ├─ 校验令牌、Origin、URL、DNS/IP 和资源预算
                 └─ HTTP(S) → 目标网站
```

目标网站脚本在重写后的 Relay 运行环境中执行。Panel 只通过精确 Origin、精确消息来源和导航事务 ID 接收地址、标题和错误状态，不读取第三方 DOM。

所有正常目标资源流量均经过 Relay，因此：

- 目标网站看到的是 Relay 服务器出口 IP；
- Relay 承担页面、脚本、图片和媒体流量；
- Relay 会终止到目标 HTTPS 的连接并能够看到明文内容；
- “使用系统浏览器打开”是显式兜底，只有该入口会让用户浏览器直接访问目标站点。

## 3. TLS 与双 Origin 部署

正式环境必须提供两个不同的 HTTPS Origin：

```text
https://panel.example.com    → Panel
https://browser.example.com  → browser-relay
```

不能只通过同一域名的不同路径部署，也不能在正式环境使用公网 IP 的明文 HTTP。Service Worker 需要安全上下文；标准浏览器仅对 HTTPS 和本机开发环境等有限场景开放该能力。

TLS 分为两段：

1. 用户浏览器到 Panel/Relay：由正式反向代理提供 HTTPS，并由用户浏览器验证证书。
2. Relay 到 HTTPS 目标：使用 Go 标准 TLS，验证目标域名和证书链；没有关闭证书验证。

访问 `http://` 目标时，用户到 Relay 仍应是 HTTPS，但 Relay 到目标的最后一段没有 TLS。该模式用于兼容旧站点，不应输入敏感信息。

反向代理必须保留流式响应，并正确转发 Service Worker 的 `Content-Type`、`Service-Worker-Allowed` 和缓存策略。`/kernel/runtime/v3/sw.js` 使用 `no-cache`，固定版本的 JS/WASM 资源使用不可变缓存。

## 4. 安全边界

### 4.1 Panel 与网页运行时隔离

- Panel Origin 与 Relay Origin 必须不同；配置校验会拒绝相同 Origin。
- Panel 的 `frame-src` 只放行自身、`blob:` 和配置的精确 Relay Origin。
- Panel 与 Relay 的 `postMessage` 同时校验 `event.origin`、`event.source` 和有界导航 ID。
- 第三方网页脚本只在 Relay Origin 的外层沙箱内执行，无法按同源策略读取 Panel DOM、Panel Cookie 或 Panel Storage。
- Relay CSP 所需的 `'unsafe-eval'` 和 `'wasm-unsafe-eval'` 不进入 Panel CSP；摄像头、麦克风、定位、支付和 USB 权限在 Relay 上关闭。

Scramjet 的隔离正确性属于安全边界的一部分。第三方脚本与重写控制器共享专用 Relay Origin，因此该 Origin 必须专用于浏览器内核，不能同时承载后台、登录页、API 控制台或其他可信业务，也不能在该 Origin 保存 Panel 凭证。

### 4.2 会话与请求授权

- Panel 登录会话通过同源、CSRF 和权限校验后才能创建浏览器会话。
- Relay 令牌使用至少 32 字节随机共享密钥和 HMAC-SHA256；默认有效期 10 分钟，协议上限 15 分钟。
- 令牌绑定随机浏览器会话 ID，不进入 URL、持久化存储或正常日志。
- `/v1/fetch` 只接受来自 Relay 自身精确 Origin 的请求，并要求 Bearer 令牌。

### 4.3 SSRF 与网络出口

Relay 只允许 `GET`、`HEAD`、`POST`、`PUT`、`PATCH`、`DELETE` 和 `OPTIONS`，并拒绝：

- 非 HTTP(S) 协议、带用户名或密码的 URL、非法端口和非规范主机名；
- `localhost`、`.local`、`.internal`、`.home.arpa`；
- 回环、私网、链路本地、共享地址、组播、文档网段、转换前缀及其他特殊用途 IPv4/IPv6；
- 任一 DNS 结果包含非公网地址的主机。

首次请求和实际拨号都会重新解析并校验目标，连接只拨向已验证 IP。Go Transport 不继承环境代理，不自动跟随重定向；重写运行时发起的后续 URL 请求会再次经过完整策略。HTTPS 目标保留标准证书和主机名验证。

### 4.4 资源与稳定性预算

| 项目 | 默认值 |
| --- | --- |
| Relay 全局并发 | 64 |
| 单浏览器会话并发 | 16 |
| 单目标主机连接 | 6 |
| 最大空闲连接 | 128 |
| 目标 URL | 16 KiB（UTF-8）；反向代理必须允许单个 16 KiB 请求头、总请求头不低于 64 KiB |
| 请求体 | 16 MiB，配置硬上限 64 MiB |
| 请求/响应头元数据 | 最终编码后 32 KiB、最多 128 对；上游响应头读取上限 64 KiB |
| 连接 / TLS / 响应头超时 | 5 秒 / 10 秒 / 15 秒 |
| 响应体空闲超时 | 30 秒 |
| 流式复制缓冲 | 32 KiB 复用缓冲 |
| 前端标签 | 最多 8 个，最多 2 个活动运行时；非活动 45 秒后休眠 |
| Relay 容器 | 0.5 CPU、128 MiB、64 PIDs、非 root、只读根、`cap_drop: ALL` |

浏览器 Transport 与 Relay 对目标 URL 使用相同的 16 KiB UTF-8 上限，避免重写后的搜索、媒体和动态资源 URL 被链路中途以不同预算拒绝；协议、凭据、主机名、DNS/IP 和 SSRF 校验不因该兼容上限改变。正式反向代理必须验证该单请求头预算，不能通过截断目标 URL 继续请求。

Relay Origin 的 `connect-src` 只允许自身以及 Controller 在本地解析的 `data:`、`blob:`；其中 `data:`、`blob:` 不产生目标站网络连接，第三方 HTTP(S) 仍只能通过带令牌并执行完整目标策略的 Relay 请求。

Scramjet 转交的请求体会在用户浏览器中按 16 MiB 上限转换为定长二进制请求，空 POST 在 Relay 侧保持明确的零长度语义，以兼容普通 HTTP/1.1 链路；超限时不会向 Relay 发送部分请求。响应体继续采用流式转发，没有固定的页面总下载字节上限；大型媒体会消耗服务器带宽，但不应被整页缓冲进 Relay 内存。并发、单主机连接和空闲超时用于限制异常站点造成的无界占用。

## 5. 兼容性与用户体验

当前 v2 运行时不再是 Phase 1 的“删除脚本和 CSS 的只读净化器”。它允许重写后的第三方 JavaScript 执行，并保留上下文相关的资源、导航和存储语义，以提高 SPA 和复杂网页兼容性。

当前实现提供：

- 地址输入、搜索、链接点击、滚动、表单交互；
- 后退、前进、刷新、页面内跳转、地址和标题同步；
- HTML、CSS、JavaScript、图片、字体及普通媒体资源的 Relay 重写和流式加载；
- 会话令牌自动刷新、加载超时提示、错误反馈和“使用系统浏览器打开”兜底；
- 最多 8 个标签、最多 2 个活动运行时和非活动标签休眠。

以下能力仍不承诺完整兼容：

- WebSocket：当前 KPanel Transport 明确返回不支持；
- DRM、受保护媒体、浏览器扩展、原生协议、摄像头/麦克风/定位/支付/USB；
- 依赖严格反代理检测、复杂 SSO、第三方登录弹窗、站点自身 Service Worker 或非标准浏览器行为的页面；
- 下载管理、跨窗口状态、Cookie/Storage/SRI/CSP/ES Module 等所有边缘语义；
- YouTube 等媒体站点的持续播放闭环，在真实 HTTPS 环境完成门禁前不能宣称完整兼容。

安全策略命中、上游主动反代理、网络不可达或未实现 API 时，应显示可操作错误或允许外部打开，不应以放宽 Panel Origin、TLS 或 SSRF 边界换取单站点兼容。

Chrome/Edge/Firefox 只作为真实用户浏览器和验收工具使用，不会被打包到 KPanel 镜像，也不会在生产服务器中作为内核进程运行。

## 6. 第三方运行时与发布状态

| 组件 | 固定版本 | 发布通道 | 完整性记录 |
| --- | --- | --- | --- |
| `@mercuryworkshop/scramjet` | `2.0.67-alpha.2` | v2 alpha，非稳定版 | `internal/browsercore/vendor/manifest.json` |
| `@mercuryworkshop/scramjet-controller` | `0.0.14` | 与上述 v2 运行时配套 | `internal/browsercore/vendor/manifest.json` |

运行文件已 vendored 到 Go 二进制，不在用户运行时从 npm 或 CDN 下载；发布前必须复核包完整性、文件 SHA-256、许可证、SBOM 和已知漏洞扫描结果。

`2.0.67-alpha.2` 不能描述为“最新稳定版”。`npm audit` 或漏洞数据库返回零已知漏洞，只说明当前数据库没有匹配公告，不代表不存在未知绕过，也不等同于独立安全审计。

因此 v0.68.0 的该能力按 Beta 管理并默认关闭。若允许生产发布，必须显式设置 Beta 模式，并在发布验收中记录针对该精确版本的限期例外、责任人、复核日期、退出条件和回滚点；兼容的 Scramjet v2 稳定版可用后，应重新评估并优先退出 alpha 例外。

## 7. 运行配置

Panel：

```text
KEJILION_PANEL_BROWSER_MODE=reader
KEJILION_PANEL_BROWSER_RELAY_URL=http://PUBLIC_IP:8081
KEJILION_PANEL_BROWSER_RELAY_INTERNAL_URL=http://browser-relay:8090
KEJILION_PANEL_BROWSER_RELAY_SECRET_FILE=/run/secrets/browser-relay-secret
KEJILION_PANEL_SECURE_COOKIE=false
```

标准 Compose 和应用市场默认使用 `reader`。Reader 需要公开 Relay Origin、固定内网 Relay Origin 与共享密钥，但 iframe 只加载 Panel 同源的 `/browser-reader/`，因此 HTTPS 反向代理不会再尝试嵌入 HTTP Relay。成功会话显式返回 `mode: "reader"`。`disabled` 继续作为硬 kill switch；其他模式值会在配置校验阶段被拒绝。

应用市场使用一次性 `KPANEL_BROWSER_MODE_MIGRATION=reader-v1` 标记记录迁移选择。由于 v0.68.0 没有记录模式来源，更新器不会静默覆盖尚未标记的 `disabled`：交互更新会询问是否启用 Reader，默认答案为否；非交互更新保持 `disabled`，需要迁移时显式设置临时环境变量 `KPANEL_BROWSER_READER_MIGRATION=reader`。标记写入后，管理员选择的 `reader` 或 `disabled` 都会被后续更新保留。

完整 Beta 配置仍为：

```text
KEJILION_PANEL_BROWSER_MODE=beta
KEJILION_PANEL_BROWSER_RELAY_URL=https://browser.example.com
KEJILION_PANEL_BROWSER_RELAY_INTERNAL_URL=http://browser-relay:8090
KEJILION_PANEL_BROWSER_RELAY_SECRET_FILE=/run/secrets/browser-relay-secret
KEJILION_PANEL_SECURE_COOKIE=true
```

Relay：

```text
kpanel-browser-relay \
  -listen :8090 \
  -mode beta \
  -allowed-origin https://panel.example.com \
  -public-url https://browser.example.com \
  -secret-file /run/secrets/browser-relay-secret
```

正式 Compose 使用同一不可变镜像启动 Panel 和独立 Relay 容器，并把同一个 `KEJILION_PANEL_BROWSER_MODE` 同时传给 Panel 和 Relay。安装器通过 `--browser-mode disabled|reader|beta` 接收模式，默认 `reader`；同时生成 32 字节随机密钥，以 `root:kejilion-panel 0640` 只读挂载给两个容器。Relay 健康检查会校验实际模式与期望模式一致，避免“健康但功能被禁用”。批准 Beta 例外后，部署配置才可显式启用 `beta`；HTTPS Beta 必须同时启用 Secure Cookie。Panel 与 Relay 必须同时升级。

## 8. 上线门禁

发布前至少完成：

1. 在应用市场直连 HTTP/IP 与单 Panel HTTPS 反向代理上验证 Reader；在真实、受信任的双 HTTPS Origin 上验证 Beta Service Worker 注册、更新和刷新。
2. 使用真实桌面 Chrome/Edge 进行静态站、服务端渲染站、SPA、搜索、表单、媒体和异常页测试，覆盖输入、点击、滚动、跳转、后退、前进、刷新和标签休眠恢复。
3. 验证 Google/Bing 搜索结果、GitHub/Baidu 等普通站点，以及 YouTube 页面和实际播放状态；未通过项必须写入已知限制。
4. 验证错误 Origin、无令牌/过期令牌、私网地址、DNS 重绑定、重定向到内网、无效 TLS 证书、超限请求和并发限流。
5. 完成持续负载、内存、CPU、重启、OOM、连接回收和服务器出口流量观测。
6. 复核固定依赖哈希、许可证、SBOM、漏洞扫描及 alpha 版本例外记录。
7. 验证默认 `reader` 的 iframe 不含 `allow-same-origin`、CSP 禁止联网、目标脚本零执行、目标请求零直连；同时验证 `disabled` 硬关闭，只有显式 `beta` 才启用完整脚本运行时。
8. 演练 v0.67.0 回滚，并验证旧内核不受客户端遗留 Service Worker 影响。

`docs/embedded-browser-core-154-acceptance.md` 记录的是 v0.66.0 Phase 1 历史验收，不能作为 v0.68.0 v2 运行时的上线证据。

## 9. 回滚

生产回滚点必须是本次上线前实际运行的不可变镜像、Compose 和 `.env`：从 `v0.68.0` 升级时回到 `v0.68.0`，仍锁定在 `v0.67.0` 的环境则回到 `v0.67.0`。浏览器内核变更没有数据库迁移，也不应删除 Panel 数据、站点、应用、Relay 密钥或业务容器。

回滚时：

1. 在替换镜像前成套恢复升级前的 Compose 与 `.env`，不得把含 `reader` 或 `-mode` 参数的新版配置交给旧版。`v0.68.0` 不识别 `reader`，`v0.67.0` Relay 还不支持 `-mode` 参数；备份不可用时停止自动回滚并人工重建目标版本的匹配配置。
2. 同时把 Panel 和 Relay 恢复到同一份上线前不可变镜像，不能混跑新版 Panel 与旧 Relay。
3. 按标准应用市场/部署回滚流程恢复镜像，复核 Panel、Relay、Agent 健康、重启次数和 OOM 状态。
4. Service Worker 是客户端持久状态，单纯回滚服务器镜像不会主动删除已注册 Worker。正式演练必须验证旧内核可正常接管；如出现干扰，应在同一 Relay Origin 提供明确的 Worker 退役流程，或切换到干净的 Relay Origin，并提示用户刷新站点数据。
5. 复测普通网页、错误提示和“使用系统浏览器打开”兜底，再恢复发布流量。

源码层面优先使用 `git revert` 撤销候选提交，不改写共享分支历史。发布标签和既有 Release 保持不可变。

紧急止损不必先回滚镜像：把 Panel 与 Relay 的模式同时改为 `disabled` 并重启两个容器，即可停止签发新会话，让 Relay 除 `/healthz` 外全部返回 503，并立即使已有令牌失去可用入口；随后再按上述流程回滚或排障。
