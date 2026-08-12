# KPanel v0.68.0 发布验收记录

日期：2026-08-12

发布级别：L3 / Browser Beta 灰度

正式提交 / 源码树 / 标签：`07b424cd46e0f43ed2ab6e65289b3608334142c6` / `3e9f4fa70d9c03fef6d335c764d31c88179897d1` / `v0.68.0`

上一稳定版本 / 回滚点：`v0.67.0` / `6aaabeacd38e2d7b4c51e70b3aeff82db5523866` / OCI index `sha256:f90c4c667eb180b407dccec4901c48d2cf691578fb7abdcfac92a72df7c32b03`

## 发布范围与边界

- 内置浏览器升级为上下文保持型 v2 重写运行时。Panel 负责短时会话，第三方公网 HTTP(S) 内容由独立 Relay Origin 获取与重写，Service Worker、Controller、WASM 和重写后的第三方脚本均停留在 Relay Origin。
- 支持普通 HTML/CSS/JavaScript、点击、滚动、地址与标题同步、前进/后退/刷新、空/非空 POST、`data:`/`blob:` Controller 路径和 16 KiB URL 边界；保留系统浏览器入口。
- 产品不嵌入 Chromium、Chrome、Electron 或测试 Runner。`@mercuryworkshop/scramjet@2.0.67-alpha.2` 仍按限期 Beta 例外管理，复核日期为 2026-09-11，不描述为稳定依赖。
- Browser Beta 在产品和应用市场默认 `disabled`；本次 154 灰度实例在双 HTTPS、Secure Cookie、安全边界和回滚门禁通过后显式配置为 `beta`。

## 候选冻结与自动门禁

- 线性基线为 `f3a7d17a8172d9f77de7832d07a5e660c336115c`，冻结候选相对基线前进 6 个提交；正式 Tag 精确指向上述源码提交，没有从其他脏工作树拼装内容。
- bundle：`C:\GitHub\_release-artifacts\v0.68.0-browser\kpanel-v0.68.0-07b424c.bundle`，SHA-256 `296ee2e4c9b36159bda88b08312556af9f9d33d3ac0937b44a119d3d0bf483e0`。
- 完整 release L3、Go 全量、核心包 race、`go vet`、Web 638 项、i18n、typecheck、生产构建、Linux amd64/arm64 构建、部署安全、依赖策略、`govulncheck`、`npm audit` 和 Trivy 均通过；最终镜像已知漏洞为 0。
- 候选 CI `31602770258`、候选 Dependency freshness `31602770328`、主线 CI `31603021017`、主线 Dependency freshness `31603020967`、Release `31603369079`、Tag Dependency freshness `31603368984` 全部成功。

## Browser Beta 真实门禁与分类

- 精确候选在 154 双 HTTPS Origin 的真实 Chromium 中完成 Service Worker 3/3 激活、基本导航、Bing、百度、GitHub、空/非空 POST、`data:`/`blob:` 和 16 KiB URL 边界；四 Context 稳定性运行 30 分钟，132/132 动作成功，页面崩溃 0，Panel/Relay restart 0、OOM false。
- 原始 headed 包：`C:\GitHub\_release-artifacts\v0.68.0-browser\e2e-07b424c-headed-r4.tar.gz`，SHA-256 `785cc9e8cc3508465d525db2035d61220225575fe2b3bd9e2a6c9c1b9706480d`。原始 trace/report 可能含短期会话信息，保持 restricted，不进入公开仓库或 Release。
- `product-hard-failure`：0。SSRF/内网阻断、DNS 重绑定、标准 TLS 验证、双 Origin、会话 Token、CSP、请求/响应/URL 预算、并发限制、容器资源边界和 kill switch 未放宽。
- `upstream-policy`：Google 搜索和 YouTube 媒体未达到完整浏览体验，属于上游 429/反滥用、媒体与第三方策略兼容性观察；Reddit 的 reCAPTCHA 信号、知乎登录重定向等同类现象不作为产品硬故障。发布不宣称 Google/YouTube 完整支持，也不做站点特判。
- `test-assertion`：旧 evaluator 仍按固定 13/13、Runner 退出码和较高采样频率输出红色；其中外站两项已按新门禁归入 `upstream-policy`，资源采样窗口断言属于测试频率问题。独立 Host 监控覆盖完整 soak，产品容器的资源阈值、健康、重启、OOM 和 fatal 检查均通过，因此不阻断 Beta 灰度。
- 常用站补充验收中，Wikipedia、MDN、npm、Stack Overflow、Hacker News、Bilibili、JD 通过；Microsoft 页面可用但 readiness 断言超时；知乎按站点策略进入登录页。未出现新的确定性白屏、产品主链路持续 5xx、产品进程崩溃或资源失控。

## Release、OCI 与 apps

