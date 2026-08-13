# KPanel v0.68.3 发布验收记录

日期：2026-08-13

发布级别：L3 / 内置浏览器样式补丁

正式提交 / 标签：`5f867855503b9e4e38376dc3625bc055e04c54f7` / `v0.68.3`

功能提交：`6ba411d5334dfa2ff14a48b21f88f6c2be489087`（`fix(browser): preserve safe page styling`）

上一版本 / 回滚点：`v0.68.2` / `22a3c14d0bd2888aaec9c62ba99f9da486ff0394`

## v0.68.2 失败结论与修复范围

- v0.68.2 已发布并部署，但生产实测发现安全阅读链路只显示 HTML，页面 CSS 丢失、排版混乱，因此停止成功验收并按变更失败记录；当时线上按用户决定维持 v0.68.2，等待本补丁，没有改写历史 Tag/Release/OCI。
- v0.68.3 保留经 allowlist 过滤的安全 `id`、`class` 和行内样式，识别 inline `<style>` 与外部 `rel=stylesheet`；外部 CSS 继续只经 Panel → Relay 获取，frame CSP 继续禁止目标页面自行联网。
- CSS 资源上限为 16 个、单个 512 KiB、总计 2 MiB，前后端使用相同边界；没有放宽脚本执行、Cookie、表单、下载、SSRF、DNS 重绑定、TLS、Origin、Token、CSP、并发或资源限制。
- 更新交互没有变化，不增加 y/n，不打印浏览器模式或实现说明。

## 环境治理调整

- `prod-108` 从本版本起固定不再作为线上测试、候选验收、灰度、canary、兼容性矩阵、失败注入、压力测试或持续观察目标，只承载正常生产运行。
- 108 本次只在稳定正式产物完成后执行停写备份、标准更新和最小非侵入式版本、健康、数据完整性与回滚就绪核对；这些结果不作为候选测试证据。
- 内置浏览器专项连续测试窗口规范调整为 10 分钟；本版本发布前用户进一步明确要求不再执行额外 Chrome 与长观察，因此按授权取消该项，只保留全量 L3、定向回归、真实页面结构和隔离健康快照。未把未执行的真实视觉复核写成已通过。
- 上述治理已纳入 `release-kpanel v1.9`、`PROJECT_RULES.md`、项目管理规范和发布验收模板。

## 候选冻结与验证

- 冻结源提交 parent 为 v0.68.2 正式主线 `22a3c14d...`；原始补丁 bundle 为 `C:\GitHub\_release-artifacts\browser-style-fix-6ba411d\kpanel-browser-style-fix-6ba411d.bundle`，SHA-256 `0a7590aa68b41958d94af8d7f55068a69a67b6e1025195da6c21d9d8163ce6a8`。
- 最终自包含候选 bundle 为 `C:\GitHub\_release-artifacts\v0.68.3-browser-style\kpanel-v0.68.3-5f86785-self-contained.bundle`，SHA-256 `83ae9cf61fd01607101c44f0cc20b25615faac73fe4fb534571963eb77e028fd`。
- 154 全量 L3 于 `2026-08-13T02:57:56Z` 完成：Go 全量测试、`go vet`、核心包 race、Web 全量测试、i18n 2144 条、typecheck、生产构建、Linux amd64/arm64、正式 Dockerfile、受限容器、安全扫描、安装安全和 apps 生命周期全部通过；末尾标记 `L3 release verification completed` 与 `app_conf_lifecycle=pass`。
- L3 日志 SHA-256 为 `e8618b9acf18289193f6c006cab5a40b13b7acfd488770a51be04a7e7590faf2`；候选镜像 inspect SHA-256 为 `0a2785eba891c9fe99bf5ad1bb45849305fc9b9ae845ee10eafe6e8388b508d7`。
- 样式专项的 `browserReaderRuntime` 与 `WebBrowserView` 共 24 项回归、`vue-tsc`、i18n、生产构建、`node --check` 和 diff-check 已通过。
- `https://dh.kejilion.pro/` 实际响应为 HTTP 200、159645 字节，含 1 个 12928 字符 inline style 和 751 个 class 属性，属于本补丁明确覆盖的结构。
- 154 隔离候选健康快照：Panel/Relay 均 healthy、restart 0、OOM false；Panel 约 13.34 MiB / 256 MiB，Relay 约 4.88 MiB / 128 MiB；随后已完整清理隔离容器、网络和临时数据。
- 候选 CI `31663210678`、候选依赖新鲜度 `31663210709`、主线 CI `31663425862`、主线依赖新鲜度 `31663425838`、Release workflow `31663731860` 和 Tag 依赖新鲜度 `31663731853` 均成功。

