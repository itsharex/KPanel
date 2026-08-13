# KPanel v0.69.0 上线验收记录

日期：2026-08-13（Asia/Shanghai）

## 发布结论

KPanel v0.69.0 已按 `release-kpanel` v1.9 完成候选、主线、Tag、公开产物、应用市场、154 上线门禁及 108 标准升级。内置浏览器及 Relay 已完整移除；桌面 URL 类应用和已部署网站的四个外跳入口统一先确认，再由系统浏览器打开。脚本型应用仍进入脚本管理。

154 用于隔离验收和正式上线门禁。108 未参与功能测试、灰度或持续观察，仅在不可变产物和 154 门禁通过后执行停写备份、标准升级和最小生产健康核对。

## 源码与版本

- Release commit：`9b15314d0b800d12d00c4b6baad65ae01a245e89`
- Release tree：`77df4b730138e5229b31664c01c504ee7bc2fb3f`
- Tag：`v0.69.0`，解析目标为上述 Release commit
- 版本：`0.69.0`
- 回滚版本：`v0.68.3`
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.69.0>
- Candidate CI：`31677078015`，通过
- Candidate dependency freshness：`31677078026`，通过
- Main CI：`31677174805`，通过
- Main dependency freshness：`31677174774`，通过
- Tag Release workflow：`31677498852`，通过
- Tag dependency freshness：`31677498853`，通过

候选分支 `release/v0.69.0-candidate` 已由 Release workflow 清理；主线通过普通快进更新，没有强推或改写历史。

## 自动化与隔离验收

- 完整 L3：通过；Go/Web、race、vet、生产构建、安装安全、应用生命周期、依赖治理、双架构构建、Trivy 和 govulncheck 均通过。
- 前端：86 个测试文件、607 项测试通过；i18n 2144 条通过。
- L3 日志 SHA-256：`1138a3997b44a7f7d9e3f42180bbe839611e4d6c37abb790780aa2970956ca8d`
- 154 公开镜像 E2E：从 Docker Hub 精确摘要拉取并启动，health 返回 `0.69.0`，revision 精确匹配，0 restart、无 OOM，根页面可读取。
- 应用市场 `kpanel.conf` 与冻结源码逐字节一致，SHA-256 为 `0cf33f7b0cb617a6948d447f797ed7b7e36fdb026b2595757868a969d8b73421`；一次性容器中的 Bash 语法和完整 install/update/uninstall/negative-path lifecycle 通过。

## 独立正式 Chrome 门禁

使用本机正式 Google Chrome `151.0.7922.138` 的独立 headless `launchPersistentContext`，配合随机临时 `user-data-dir` 完成。未读取、枚举、切换或接管用户 Chrome Profile/标签，未使用 Chrome 扩展；进程和临时 Profile 已清理。

四条真实候选路径均通过：

1. URL 类已安装应用桌面图标：确认前不外跳；取消不外跳；确认后请求精确 URL；`window.opener === null`。
2. 已部署网站桌面图标：同一确认窗和精确 URL。
3. 已部署网站右键菜单：同一确认窗和精确 URL。
4. 已部署网站详情页入口：同一确认窗和精确 URL。

候选真实数据中没有脚本型已安装应用，因此另用仅浏览器响应边界的 fixture 验证：脚本型应用进入脚本管理，不出现外跳确认，也不创建外部页面。fixture 未修改源码、数据库、Agent、宿主机或生产数据。

关键证据：

- `headless-chrome-gate.json`：`CD2E5F705B584417C5DC2ADC84B617A51BE5F2B7480F25994F80EE25FBF81BF7`
- `headless-chrome-script-fixture.json`：`EC126F681D95DD3458A8A8DEA9FB39DC41DA35E0A3F96DF55D72C89947FA0035`
- 页面错误：0

Chrome 扩展控制通道此前的连接失败被归类为证据工具故障；项目规范没有要求必须使用该扩展。最终门禁未跳过用户旅程，而是改用可复核的独立正式 Chrome 通道。

## 不可变发布产物

- OCI index / `0.69.0` / `latest`：`sha256:3e16ac037267c3d1b7e3b4db033f8011de281dc2826c9481fbed178d7edc2ccb`
- Linux amd64 manifest：`sha256:61fd51b097d36fc48cc9e4f394a078a5871f67902162124e96b799f63e430a27`
- Linux amd64 config：`sha256:2f5514f7371b4380cf1dee3f7c3046dbe76ce3c6a6d90f2f3405496f4fd42e70`
- Linux arm64 manifest：`sha256:e7955fac721f9f73602d657be2f73091bb460ae085e91b2a512e4f352fb63bf1`
- Linux arm64 config：`sha256:d15748302ecbc7cc496357055136be479798699180b83d3fcc5aa11f7b76c9ab`
- 两个架构的 OCI label 均为 revision `9b15314d0b800d12d00c4b6baad65ae01a245e89`、version `0.69.0`。
- GitHub Release 已公开，非 draft、非 prerelease，包含双架构 Agent/Node、部署包、许可证、第三方声明和 `SHA256SUMS`。

