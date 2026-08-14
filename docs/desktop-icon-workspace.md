# KPanel 桌面图标工作区

- 状态：已实现，待 L3 上线审批
- 适用范围：宽屏桌面图标布局、已安装应用与网站入口显隐、站点别名、自定义快捷方式
- 业务真源：静态桌面入口、应用市场运行时清单、Agent 网站清单
- 持久化：Panel 本地有界工作区；不写 Agent，不复制或删除宿主机真实资源

## 1. 目标与非目标

管理员可以在宽屏桌面拖动图标、交换位置或自动排列；刷新、重新登录和 Panel 重启后
恢复布局。已安装应用和网站可以仅从桌面隐藏并再次恢复；自定义快捷方式支持名称、描述、绝对
HTTP(S) URL 和可选本地图标的创建、编辑与删除。

本期不提供文件夹、多选框选、像素级自由重叠、多人独立桌面、跨主机同步或远程图标抓取；不改变
经典模式、应用安装状态、网站配置、Nginx、Docker 或 `kejilion.sh` 产物。桌面图标落点吸附现有
网格；超过单页容量时扩展纵向可滚动工作区，不隐藏、截断或重叠入口。

## 2. 真源与删除语义

桌面合并四类入口：

1. `desktopApps` 提供固定系统入口，稳定键为 `nav:<route>`；
2. 应用市场 inventory 提供已安装应用、状态、名称、URL 和市场图标，稳定键为 `app:<id>`；
3. sites list 提供已启用网站、域名、URL 和站点图标，稳定键为 `site:<32 lowercase hex>`；
4. 工作区保存自定义快捷方式，布局稳定键为 `shortcut:<32 lowercase hex>`。

应用与网站先按既有规则去重，再应用工作区的显隐、站点别名和位置。工作区不保存应用或网站的
URL、图标、安装状态等快照；真源变化后桌面刷新应采用最新值。暂时未发现的来源不主动删除其位置，
来源重新出现后仍可恢复。

“从桌面移除”只把 `app:<id>` 或 `site:<id>` 加入 `hiddenEntryKeys`，绝不调用应用卸载、网站删除、
容器、Nginx 或宿主机写入接口；固定 `nav:` 入口和 `shortcut:` 不可加入隐藏集合。管理窗口负责恢复
隐藏入口。删除快捷方式只从工作区移除对应 shortcut 及 `shortcut:<id>` 位置，metadata 提交后由
服务端回收其无引用图标。

## 3. Workspace v1

`GET /api/v1/desktop/workspace` 的响应结构如下；`iconVersion` 和 `iconURL` 仅在快捷方式已有合法图标时
返回，`warning` 仅在存储不可用时返回：

```json
{
  "schemaVersion": 1,
  "resourceVersion": "sha256:<64 lowercase hex>",
  "available": true,
  "hiddenEntryKeys": ["app:thirdparty-example"],
  "positions": {
    "nav:/overview": { "x": 0, "y": 0 },
    "shortcut:0123456789abcdef0123456789abcdef": { "x": 0.25, "y": 0.5 }
  },
  "labels": {
    "site:0123456789abcdef0123456789abcdef": "我的网站"
  },
  "shortcuts": [
    {
      "id": "0123456789abcdef0123456789abcdef",
      "name": "管理入口",
      "description": "自定义说明",
      "url": "https://example.com/path",
      "iconVersion": "<64 lowercase hex>",
      "iconURL": "/api/v1/desktop/shortcuts/0123456789abcdef0123456789abcdef/icon?v=<version>",
      "createdAt": "2026-08-14T00:00:00Z",
      "updatedAt": "2026-08-14T00:00:00Z"
    }
  ]
}
```

全量 `PUT /api/v1/desktop/workspace` 请求只包含可写业务字段：

```json
{
  "expectedResourceVersion": "sha256:<64 lowercase hex>",
  "hiddenEntryKeys": [],
  "positions": {
    "nav:/overview": { "x": 0, "y": 0 }
  },
  "labels": {},
  "shortcuts": [
    {
      "id": "0123456789abcdef0123456789abcdef",
      "name": "管理入口",
      "description": "自定义说明",
      "url": "https://example.com/path"
    }
  ]
}
```