## Release、OCI 与应用市场

- [GitHub Release v0.68.3](https://github.com/kejilion/KPanel/releases/tag/v0.68.3) 已公开，非 draft、非 prerelease；annotated Tag 对象 `d5d43cc55a92a42e4fb502553ac7ff691ef34d95` 精确剥离到 `5f867855...`。
- `docker.io/kjlion/kejilion-panel:0.68.3` 与 `latest` 指向 OCI index `sha256:09dd3a78750db8cdc441e57485012a50a449ac805d6d011e3bcb7169ec62f52f`。
- `linux/amd64` manifest 为 `sha256:8f8b6a2bf6debaeefd4fd54e40a22aabc4e8d5c85ef8e4dda34e2a77ad474281`；`linux/arm64` manifest 为 `sha256:f1e99c2592a2cb986e6187710eab8153f811a560f744c69c030572027ed52db9`。version/revision 为 `0.68.3` / `5f867855...`，公开拉取 E2E 输出 `image_e2e=pass`。
- `packaging/kejilion-app/kpanel.conf` 相对 v0.68.2 blob 未变化，仍为 `88242c156bb89137865b40ac5feb43cda410af6f`；apps `main` 的 `0a288d652eee4cbcf5f839b6cb650a9a81865609` 已包含该契约，因此没有制造空提交。

## 生产备份与升级

### arena-154

- 停写一致性备份：`/root/kpanel-backups/v0.68.3-preupgrade-arena154-20260813T033056Z`；归档 SHA-256 `9ca1aee982ea368acaf813b74658a83a435c66d1ffa7ff56d31bcd074afc4133`；v0.68.2 镜像归档 SHA-256 `6bafb3ad8e76dbe22609e7fae46ab741a7a67035eeef5399d673f6d3f4b2f910`。
- 备份完成独立解包、文件摘要、SQLite/JSON/JSONL、权限和数据完整性恢复核验。
- 使用标准应用市场更新入口升级；更新输出未出现“是否”、`[y/N]`、“内置浏览器”或 `KPANEL_BROWSER_MODE`。升级后 Panel health 为 `0.68.3`，Relay mode 保持 `beta`；两容器 healthy、restart 0、OOM false，Agent active、restart 0，数据完整性通过。

### prod-108

- 停写一致性备份：`/root/kpanel-backups/v0.68.3-preupgrade-prod108-20260813T033056Z`；归档 SHA-256 `60751cde6a76ae5b01f45016aac4b7cae402d981a9a1a9fa10721c4ee70fba5d`；v0.68.2 镜像归档 SHA-256 `82c444a727d08c5cbd78fb7f84d4fc09e1d39af621a518ba3d8fce27fef8b0d1`。
- 备份完成独立恢复核验。既有 `/root/apps` 保持原提交 `5f0c9ec869715a89238fb372c867b6930f110fb8`，没有被发布流程覆盖；标准更新使用隔离的 apps 正式提交 `0a288d6`。
- 按固定环境策略，108 未执行内置浏览器测试、灰度或持续观察；只完成稳定版本标准升级和最小安全核对。Panel health 为 `0.68.3`，Relay mode 为 `reader`；两容器 healthy、restart 0、OOM false，Agent active、restart 0，数据完整性通过。

两台正式镜像标签 revision/version 均精确为 `5f867855...` / `0.68.3`，RepoDigest 均为 `sha256:09dd3a...`。两台 Agent SHA-256 均为 `ab93d78c6a27e87b7567e1998992cbc2327d899c605b76dfa9e065237cc9309d`；托管 `kejilion.sh` SHA-256 均为 `d73231f146f7398d7b50133695faf2116134fbfe33a7b94068e277cc7b82df55`。

## 回滚与遗留边界

- v0.68.2 的源码、Tag、Release、OCI 和本次两台停写备份均保留为精确回滚点。回滚必须成套恢复匹配版本的镜像、Compose、`.env`、数据和密钥，不得只替换镜像或只修改浏览器模式。
- 本次按用户明确决定没有执行补丁后的真实 Chrome 最终视觉复核和 10/30 分钟持续观察，因此不能把 `dh.kejilion.pro` 的最终渲染效果写成已实测；覆盖证据为真实页面结构、定向回归、全量 L3、公开产物 E2E 和隔离/生产健康快照。
- 如仍发现样式兼容问题，应优先禁用内置浏览器执行 kill switch，或成套回滚至 v0.68.2，并在新的补丁候选中增加对应真实结构回归；不得在 108 上开展测试。
- 原始 Cookie、Token、密码、密钥和未脱敏 trace 未进入仓库、Release 或公开材料。
