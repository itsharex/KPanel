# KPanel 轻量安全浏览器内核

- 状态：第一阶段安全阅读内核，已进入 v0.66.0 独立发布候选与生产门禁
- 更新：2026-08-12
- 目标：允许管理员访问任意公网 HTTP(S) URL，同时不把第三方站点直接嵌入 KPanel

## 1. “任意 URL”的准确含义

内核接受语法有效、无用户信息、长度不超过 2048 字节的公网 `http://` 和 `https://` URL。
HTTP 目标由 Relay 在服务端访问，因此 KPanel 使用 HTTPS 时也不会产生浏览器混合内容请求。

默认拒绝以下目标：

- `localhost`、`.local`、`.internal`、`.home.arpa`
- 回环、私网、链路本地、组播、文档网段及其他特殊用途 IPv4/IPv6 地址
- DNS 同时返回公网与非公网地址的主机
- 非 HTTP(S) 协议、带用户名或密码的 URL、非法端口和非浏览器规范化主机名

因此，“任意 URL”不等于允许访问宿主机、内网服务、云元数据地址或本地文件。未来如需访问私有站点，必须通过显式 CIDR/域名白名单单独设计，不能关闭默认 SSRF 防护。

## 2. 信任边界与数据流

```text
KPanel 页面
  ├─ POST /api/v1/browser/sessions
  │    └─ 获得 10 分钟短时签名令牌
  └─ iframe src=https://独立Relay域名/kernel/
       └─ postMessage(令牌, 目标URL；精确校验双方 origin)
            └─ Relay /v1/fetch
                 ├─ URL/DNS/IP/端口校验
                 ├─ 每次新连接重新解析并固定到已校验 IP
                 ├─ 并发、请求体、请求头和空闲超时限制
                 └─ 流式访问公网目标
                      └─ Kernel 净化 HTML、代理图片、隔离渲染
```

KPanel 的 iframe 永远只加载 Relay 的 `/kernel/`，不会出现 `iframe src="https://第三方站点"`。令牌不进入 URL、日志或持久化存储。Relay 必须使用与 KPanel 不同的 HTTP(S) origin。

## 3. 已实现的安全与稳定性控制

| 控制面 | 当前实现 |
| --- | --- |
| 身份 | HMAC-SHA256 短时令牌，默认 10 分钟，绑定随机浏览器会话 ID |
| SSRF | 公网地址策略、全 DNS 结果检查、连接时二次解析并固定已验证 IP |
| Origin | `/v1/fetch` 只接受规范化后的 Relay 自身 origin；KPanel origin 只用于 frame ancestor 与消息校验 |
| 隔离 | 独立 Relay origin；外层 iframe 只开放 `allow-same-origin allow-scripts`；内容 iframe 不开放脚本 |
| 内容 | 删除脚本、样式、外链样式、表单、iframe、对象、SVG/MathML 及事件属性；链接重新走内核 |
| 响应头 | 上游 `Set-Cookie`、CSP、X-Frame-Options 等不进入 Relay HTTP 响应头，只作为有界元数据交给内核 |
| 网络 | 不继承环境代理；HTTP/2；连接、TLS、响应头和响应体空闲超时；禁止自动重定向 |
| 并发 | 默认全局 24、每会话 6；KPanel 前端最多 8 标签、2 个活动内核文档、45 秒休眠 |
| 内存预算 | HTML 8 MiB；单个二进制 16 MiB；单图 2 MiB；每页图片合计 12 MiB；图片并发 4 |
| 流式 | Relay 使用 32 KiB 复用缓冲区；Kernel 使用 `ReadableStream` 有界读取，超限立即取消 |

## 4. 当前兼容性边界

第一阶段优先验证安全边界和轻量传输，当前属于“安全阅读/预览内核”，不是完整 Chromium 替代品。

已支持：

- 普通 HTML 内容、文本、表格、链接和有限数量图片
- 同站及跨站 HTTP(S) 跳转，最多 5 次重定向
- 图片和小型二进制资源通过 Relay 加载
- 标签休眠、短时会话自动刷新、系统浏览器兜底

尚未支持：

- 执行第三方 JavaScript、现代 SPA hydration、Service Worker、WebSocket
- 原站 CSS 布局、视频/音频流、下载管理
- 表单提交、登录态、Cookie Jar、LocalStorage/IndexedDB
- CSP/SRI/ES Module/CSS URL 的完整重写语义

这些限制会明显影响复杂网站体验。因此在完成第二阶段“资源图谱重写”和第三阶段“受限脚本运行时”前，不能把本 MVP 宣称为任意现代网站的完整兼容浏览器，也不应移除“用系统浏览器打开”的兜底入口。

## 5. 后续兼容层路线

1. 资源图谱：HTML/CSS URL 解析与重写、字体/媒体分段流、缓存和请求去重。
2. 会话状态：Relay 侧按短时会话隔离 Cookie Jar，执行 Public Suffix 校验并设置总量预算。
3. 受限运行时：在独立 origin 内实现脚本 URL/API 重写，覆盖 `fetch`、XHR、Worker 和 WebSocket；默认按站点能力降级。
4. 兼容性门禁：建立静态站、传统服务端站、登录站、SPA、媒体站五类回归样本和长稳压测。

任何阶段都不得回退为直接加载第三方 iframe，也不得把任意 URL 网络出口移回 Panel 进程。

## 6. 运行配置

Panel 需要同时设置：

```text
KEJILION_PANEL_BROWSER_RELAY_URL=https://browser.example.com
KEJILION_PANEL_BROWSER_RELAY_SECRET_FILE=/run/secrets/browser-relay-secret
```

Relay 使用同一密钥文件启动：

```text
kpanel-browser-relay \
  -listen :8090 \
  -allowed-origin https://panel.example.com \
  -public-url https://browser.example.com \
  -secret-file /run/secrets/browser-relay-secret
```

正式 Compose 以同一不可变镜像启动独立 Relay 容器，限制为 0.5 CPU、128 MiB、64 PIDs、只读根文件系统、无 Linux capabilities。安装器生成 32 字节随机密钥并以 `root:kejilion-panel 0640` 只读挂载给两个容器。`browser.example.com` 应由独立反向代理转发到 Relay 并保持流式响应；不能与 `panel.example.com` 共用 origin。

## 7. 本地验证与性能基线

2026-08-12 在 Windows amd64、Intel i5-12600KF、Go 1.26.5 上验证：

- `go test ./internal/browsercore ./cmd/kpanel-browser-relay`：通过
- Panel 浏览器会话与配置定向测试：通过
- `go vet`（browsercore、Relay、Panel）：通过
- 前端类型检查、i18n 检查、生产构建：通过
- 浏览器核心与桌面浏览器组件：14 项测试通过
- 64 KiB 内存上游 Relay 微基准：约 25.1–25.5 µs/op、约 12.1 KiB/op、102 allocs/op
- 并发限制器微基准：多数约 87 ns/op、64 B/op、2 allocs/op；一次运行出现 202 ns/op 抖动

微基准只测进程内策略、令牌、请求封装和流式复制，不包含公网 DNS、TCP/TLS、目标站延迟和浏览器 DOM 解析，不能当作真实页面加载耗时。真实体验主要由目标站 RTT、资源数量及第二阶段重写能力决定。

完整 `go test ./...` 在 Windows 上仍有仓库既有的 Linux/path/静态资源假设失败；本次新增包和定向测试通过。`go test -race` 因当前 Windows Go 环境未启用 CGO 而未执行，合并前应在 Linux CI 运行竞态检测及 30–60 分钟并发长稳测试。
