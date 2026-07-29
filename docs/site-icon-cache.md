# 网站图标派生缓存

## 目标与事实来源

网站列表可以显示实际站点 favicon，但图标不是网站配置事实，也不参与
`resourceVersion`。站点身份仍来自 `/home/web/conf.d` 等真实产物；缓存只保存展示用的
派生位图，不写回 `/home/web`，不修改 `kejilion.sh` 或 Nginx 配置。

请求链路：

```text
Browser <img loading=lazy fetchpriority=low>
  -> authenticated Panel GET /api/v1/sites/{id}/icon
  -> Agent GET /v1/sites/{id}/icon
  -> discovered site identity
  -> 127.0.0.1:80/443 local Nginx with original Host/SNI
  -> Agent private disk cache
```

## 安全边界

- 只接受当前 Agent 已发现且启用的 32 位站点 ID。
- 不按网站域名解析 DNS；Transport 无环境代理，实际 Dial 目标固定为
  `127.0.0.1:80/443`。
- `<link rel="icon">` 只允许当前站点的主域名和已发现别名；拒绝用户信息、显式端口、
  非 HTTP(S) 协议、跨域名和通配域名；跳过 `mask-icon` 及已明确声明为 SVG 等不支持
  类型的候选，避免无效下载。
- 禁止重定向；首页与图标响应分别限制为 `256 KiB`，响应头限制为 `32 KiB`。可能阻塞的
  站点发现等待、全局排队与网络抓取共用 `4 秒` 请求预算。
- 全局最多同时获取 4 个站点；同一站点的并发首次请求合并。
- 只接受经过魔数、结构和尺寸校验的 PNG、JPEG、GIF、ICO、WebP；拒绝 SVG、HTML 和
  超过 `2048×2048` / 约 400 万像素的位图。
- Panel 对 MIME、魔数和 `256 KiB` 上限再次校验；图标接口必须登录，不构成公开代理。

## 缓存与效率

- Agent 正缓存：7 天。
- Agent 明确缺失负缓存：24 小时。
- Agent 暂时失败负缓存：15 分钟。
- 浏览器成功缓存：7 天，带 `ETag`；明确缺失缓存 1 小时。
- 缓存过期刷新失败时继续返回上一次已验证的同站点图标。
- 缓存目录为 `${KEJILION_AGENT_STATE_DIR}/site-icons`，目录 `0700`、文件 `0600`；
  写入使用同目录临时文件、`fsync` 和原子替换。
- 每条内容有 SHA-256 完整性校验；站点主域名或 HTTP/HTTPS 身份变化后旧内容不会复用。
- 完整缓存最多保存 256 条、总位图约 32 MiB，超限按最旧获取时间清理；异常中断遗留的
  临时文件或无 metadata 图标老于一分钟后，会在后续缓存写入触发 `prune` 时清理，避免
  与正在完成的原子写入竞争；当前没有后台定时清理任务。

图标请求独立于 `/api/v1/sites` 列表，不阻塞页面数据加载。前端仅懒加载视口附近的图片，
失败后直接显示原有 `Globe2`，不进入重试循环；只有下一次网站列表成功刷新时才允许重新
尝试一次。

## 回归检查

- `internal/sites/icon_cache_test.go`：真实回环 Transport、持久化、TTL、负缓存、陈旧回退、
  同站点并发合并、全局并发上限、发现超时/失败退避、跨域与 SVG 候选拒绝、ICO 内嵌尺寸、
  格式/尺寸/容量及异常遗留文件边界。
- `internal/agent/site_icons_test.go`：鉴权、精确路由、错误映射和媒体限制。
- `internal/panel/site_icons_test.go`：Session、Agent 转发、二次校验、`ETag`、304 和缓存头。
- `web/src/components/sites/SiteFavicon.test.ts`：同源 URL、低优先级懒加载和 Globe 降级。
- `.codex-workflows/kpanel-site-icon-cache-validation.workflow.yaml`：隔离候选和真实浏览器复验。