`positions` 仅保存宽屏工作区内的逻辑坐标：`x ∈ [0,1]`，`y ∈ [0, MaxPositions]`。横坐标在当前
可用宽度内归一化；纵坐标以当前可视区的纵向可移动距离为归一化单位，不足一个网格步长时取一个
步长。整数和小数共同表示连续的跨页偏移，`y > 1` 的位置位于首屏以下，因此不受单页网格容量限制。
服务端拒绝 NaN、Infinity、越界值及超过位置总量上限的请求。

允许的位置键为 `nav:`、`app:`、`site:` 和已存在的 `shortcut:`；`labels` 只接受 `site:` 键，且只
改变桌面显示名，不修改网站真源。`resourceVersion` 由规范化 metadata 计算，不包含图标内容；快捷
方式时间戳由服务端维护，不由客户端提交。请求采用严格 JSON，未知字段、重复或非法稳定键、悬空
shortcut 位置均拒绝。

| 项目 | v1 上限 |
| --- | ---: |
| 自定义快捷方式 | 64 个 |
| 隐藏入口 | 512 个 |
| 宽屏位置 | 512 个 |
| 工作区编码后大小 | 256 KiB |
| 名称 / 描述 | 48 / 160 个 Unicode 字符 |
| URL | 2048 字节 |
| 单张图标 / 图标总量 | 256 KiB / 16 MiB |
| 图标边长 / 总像素 | 单边最大 1024 px / 最大 100 万像素 |

现有 `kpanel:desktop-site-names:v1` 只做一次迁移：将仍有效且服务端尚无值的站点别名合并进
`labels`，全量 PUT 成功后才删除旧 localStorage 键；冲突或写入失败时保留旧值，避免丢失或重复覆盖。

## 4. API、权限与并发

当前接口为：

```text
GET    /api/v1/desktop/workspace
PUT    /api/v1/desktop/workspace
GET    /api/v1/desktop/shortcuts/{id}/icon[?v=<64 lowercase hex>]
PUT    /api/v1/desktop/shortcuts/{id}/icon
DELETE /api/v1/desktop/shortcuts/{id}/icon
```

- 全部接口要求有效 Session；PUT/DELETE 还要求同源 Origin 和 CSRF。错误方法、路径、RawPath 与未允许
  的查询参数必须被拒绝。
- workspace PUT 是全量替换，必须携带当前 `expectedResourceVersion`；版本不匹配返回 `409`。前端将
  连续写入串行化，每次基于最新已确认快照提交；冲突时重载服务端胜者并向用户报告失败，不自动重放。
- 图标 PUT 的 body 是原始二进制，不是 JSON 或 multipart；`Content-Type` 只能是 `image/png`、
  `image/jpeg` 或 `image/webp`。DELETE 无请求体。图标写入与 workspace `resourceVersion` 相互独立，
  图标端点当前不使用 `expectedResourceVersion`。
- 创建或编辑带图快捷方式是两阶段操作：先全量 PUT metadata，再按需 DELETE/PUT 图标并重新 GET
  workspace。图标阶段失败时，已提交的快捷方式或 metadata 保留，编辑窗显示错误并允许重试；不得
  宣告整个操作成功，也不得把该边界描述为跨文件原子事务。
- 自定义 URL 前后端均解析校验，只允许无 userinfo 的绝对 `http:`/`https:` URL；拒绝相对地址、
  控制字符、无 host、非法端口及 `javascript:`、`data:`、`file:`、`blob:` 等协议。localhost 与
  私网地址可作为管理员浏览器的跳转目标，但 Panel/Agent 不访问目标，不形成 SSRF。
- URL 入口继续复用跳转确认窗并使用安全的新窗口打开策略。workspace 和图标变更均写审计；审计只
  记录入口 ID、类型、字节数或变化计数，不记录完整 URL、查询参数、描述和本地文件名。

## 5. 布局与交互

`.desktop__icons` 的一页可视区域按容器实际宽高和现有单元尺寸计算网格，不使用窗口坐标猜测。
工作区允许纵向滚动并按相同网格步长连续扩展；水平坐标始终夹取在当前页内，纵向坐标可落到后续
可视区。拖到已占网格时交换位置；所有可见键在整个滚动工作区必须对应唯一显示位置，超过单页容量
的入口继续排列到下一页，绝不能回退到 `(0,0)`、重叠或变为不可访问。

最多 512 个入口拥有可持久化位置，并分配至全局首个空位。自动排列在此上限内按当前入口顺序逐页
执行 column-major：先在本页按列从上到下、再从左到右填满唯一网格位，然后进入下一页。可见入口
超过 512 个时，超出部分进入相互不重叠的临时区域，仍可打开、隐藏或删除，但不能拖动、键盘移动或
参与自动整理；自动整理应停止并提示限制，不发 PUT、不报告成功。滚动偏移不属于 workspace，不得
误写为图标位置。