## 应用市场

- 仓库：`kejilion/apps`
- Commit：`b7d797f75c80d666aec88f51daa6fe43b61b5e72`
- 变更：仅 `kpanel.conf`
- 结果：移除 Relay 服务、浏览器运行模式和相关 secret/Compose 绑定；更新使用 `--remove-orphans` 清理旧 Relay。
- Lifecycle：通过。

## 生产上线

### arena-154

- 端口：`8080`
- 上线后版本：`0.69.0`
- Panel：healthy，restart 0，OOM false
- Agent：active
- 旧 `kejilion-browser-relay`：不存在
- Compose 服务：仅 `panel`
- OCI 身份：index digest `sha256:3e16ac037267c3d1b7e3b4db033f8011de281dc2826c9481fbed178d7edc2ccb`，revision/version 正确
- 数据：SQLite `quick_check`、JSON/JSONL 解析通过
- 证据：`/root/kpanel-release-evidence/v0.69.0/arena154-20260813T073544Z`
- 备份：`/root/kpanel-backups/v0.69.0-preupgrade-arena154-20260813T073544Z.tar.gz`
- 备份 SHA-256：`bea835e2fd86a13af71574e02d42a143c05f0a15be211fc5f926c04329accdbe`
- 旧镜像归档：`/root/kpanel-backups/v0.69.0-preupgrade-arena154-20260813T073544Z.image.tar`
- 镜像归档 SHA-256：`2877b6ea3c2b77e820e185beab7b35532127f6c25797aebaf49dce264d674b7d`
- 旧镜像 ID：`sha256:09dd3a78750db8cdc441e57485012a50a449ac805d6d011e3bcb7169ec62f52f`

### production-108

- 端口：`5566`
- 上线后版本：`0.69.0`
- Panel：healthy，restart 0，OOM false
- Agent：active
- 旧 `kejilion-browser-relay`：不存在
- Compose 服务：仅 `panel`
- OCI 身份：amd64 config digest `sha256:2f5514f7371b4380cf1dee3f7c3046dbe76ce3c6a6d90f2f3405496f4fd42e70`，revision/version 正确
- 数据：SQLite `quick_check`、JSON/JSONL 解析通过
- 证据：`/root/kpanel-release-evidence/v0.69.0/prod108-20260813T073841Z`
- 备份：`/root/kpanel-backups/v0.69.0-preupgrade-prod108-20260813T073841Z.tar.gz`
- 备份 SHA-256：`6d7b761e0d9ecd3eb01130560ecd0048b451e1367bae2abcaadda1909b1084e5`
- 旧镜像归档：`/root/kpanel-backups/v0.69.0-preupgrade-prod108-20260813T073841Z.image.tar`
- 镜像归档 SHA-256：`dda725d06666e4763bd586cac31fd491322a9926a4b2a28568fb50a4078106af`
- 旧镜像 ID：`sha256:ddbf5612b9f751dc62870cf217e6ae99c5ff530e3431283bba5c9122118bceea`
- 额外保存实际 v0.68.3 apps 配置：`kpanel.conf.active-v0683`，SHA-256 `f084c8cd57ddfaa85f4d333e8c989648da7ef36ef938d1abf17b5e454b2e3bfe`

两台服务器的备份均在停写状态生成；源数据完整性、归档可读性、独立恢复目录的 manifest/文件摘要和恢复后数据完整性均已验证。测试候选容器、隔离网络、临时远端脚本和临时 Chrome Profile 已清理；生产备份及证据保留。

## 回滚方案

任一主机需要回滚时，必须成套恢复，不允许只切镜像：

1. 停止 `kejilion-agent.service`、Panel，并确保当前写入停止。
2. 校验对应 `.tar.gz` 和 `.image.tar` 的 SHA-256。
3. 从备份归档恢复 `/home/docker/kpanel`、匹配的 apps 配置和 `kejilion-agent.service`。
4. `docker load` 对应旧镜像归档，并将记录的旧镜像 ID 重新标记为 `docker.io/kjlion/kejilion-panel:latest`。
5. 执行 `systemctl daemon-reload`，启动 Agent；使用恢复的 `.env` 和旧 Compose 执行 `docker compose up -d --force-recreate`，恢复 v0.68.3 的 Panel 与 Relay 组合。
6. 验证版本 `0.68.3`、Panel/Relay health、Agent active、数据完整性及旧端口。

本次没有触发生产回滚。备份中的独立恢复校验已经通过，回滚入口保持可执行。
