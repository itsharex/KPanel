# KPanel 轻量内置浏览器内核

- 候选版本：`v0.68.0`
- 产品状态：Beta，默认关闭，必须显式配置 `KEJILION_PANEL_BROWSER_MODE=beta`
- 运行时：Scramjet v2 Service Worker/WASM + Scramjet Controller + KPanel Relay Transport
- 更新日期：2026-08-12

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
KEJILION_PANEL_BROWSER_MODE=beta
KEJILION_PANEL_BROWSER_RELAY_URL=https://browser.example.com
KEJILION_PANEL_BROWSER_RELAY_SECRET_FILE=/run/secrets/browser-relay-secret
KEJILION_PANEL_SECURE_COOKIE=true
```

默认值为 `disabled`。只有精确值 `beta` 会初始化浏览器令牌服务、把 Relay 加入 Panel `frame-src` 并允许创建浏览器会话；成功响应显式返回 `mode: "beta"`，界面常驻 Beta 标识。禁用时会话接口返回 `503 browser_beta_disabled`，Relay 只保留 `/healthz`，旧令牌立即失效，并保留系统浏览器兜底；其他模式值会在配置校验阶段被拒绝。停用 Beta 不需要删除 Relay 密钥或数据。

Relay：

```text
kpanel-browser-relay \
  -listen :8090 \
  -mode beta \
  -allowed-origin https://panel.example.com \
  -public-url https://browser.example.com \
  -secret-file /run/secrets/browser-relay-secret
```

正式 Compose 使用同一不可变镜像启动 Panel 和独立 Relay 容器，并把同一个 `KEJILION_PANEL_BROWSER_MODE` 同时传给 Panel 和 Relay。安装器通过 `--browser-mode disabled|beta` 接收模式，默认 `disabled`；同时生成 32 字节随机密钥，以 `root:kejilion-panel 0640` 只读挂载给两个容器。批准 Beta 例外后，部署配置才可显式启用 `beta`；HTTPS Beta 必须同时启用 Secure Cookie。Panel 与 Relay 必须同时升级，避免 Service Worker、Controller、WASM 和 Transport 版本不一致。

## 8. 上线门禁

发布前至少完成：

1. 在真实、受信任的双 HTTPS Origin 上验证 Service Worker 注册、更新和刷新，不使用公网 IP 明文 HTTP 代替。
2. 使用真实桌面 Chrome/Edge 进行静态站、服务端渲染站、SPA、搜索、表单、媒体和异常页测试，覆盖输入、点击、滚动、跳转、后退、前进、刷新和标签休眠恢复。
3. 验证 Google/Bing 搜索结果、GitHub/Baidu 等普通站点，以及 YouTube 页面和实际播放状态；未通过项必须写入已知限制。
4. 验证错误 Origin、无令牌/过期令牌、私网地址、DNS 重绑定、重定向到内网、无效 TLS 证书、超限请求和并发限流。
5. 完成持续负载、内存、CPU、重启、OOM、连接回收和服务器出口流量观测。
6. 复核固定依赖哈希、许可证、SBOM、漏洞扫描及 alpha 版本例外记录。
7. 验证默认 `disabled` 不签发浏览器会话、不放宽 Panel `frame-src`，只有显式 `beta` 才启用运行时。
8. 演练 v0.67.0 回滚，并验证旧内核不受客户端遗留 Service Worker 影响。

`docs/embedded-browser-core-154-acceptance.md` 记录的是 v0.66.0 Phase 1 历史验收，不能作为 v0.68.0 v2 运行时的上线证据。

## 9. 回滚

生产回滚点为上一稳定版本 `v0.67.0` 的不可变镜像和发布资产。浏览器内核变更没有数据库迁移，也不应删除 Panel 数据、站点、应用、Relay 密钥或业务容器。

回滚时：

1. 同时把 Panel 和 Relay 恢复到同一份 v0.67.0 不可变镜像，不能混跑 v2 Panel 与旧 Relay。
2. 按标准应用市场/部署回滚流程恢复镜像，复核 Panel、Relay、Agent 健康、重启次数和 OOM 状态。
3. Service Worker 是客户端持久状态，单纯回滚服务器镜像不会主动删除已注册 Worker。正式演练必须验证旧内核可正常接管；如出现干扰，应在同一 Relay Origin 提供明确的 Worker 退役流程，或切换到干净的 Relay Origin，并提示用户刷新站点数据。
4. 复测普通网页、错误提示和“使用系统浏览器打开”兜底，再恢复发布流量。

源码层面优先使用 `git revert` 撤销候选提交，不改写共享分支历史。发布标签和既有 Release 保持不可变。

紧急止损不必先回滚镜像：把 Panel 与 Relay 的模式同时改为 `disabled` 并重启两个容器，即可停止签发新会话，让 Relay 除 `/healthz` 外全部返回 503，并立即使已有令牌失去可用入口；随后再按上述流程回滚或排障。
