# KPanel v0.70.2 上线验收记录

日期：2026-08-14（Asia/Shanghai）

## 发布结论

KPanel v0.70.2 已按 `release-kpanel` v2.1 完成候选冻结、L3、真实 Chrome、候选 CI、主线 CI、Tag、GitHub Release、双架构公开镜像、应用市场契约、公开镜像 E2E、停写一致性备份以及 154/108 正式升级。

108 未参与候选、功能、浏览器、灰度或持续观察测试；只在不可变产物与 154 门禁全部通过后执行标准升级及最小部署安全核对。

## 上线内容与排除项

- 文件管理多选右键菜单补齐批量下载、复制、剪切、压缩、修改权限和移入回收站；批量权限修改携带资源版本，回收站支持选择当前列表。
- 主机终端与应用交互终端在会话激活和标签切换后优先恢复 Shell 输入焦点。
- Docker 输入框、下拉框、复选框和部署编辑器统一控件尺寸、圆角与明暗主题表现。
- 桌面外部打开确认窗改为紧凑、扁平的信息结构，继续保留取消不外跳、确认后精确打开系统浏览器与 `noopener` 隔离。
- 将间接依赖 `nanoid` 从 `3.3.17` 更新到 `3.3.18`，修复发布期间新增的 High 级漏洞公告。
- 未纳入旧内置浏览器、历史开发工作树、未批准分支、`kejilion.sh` 变化或上述安全补丁之外的依赖升级。

## 源码、CI 与版本

- 发布基线：`6e9b3b9dc487ded944e78a21f803eb4235c63bab`（v0.70.1 上线验收记录）
- v0.70.2 Release commit：`30c4c1153e320f28a31bb4bd89428e9da1f3f8e3`
- tree：`b8759b9c6b378da8605f1be4231b443d2690614c`
- Tag：`v0.70.2`，解析目标为上述 Release commit
- Candidate CI：`31722039088`，通过
- Candidate dependency freshness：`31722039094`，通过
- Main CI：`31722233732`，通过
- Main dependency freshness：`31722233796`，通过
- Tag Release workflow：`31722466596`，通过
- Tag dependency freshness：`31722466575`，通过
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.70.2>，非 draft、非 prerelease，共 8 个附件

初始候选 `81662ad4aba218864bba6d8388d199405026d758` 的候选 CI 曾通过；随后漏洞数据库新增 `nanoid <3.3.18` High 公告，导致 main CI 的 `npm audit` 两次稳定失败。发布在 Tag 前停止，使用单一锁文件提交 `30c4c1153e320f28a31bb4bd89428e9da1f3f8e3` 修复并从 L3、浏览器、候选 CI、main CI 全部重跑，没有绕过门禁或改写既有 Tag。

## 自动化与隔离验收

- 最终候选完整 L3：Go 全量、race、vet、Web 86 个测试文件/616 项测试、i18n 2144 条、TypeScript、生产构建、govulncheck、npm audit、Trivy 源码/镜像、双架构构建、受限容器、安装安全、治理和应用生命周期全部通过。
- L3 日志 SHA-256：`2e7d8c2dd3b9c908ff9fd1a13e5f26c28148efd803e22e5a25cc4ec1a49aa197`
- L3 证据：`/root/kpanel-release-v0702-30c4c11-r2`；本地副本：`C:\GitHub\_release-artifacts\v0.70.2`
- 完整 Git bundle：`kpanel-v0.70.2-30c4c11.bundle`，SHA-256 `753ae569cd7dc8c963091d3685066aa16bed674143ea086f2a274954dedb9489`
- Google Chrome `151.0.7922.138` 使用随机临时 Profile 在后台验收：文件批量操作、真实 Shell 焦点、Docker 深浅主题控件和桌面外部确认全部通过；结果 SHA-256 `a34bedd06d84b5b99f9df9a39fe82e6aebde9c21f4c28c6e8b599172e664f3ef`。
- Chrome 未读取或接管用户日常 Profile；临时 Profile、隧道、测试容器、网络和两个测试文件均已精确清理。

验证环境中的站点 `final-mowing` 是仅用于本轮确认窗的未解析测试域。验收记录了精确顶层导航请求和 `window.opener === null`；其 DNS 不可解析不属于产品失败。

## 不可变产物与应用市场