- [GitHub Release v0.68.0](https://github.com/kejilion/KPanel/releases/tag/v0.68.0) 已公开，非 draft、非 prerelease；annotated tag object 为 `21780454713b0c75864eeb04a1a2c26e31c240db`，peel 后精确指向正式提交。
- `docker.io/kjlion/kejilion-panel:0.68.0` 与 `latest` 均指向 OCI index `sha256:e65cd6c94c405539b7d6bb359aee99793ffcbb54a0977bf8f52c9fbfe1ee3ac4`。
- `linux/amd64` 子清单为 `sha256:2e2dcf4e87ecd523aab43d8400221a5d03659e049e056c4975e234b9b6b109a4`；`linux/arm64` 子清单为 `sha256:979a94598c2f4383528a190c470dbd2e580981fc5eb485f5f8aef9db87819546`；两个架构的 OCI revision/version 均精确为正式提交和 `0.68.0`。
- 154 从 Docker Hub 重新拉取公开镜像后，公开镜像 E2E 输出 `image_e2e=pass`。
- apps 提交为 `5f0c9ec869715a89238fb372c867b6930f110fb8`；`kpanel.conf` SHA-256 为 `5d80736946bfedfa8ee7dc9b1fecbde23ca359766e4e2308bc4b1eae7a122822`，Linux 语法和一次性安装/更新/卸载 lifecycle 均通过。

## 154 备份、HTTPS 与生产升级

- 唯一生产目标为 `arena-154`。Panel Origin 为 `https://kpanel.154.36.153.9.sslip.io`，Relay Origin 为 `https://kpanel-browser.154.36.153.9.sslip.io`；没有配置、接管或使用 `kp.kejilion.pro`。
- 停写一致性备份：`/root/kpanel-backups/v0.68.0-preupgrade-20260812T141505Z.tar.gz`，SHA-256 `cd62333e7853529d94c81db2a11df082b49baf884b0a3753342c597c5778c89f`。独立解包后 SQLite、JSON、JSONL、清单、权限和文件摘要均通过恢复校验。
- HTTPS 变更前备份：`/root/kpanel-backups/v0.68.0-https-prechange-20260812T142234Z.tar.gz`，SHA-256 `d7cf4766f5288558a33342bdfa8da5bc4e21decbf47111b4ada8b279f0cbc16a`；Nginx 配置、证书和防火墙快照可恢复。
- 使用应用市场标准更新入口完成升级，没有触发自动回滚。生产 Panel/Relay 使用同一不可变 OCI，均为非 root、只读根、受限 CPU/内存/PIDs，Agent 保持 active。
- `.env` 精确配置双 HTTPS、`KPANEL_SECURE_COOKIE=true` 和 `KPANEL_BROWSER_MODE=beta`；Panel CSP 的 `frame-src` 只加入 Relay Origin，Relay CSP 的 `frame-ancestors` 只允许 Panel Origin，Service Worker 使用 `text/javascript`、`no-cache` 和 `Service-Worker-Allowed: /`。
- 完整生产复核通过：Panel `0.68.0/ok/v1alpha1`、Relay `ok/kpanel-browser-core/v1`、两个容器 healthy/0 restart/OOM false、Agent active/0 restart；镜像 revision/version、apps、环境契约、CSP、HTTPS、固定脚本与数据完整性均一致。末次数据检查为 2 个 SQLite、5 个 JSON、13 个 JSONL、15877 条有效记录。

## Kill switch 与回滚演练

- 在生产执行可逆 kill switch 往返：切换 `disabled` 后 Relay `/kernel/` 精确返回 503，Panel/Relay 继续 healthy；恢复 `beta` 后内核恢复 200，双 HTTPS 与正式配置恢复正常。切换窗口内已打开的浏览器页会保留错误态，点击“重新加载”后重新申请会话。
- 使用新停写备份和上一稳定 OCI 在 154 独立容器、独立网络、仅本机测试端口完成隔离回滚演练；恢复数据通过 SQLite/JSON/JSONL 校验，v0.67.0 Panel/Relay 成功启动、restart 0、OOM false。演练全程没有替换正式数据或正式容器，清理后生产仍为 v0.68.0 healthy。
- 回滚证据目录：`/root/kpanel-release-evidence/v0.68.0/production-20260812T141505Z/rollback-drill`。
- 即时止损优先将 `KPANEL_BROWSER_MODE=disabled` 并重建 Panel/Relay；完整回滚则停止 Panel/Relay/Agent，恢复停写备份、apps 配置和 systemd 链路，固定上一 OCI index `sha256:f90c4c667eb180b407dccec4901c48d2cf691578fb7abdcfac92a72df7c32b03`，再复核 v0.67.0、数据、Agent、HTTPS、重启和 OOM。

## 生产持续观察

- 从 `2026-08-12T14:59:29Z` 至 `15:36:19Z` 连续观察 36 分 50 秒，共完成 121 个双容器样本；本机/公网 Panel、Relay、Agent、容器健康、重启/OOM 和 fatal 日志门禁全部通过。
- Panel 内存最小/最大/平均为 11.07/12.50/11.87 MiB，末 20 样本增量 -0.01 MiB，最大 8 PIDs；Relay 为 4.88/7.00/5.52 MiB，末 20 样本增量 +0.01 MiB，最大 8 PIDs。两者均 0 restart、OOM false、fatal 0，没有单调资源泄漏。
- 观察期间执行一次真实浏览器重新加载，Relay 工作集从约 4.9 MiB 上升到约 7.0 MiB 后持平，属于处理活动后的正常常驻工作集，不接近 128 MiB 限额。
- 生产观察证据：`/root/kpanel-release-evidence/v0.68.0/production-20260812T141505Z/observation-30m`。

## 补丁规则、限制与沉淀

- `v0.68.0` Tag、Release 和版本镜像保持不可变。后续只有确认的产品缺陷才整理为独立补丁版 `v0.68.1`；不因上游 CAPTCHA/429/登录策略或测试断言误报改写现有版本，也不把无关功能混入补丁。
- Google/YouTube、复杂 SSO、DRM、WebSocket、第三方登录弹窗、完整下载管理和媒体边缘语义继续属于 Beta 兼容性观察项；遇到上游挑战必须显示可理解状态并保留系统浏览器入口，不允许静默白屏。
- 本轮复用并完整执行 `release-kpanel v1.7` 与项目版本治理流程，没有新增重复工作流；发布事实、精确摘要、备份和回滚证据沉淀在本文件中。