- `window.innerWidth > 760`：读取并保存唯一一套宽屏 `positions`；`x` 映射到当前页可用宽度，`y`
  映射到连续纵向页面。视口变化后重新计算每页行列与唯一占位，但不得让溢出键重叠。
- `window.innerWidth <= 760`：始终按入口顺序逐页派生紧凑滚动布局，不读取为独立位置档位，也不把
  派生结果回写。拖拽、键盘移动和自动排列在此宽度禁用；返回宽屏后恢复原宽屏位置。

鼠标按下时只记录拖拽候选，不得立即调用 Pointer Capture。移动距离达到 6 px 后才进入拖拽并获取
pointer capture；阈值内的普通 `click`、`dblclick` 必须继续以子按钮为事件目标，分别完成选择和打开。
成功拖拽后短时抑制 click/dblclick，避免放下即打开。`pointercancel`、`lostpointercapture`、第二
指针和窗口失焦取消本次预览，不提交位置。拖动到可视区上下边缘时应有界自动滚动，并用工作区坐标
计算跨页落点。

触摸与手写笔保留单击打开和长按菜单；宽屏下移动达到 12 px 后直接拖动，无需额外整理模式。
键盘聚焦图标后使用 `Ctrl/Command + Arrow` 移动或交换，Enter/Space 打开，ContextMenu 或
Shift+F10 打开菜单；位置变化通过 `aria-live` 宣告。可见焦点、上下文菜单与减少动画偏好不得退化。

## 6. 图片与持久化安全

服务端先用 `http.MaxBytesReader` 限制原始请求体，再同时校验声明 MIME、魔数、实际解码格式、边长和
像素数；PNG、JPEG、WebP 以外的 SVG、GIF、HTML、伪 MIME、坏图和像素炸弹全部拒绝。服务端忽略
客户端文件名和扩展名，不提供远程图标下载。

工作区保存在 `${DataDir}/desktop-workspace/workspace.json`，图标保存在同目录 `icons/<id>.icon`；
目录权限为 `0700`，文件权限为 `0600`。图标文件名由已校验 shortcut ID 决定，`iconVersion` 是内容
SHA-256 的 64 位十六进制值。图标 GET 要求 Session，返回 `ETag` 和 `nosniff`；带匹配 `?v=` 时使用
private immutable 缓存，否则使用 private no-cache。

workspace 全量 PUT 通过同目录临时文件、`fsync` 和原子重命名提交，metadata 重命名是提交点；成功后
再尽力回收无引用图标。图标 PUT 单独原子替换 `<id>.icon`，图标 DELETE 是独立删除操作。启动和每次
metadata 提交后清理遗留临时文件与无引用 `.icon` 文件，但不删除其他任意文件。并发图标写入没有
metadata 乐观锁，最终内容以最后完成的成功原子替换为准，不能影响 workspace metadata。

缺少 workspace 文件时创建空 v1 配置。文件损坏、未知 schema、超限或非法结构时保留现场，并以
`available:false`、`warning:"desktop_workspace_unavailable"` 返回空工作区；桌面继续使用真实来源和
默认布局，后续 workspace 与图标写入返回 `503`，不得覆盖损坏文件。核心登录、应用、网站和经典模式
不得受影响。

代码回滚只需恢复上一稳定提交；旧版本不会读取独立工作区，新版本再次启用时仍可读取原数据。如需
回退数据，应停写并成对备份/恢复 `workspace.json` 与 `icons/`，再校验权限、JSON、快捷方式与图标
对应关系，不能只恢复一侧。

## 7. 资源与兼容门禁

桌面首次加载只新增一次 workspace GET，并与 inventory/sites 并行；应用、网站入口不得各自新增
metadata 请求。自定义图标按实际显示懒加载并使用版本缓存。拖拽预览与纵向滚动期间不发网络或磁盘
请求，只在放下、自动排列、显隐或编辑确认后提交。最大数据集下需验证 256 KiB metadata、
512 个位置、64 个快捷方式和 16 MiB 图标配额不会造成图标重叠、明显主线程卡顿、无界内存增长或
磁盘写放大。