- OCI index（`0.70.2` 与 `latest`）：`sha256:dd735ce066930db6026a248f7fdce850ee14df9c4a5cfff80c6c8c4ff0bd57f6`
- Linux amd64 manifest：`sha256:adb32edd5336eca4bbd2db34e50e7d34bc28729d70c8109a525e2fbcaacc74d2`
- Linux arm64 manifest：`sha256:e01eeda1515d432dc1560f029e159a517a2304f1d417efc2162ac65abb4659d5`
- 两个架构的 OCI version 为 `0.70.2`，revision 为 `30c4c1153e320f28a31bb4bd89428e9da1f3f8e3`。
- 154 从 Docker Hub 按不可变摘要重新拉取，公开镜像 E2E 输出 `image_e2e=pass`；日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- `kejilion/apps` commit：`c5cc79ce4acd7f7b373573952616dbabd2060b7d`
- 本轮应用配置无源码差异；apps 与 KPanel 的 Git blob 均为 `34316059d4e42f527819bc7d56e0ff14ec434c96`，Linux 文件 SHA-256 `82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`，因此未产生空 apps 提交。
- 上线前发现两台主机 `/root/apps/kpanel.conf` 的本地缓存不是上述最新 blob；实时 Compose 均已没有 Browser Relay，业务运行未漂移。备份旧缓存后，两台主机均先同步精确官方配置，再执行标准 `docker_app_update`，当前应用配置已一致。

## 正式部署

### arena-154

- 端口：`8080`
- 上线后：版本 `0.70.2`，revision 与 OCI 摘要精确匹配；Panel healthy、restart 0、OOM false；Agent active
- SQLite `quick_check`、JSON 解析、Compose 解析、错误日志和 3 次健康采样通过；旧 v0.70.1 运行镜像已清理
- 证据：`/root/kpanel-release-evidence/v0.70.2/arena154-20260813T165722Z`
- 备份：`/root/kpanel-backups/v0.70.2-preupgrade-arena154-20260813T165722Z.tar.gz`
- 备份 SHA-256：`7b12026185e59f0c79044acd82c5bf781eadff99b669265678b7355512dc8620`
- v0.70.1 镜像归档：`/root/kpanel-backups/v0.70.2-preupgrade-arena154-20260813T165722Z.image.tar`
- 镜像归档 SHA-256：`9637b138b68418b080e257862351c68cebd4bedbe396906742430a2031dc7d84`

### production-108

- 端口：`5566`
- 上线后：版本 `0.70.2`，revision 与 OCI 摘要精确匹配；Panel healthy、restart 0、OOM false；Agent active
- 仅执行停写备份、标准应用市场更新、版本/健康/Agent/数据/Compose/回滚核对和 3 次最小健康采样；未执行功能、浏览器、灰度、故障注入或长时间观察
- 旧 v0.70.1 运行镜像已清理
- 证据：`/root/kpanel-release-evidence/v0.70.2/prod108-20260813T170034Z`
- 备份：`/root/kpanel-backups/v0.70.2-preupgrade-prod108-20260813T170034Z.tar.gz`
- 备份 SHA-256：`cf5f3fbcf51ccbec51e67e1e1985b8a801e8aa5a1d36d6322f94ee916b3b2c45`
- v0.70.1 镜像归档：`/root/kpanel-backups/v0.70.2-preupgrade-prod108-20260813T170034Z.image.tar`
- 镜像归档 SHA-256：`e2e9446bdd7aff67d6c8fbe6ddddd1c87017490f3bbaf8fc0fa05eb34b53fc80`

两台主机的备份均在 Panel 与 Agent 停写状态生成；归档文件摘要、完整文件清单、权限/属主/符号链接清单、SQLite/JSON 数据、镜像归档可读性及独立恢复目录校验全部通过。恢复校验后旧版本先恢复健康，再进入正式升级。

## 回滚方案

回滚点：`v0.70.1` / `9f124c42c175a8f1c2b2a2194c6a31e4bb89c8f5` / OCI `sha256:ceae5b2ec62f7d93b96fdc0a0caff8817ced20e30887a4aef74a98da04f99813`。

每台主机回滚必须成套执行：

1. 停止 Panel 与 `kejilion-agent.service`，确认写入停止。
2. 校验该主机 `.tar.gz` 与 `.image.tar` 的 SHA-256。
3. 从归档恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 和 `kejilion-agent.service`。
4. `docker load` 对应 v0.70.1 镜像归档，并按证据中的 `old-image.id` 恢复 `docker.io/kjlion/kejilion-panel:latest`。
5. `systemctl daemon-reload`，启动 Agent，再使用已恢复 `.env` 与 Compose 重建 Panel。
6. 验证 v0.70.1、原端口、Panel healthy、Agent active、数据完整性和 revision。

本次没有触发生产回滚。

## 遗留风险与工作流沉淀

- 本轮没有数据库、API、端口、Compose、Agent 权限或 `kejilion.sh` 协议变化；主要风险集中在前端交互，已由自动化与真实 Chrome 覆盖。
- `nanoid` 公告在发布过程中进入审计数据库，证明外部漏洞源会动态变化；本轮按既有 fail-closed 门禁修复并全量重验，没有放宽规则。
- 未新增重复规范；继续复用 `release-kpanel` v2.1、项目版本控制、环境策略、应用生命周期和验收模板。
