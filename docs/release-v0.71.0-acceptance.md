# KPanel v0.71.0 上线验收记录

日期：2026-08-14（Asia/Shanghai）

## 发布结论

KPanel v0.71.0 已按 `release-kpanel` v2.1 完成候选冻结、完整 L3、154 隔离升级与真实 Chrome、候选 CI、主线 CI、Tag、GitHub Release、双架构公开镜像、应用市场契约、公开镜像 E2E、停写一致性备份以及 154/108 正式升级。

108 未参与候选、功能、浏览器、灰度或持续观察测试；只在不可变产物与 154 门禁全部通过后执行标准升级及最小部署安全核对。

## 上线内容与排除项

- 新增桌面工作区 v1：支持桌面图标拖拽定位、键盘微调、自动排列、位置持久化和紧凑视口保护。
- 新增自定义桌面快捷方式：名称、地址、描述和可选 PNG/JPEG/WebP 图标；支持创建、编辑、删除和刷新后持久化。
- 已安装应用与已部署网站可仅从桌面隐藏，并可从桌面图标管理器恢复；不卸载应用、不删除网站或目标数据。
- 图标工作区独立保存于 `${DataDir}/desktop-workspace/`，`workspace.json` 与 `icons/` 必须成对备份和恢复；旧版本忽略并保留该目录。
- 更新桌面图标视觉、青龙与 Nginx Stream 图标，增加项目级应用图标规范化工作流；本地预览兼容 WSL API 地址。
- Go 工具链统一升级到 1.26.6，并固定 Alpine 构建镜像摘要。
- README 更新为当前产品概览、能力边界、快速开始、文档与截图入口。
- 未纳入旧内置浏览器、旧草稿/重复工作树、已在 v0.70.2 上线的 Docker 下拉框优化、无关依赖升级或 `kejilion.sh` 变化。

## 源码、CI 与版本

- 发布基线：`606268a544bb3829bafea85bc428d7b4cc35236b`（v0.70.2 上线验收记录）
- v0.71.0 Release commit：`851d3a83940748671bf182c475f80778c9513ffe`
- tree：`de6ec112b85495a009724948f4b061745ebed9de`
- Tag：`v0.71.0`，解析目标为上述 Release commit
- Candidate CI：`31784487996`，通过
- Candidate dependency freshness：`31784488020`，通过
- Main CI：`31784863354`，通过
- Main dependency freshness：`31784863385`，通过
- Tag Release workflow：`31785221749`，通过
- Tag dependency freshness：`31785221735`，通过
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.71.0>，非 draft、非 prerelease，共 8 个附件

主线使用普通快进更新，没有强推或改写既有版本；候选分支由 Release workflow 在发布完成后自动清理。

## 自动化与隔离验收

- 完整 L3：Go 全量、核心包 race、vet、Web 91 个测试文件/657 项测试、i18n 2144 条、TypeScript、生产构建、govulncheck、npm audit、Trivy 源码/镜像、Linux amd64/arm64、受限容器、安装安全、治理、固定脚本契约和应用生命周期全部通过。
- Go 固定版本：1.26.6；构建镜像摘要：`sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df`。
- L3 日志 SHA-256：`f95566c6adb9708e3c152f0d203d6df60e7ccfb0cf86d6375fdc620bd15884b1`
- L3 证据：`/root/kpanel-release-v0710-851d3a8`；本地副本：`C:\GitHub\_release-artifacts\v0.71.0`
- 完整 Git bundle：`kpanel-v0.71.0-851d3a8.bundle`，SHA-256 `a4115a2ed8419699e63311e74fc825499baf1fabd8ee8ad252ce9d687ac25265`
- Google Chrome `151.0.7922.138` 使用随机临时 Profile 在后台验收：v0.70.2 隔离数据升级生成 workspace v1、创建带图标快捷方式、拖拽和编辑刷新持久化、隐藏应用但不卸载、管理器恢复、删除快捷方式以及 390px 无横向溢出全部通过；页面脚本错误 0。
- 浏览器结果 SHA-256：`8b963ab2b1fc67c537bba5319a44e75d92478d271c83f79e64169978a65800a7`。
- Chrome 未读取或接管用户日常 Profile；临时 Profile、SSH 隧道、候选容器、网络和隔离数据均已精确清理。

浏览器最终通过前的若干运行只暴露了验收脚本的 Node 路径、中文按钮名称、动态 locator 和桌面过渡时序问题；候选源码没有因此修改。涉及状态写入的重试前均从同一 v0.70.2 隔离数据重新复制，最终通过结果来自精确 Release commit。

## 不可变产物与应用市场

