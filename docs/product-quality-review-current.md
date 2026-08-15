# KPanel 当前业务事实与规范适配基线

- 复核日期：`2026-08-15`
- 基线提交：`431b9ca5ac21d5a3d1629e46e209ce718367569f`
- 基线版本：`v0.74.0`
- 上一份完整复核：[`product-quality-review-2026-08-13.md`](product-quality-review-2026-08-13.md)
- 自动刷新门槛：基线后达到 50 个提交，或同时达到 20 个提交和 8 个正式版本；产品性质、业务真源、权限边界或核心旅程发生实质变化时立即复核

本文件是 AI 会话定位当前产品形态的稳定入口。历史复核保留对应时点证据；这里不重复完整质量规范，
只记录会影响任务分级、设计判断和验收范围的当前业务事实。

## 当前产品性质与核心思想

1. KPanel 仍是面向单管理员、直接管理 Linux 宿主机真实资源的轻量控制面，不是多租户 SaaS。
2. `kejilion.sh`、Docker Engine、Nginx、systemd 和实际文件仍是业务真源；Web、脚本、SSH 与 Compose
   必须能继续管理同一资源，Panel 数据不得形成排他第二真源。
3. Panel 保持无特权，宿主机写操作只经 Unix Socket 上的 Agent 固定动作或登记的脚本协议执行；
   `kejilion.sh` PTY、目标容器内有界控制台与宿主机任意 Shell 必须继续区分。
4. 低配主机资源预算、长任务恢复、真实失败状态、管理员完整能力和脚本/Web 双向互通仍是产品能力。
5. 0.x 阶段继续允许聚焦的小步高频交付；风险越高证据越完整，不按文件数、版本数或等待时长机械加码。

## 相对 v0.69.0 的主要变化

- 桌面工作区已从视觉入口扩展为可持久化的图标、文件/目录快捷方式、批量操作、外部文件拖入、
  跨窗口文件传输和目录窗口复用，新增真实文件状态、并发版本与回滚边界。
- 文件管理器与桌面共享文件旅程；窗口关闭、批量失败、外部拖入和文件摘要一致性成为受影响验收重点。
- 桌面新增浏览器全屏、框选和自定义图标布局；导航嵌套路由、浅/深主题、窄视口、键盘/焦点和
  浏览器拒绝态继续按受影响范围验证。
- 应用市场新增官方动态目录：只读拉取固定来源并校验 schema、数量、唯一标识、selector 和 URL；
  不可达或越界时保留内置安全目录。新增 selector 仅可调用满足属主、权限和协议约束的宿主脚本，
  旧脚本不支持时明确失败，不回退执行远端命令。
- 发布与治理已采用风险感知 `verify-change`、后台浏览器作业、依赖新鲜度检测和环境用途门禁；
  `prod-108`/`108` 禁用全部 KPanel 操作，隔离验证和唯一正式部署默认使用 `arena-154`。

这些变化扩大了桌面、文件和应用市场业务域，但没有改变 KPanel 的产品性质、业务真源、Panel/Agent 权限边界、
低配主机定位或“不以安全名义削减管理员能力”的核心思想。

## 当前业务域与最小验收入口

| 业务域 | 代码真源 | 风险感知验收重点 |
| --- | --- | --- |
| 身份、Panel/Agent | `internal/auth`、`internal/panel`、`internal/agent` | Session、CSRF/Origin、Unix Socket、离线降级 |
| 系统、网站、应用 | `internal/systemmanage`、`internal/sites`、`internal/appmarket` | 脚本同源、动态目录来源/回退、selector 边界、真实状态、后台任务、失败恢复 |
| Docker、文件 | `internal/dockerx`、`internal/filemanager` | 外部资源、路径/归档、资源版本、批量部分失败 |
| 桌面工作区 | `internal/desktopworkspace`、`web/src/components/desktop` | 持久化、窗口/快捷方式互通、并发版本、跨窗口传输 |
| 集群、终端、AI | `internal/cluster`、`internal/terminal`、`internal/ai` | 协议身份、断线恢复、输入输出上限、固定工具 |
| 全局体验 | `web/src/router.ts`、`web/src/views`、`web/src/styles` | 受影响旅程、双视口、主题、焦点、多语言、失败态 |

日常任务先从本表定位受影响域，再执行 `make verify-change`；只有跨权限、协议/数据、部署或发布风险
才升级 L2/L3。当前事实入口的新鲜度由 `scripts/check-business-context-freshness.mjs` 以本地 Git 数据检查，
达到门槛只要求刷新业务事实，不自动推断质量下降，也不触发发布或生产操作。