支持当前项目浏览器基线的 Pointer Events、`crypto.randomUUID` 或兼容降级 ID 生成；服务端仍对所有 ID
做最终校验。功能保持独立 schema 和目录，避免旧版本整体写回其他状态文件时丢失新字段。候选需完成
L3 门禁并获得上线授权后，才能合并 `main`、打标签、发布镜像或部署生产环境。

## 8. L2 验收矩阵

| 维度 | 必须验证 |
| --- | --- |
| 业务真源与删除 | 隐藏应用/网站后 inventory、网站配置、容器、Nginx 和真实数据均不变；管理窗口可恢复；来源名称、URL、图标变化后刷新采用新真源；固定入口不可删除 |
| Workspace 契约 | GET/全量 PUT 字段与类型一致；仅接受 `nav:`、`app:`、`site:`、`shortcut:` 位置键；shortcut 位置必须有对应记录；`labels` 仅接受 `site:`；`x ∈ [0,1]`、`y ∈ [0,MaxPositions]`，NaN/Infinity/越界及各容量上限校验生效 |
| 自定义快捷方式 | 无图回退、创建、编辑、描述、跳转确认、删除、64 个上限；删除 metadata 后图标回收；图标阶段失败时 metadata 状态明确、编辑窗不报假成功且可重试 |
| URL 安全 | 危险协议、userinfo、控制字符、相对 URL、无 host、非法端口和超长 URL 被拒绝；私网 URL 仅由浏览器确认后打开，服务端零出站请求 |
| 图片安全 | raw binary PUT 成功；multipart、SVG、HTML、GIF、伪 MIME、坏图、像素炸弹、超 1024 边长、超 100 万像素、超 256 KiB 和总配额超限均拒绝；未登录读取失败；`ETag`/304 与版本缓存正确 |
| 权限与审计 | 未登录 GET/PUT/DELETE 被拒绝；缺 Origin 或 CSRF 的写入被拒绝；错误方法、RawPath、查询参数、Content-Type 和未知 JSON 字段返回明确 4xx；workspace 与图标变更有审计且不泄漏完整 URL/描述/文件名 |
| 并发 | 两标签基于同一 workspace 版本全量 PUT 只有一方成功、另一方 409 并重载远端胜者；快速连续保存按确认顺序串行，乱序响应不回滚；并发图标替换保持单个完整文件且不改 metadata 版本 |
| 鼠标与键盘 | 真实 Chromium/Firefox 验证阈值前不 capture，普通 click/dblclick 仍命中子按钮；6 px 后才 capture 并拖拽，放下不误打开；同页/跨页交换、边缘自动滚动、边界夹取、自动排列、Ctrl/Command+方向键、Enter/Space、菜单键和焦点/播报正确 |
| 触摸与手写笔 | 单击打开、长按菜单、宽屏直接拖动、12 px 阈值、多指、`pointercancel`、失焦与滚动边界不误开或误保存；窄屏不允许重排 |
| 响应式与容量 | 至少覆盖 1440、1280、761、760、390、320 px 和 80%、125%、200% 缩放；单页容量、单页容量+1、64 shortcuts+固定/动态入口均逐页唯一且可滚动访问；第 513 个入口进入独立临时区且可打开，不能重排，自动整理不 PUT、不报成功；`<=760` 只派生紧凑布局且不发位置 PUT，`1440 → 390 → 1440` 恢复原宽屏位置；图标不侵入任务栏或安全区 |
| 失败与损坏 | 磁盘满、只读目录、原子替换失败、请求取消和进程重启不报告假成功；metadata 已提交而图标失败时可辨识和重试；损坏/未知 schema 保留现场、返回 unavailable 并只读降级 |
| 存储与回滚 | 空目录初始化、权限、工作区/图标配额、孤儿回收、成对备份恢复；新版本写入后回滚旧二进制再升级，workspace 和图标仍可恢复 |
| 性能与体验 | 最大数据集、连续拖拽、图标懒加载/缓存、深浅色、中英文、减少动画、加载/空/失败/冲突状态和零新增控制台错误 |
| 工程门禁 | 相关 Go 单测与 race、Web store/API/布局/组件测试及构建通过；`make verify-change`、`make verify-l2`、Linux amd64/arm64 构建和真实浏览器验收形成证据 |

本地 mock 预览只证明布局和交互观感，不能替代 Session/CSRF、真实落盘、并发冲突、Linux 重启、
失败注入和回滚兼容证据。上线前必须另行冻结候选，并按项目 L3 发布流程重新验收。