- OCI index（`0.71.0` 与 `latest`）：`sha256:4857c207d127facc2228aa205862e51d52e6a8428f9750a1516c8222e085d5b5`
- Linux amd64 manifest：`sha256:68a8f8816ac8b9f503c2e4ab08c25295346bc82f337900f07bed89b2fba5497b`
- Linux arm64 manifest：`sha256:a69b7b1cc5c665572fbdfbf1aac76c51ee0240e55e0b1834b9372ad739a9c703`
- 两个架构的 OCI version 为 `0.71.0`，revision 为 `851d3a83940748671bf182c475f80778c9513ffe`。
- 154 从 Docker Hub 按不可变摘要重新拉取并执行冷启动 E2E，输出 `image_e2e=pass`；日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- `kejilion/apps` commit：`c5cc79ce4acd7f7b373573952616dbabd2060b7d`
- 本轮应用配置无源码差异；apps 与 KPanel 的 Git blob 均为 `34316059d4e42f527819bc7d56e0ff14ec434c96`，文件 SHA-256 为 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，因此未产生空 apps 提交。
- 应用配置生命周期由候选 L3 与 Tag Release workflow 对同一文件再次验证通过。

## 正式部署

### arena-154

- 端口：`8080`
- 上线后：版本 `0.71.0`，revision 与 OCI 摘要精确匹配；Panel healthy、restart 0、OOM false；Agent active
- SQLite `quick_check`、JSON 解析、Compose 解析、错误日志和 3 次健康采样通过；v0.70.2 升级前运行镜像已清理
- 最终快照：Panel 约 74.94 MiB / 256 MiB、8 PIDs
- 证据：`/root/kpanel-release-evidence/v0.71.0/arena154-20260814T085451Z`
- 备份：`/root/kpanel-backups/v0.71.0-preupgrade-arena154-20260814T085451Z.tar.gz`
- 备份 SHA-256：`63f7d0918f879b434d6383d2345d8da6d3e9fe613a1ac43a2bfd79a339f0a1b4`
- v0.70.2 镜像归档：`/root/kpanel-backups/v0.71.0-preupgrade-arena154-20260814T085451Z.image.tar`
- 镜像归档 SHA-256：`549bb4573ccd6e0e3d3b77dd7250fb15b013e272ab799c8a44e2173390660d7e`

### production-108

- 端口：`5566`
- 上线后：版本 `0.71.0`，revision 与 OCI 摘要精确匹配；Panel healthy、restart 0、OOM false；Agent active
- 只执行停写备份、标准应用市场更新、版本/健康/Agent/数据/Compose/回滚核对和 3 次最小健康采样；未执行功能、浏览器、灰度、故障注入或长时间观察
- 最终快照：Panel 约 75.82 MiB / 256 MiB、8 PIDs
- v0.70.2 升级前运行镜像已清理
- 证据：`/root/kpanel-release-evidence/v0.71.0/prod108-20260814T085743Z`
- 备份：`/root/kpanel-backups/v0.71.0-preupgrade-prod108-20260814T085743Z.tar.gz`
- 备份 SHA-256：`222f310206577652dbb3db4ed76c9e7a7c1e1d34e9097742decce4bee6c03c96`
- v0.70.2 镜像归档：`/root/kpanel-backups/v0.71.0-preupgrade-prod108-20260814T085743Z.image.tar`
- 镜像归档 SHA-256：`480d31500350df477c5b02f6f540fec0c5aaef67bdc757ef136fcbab363d1cdb`

两台主机的备份均在 Panel 与 Agent 停写状态生成；归档摘要、完整文件清单、权限/属主/符号链接清单、SQLite/JSON 数据、镜像归档可读性及独立恢复目录校验全部通过。恢复校验后旧版本先恢复健康，再进入正式升级。

## 回滚方案

回滚点：`v0.70.2` / `30c4c1153e320f28a31bb4bd89428e9da1f3f8e3` / OCI `sha256:dd735ce066930db6026a248f7fdce850ee14df9c4a5cfff80c6c8c4ff0bd57f6`。

每台主机回滚必须成套执行：

1. 停止 Panel 与 `kejilion-agent.service`，确认写入停止。
2. 校验该主机 `.tar.gz` 与 `.image.tar` 的 SHA-256。
3. 从归档恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 和 `kejilion-agent.service`。
4. `docker load` 对应 v0.70.2 镜像归档，并按证据中的 `old-image.id` 恢复 `docker.io/kjlion/kejilion-panel:latest`。
5. `systemctl daemon-reload`，启动 Agent，再使用已恢复 `.env` 与 Compose 重建 Panel。
6. 验证 v0.70.2、原端口、Panel healthy、Agent active、数据完整性和 revision。

v0.70.2 会忽略并保留 `${DataDir}/desktop-workspace/`；如果需要连同桌面工作区数据一起回退，必须使用本轮完整停写归档，不得只切换镜像。本次没有触发生产回滚。

## 遗留风险与工作流沉淀

- 本轮没有数据库、端口、Compose、Agent 权限或 `kejilion.sh` 协议变化；主要新增风险是桌面布局和自定义图标数据，已由自动化、升级隔离数据和真实 Chrome 覆盖。
- `workspace.json` 与 `icons/` 必须成对备份/恢复；该要求已经写入 CHANGELOG Upgrade Notes 和桌面工作区设计文档。
- README、图标规范化工作流和 Go 工具链固定均随 Release commit 发布，没有另建重复发布规范。
- 未新增重复流程；继续复用 `release-kpanel` v2.1、项目版本控制、环境策略、后台浏览器验收和应用生命周期门禁。
