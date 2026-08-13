# KPanel v0.70.1 上线验收记录

日期：2026-08-13（Asia/Shanghai）

## 发布结论

KPanel v0.70.1 已按 `release-kpanel` v2.1 完成候选、主线、Tag、GitHub Release、双架构公开镜像、应用市场同步、154 隔离验收以及 154/108 正式升级。

本轮先发布 v0.70.0，包含桌面外部打开确认窗显示应用/网站图标、KPanel 更新后清理旧官方镜像，以及发布强制校验修复。生产升级后发现经典 Docker `overlay2` 中无 tag 的旧镜像不会被 `docker image ls` 枚举，因此 v0.70.0 服务虽健康，但镜像清理目标未完全实现。v0.70.1 将枚举改为 `docker image ls --all`，补齐防止测试替身漏报的回归，并作为不可变补丁版本重新走完整发布门禁。

108 未参与候选、浏览器、灰度或持续观察测试；只在不可变产物、154 门禁和可恢复备份通过后执行标准升级及最小健康核对。

## 上线内容与排除项

- 桌面 URL 类安装应用图标、已部署网站桌面图标、右键菜单和详情页入口继续统一先显示确认窗；确认窗显示对应应用/网站图标，图标不可用时使用既有回退图标。
- 脚本型应用仍进入脚本管理，不进入外部打开确认流程。
- KPanel 应用市场更新成功后，仅清理具有 KPanel 官方 OCI 标签、不是当前镜像且未被任何容器引用的旧镜像。
- v0.70.1 补齐经典 Docker 中 dangling 镜像枚举，不扩大删除条件，不删除当前镜像、被容器引用镜像、无关产品镜像或显式回滚资料。
- 强制 Release 校验在 clean checkout 中也完整执行，不再提前退出。
- 未纳入其他脏工作树、历史开发分支、临时浏览器产物或无关依赖升级。

## 源码、CI 与版本

- v0.70.0 Release commit：`3fe0fb15f5792ee6fb8d9893540dd996204a9462`
- v0.70.1 Release commit：`9f124c42c175a8f1c2b2a2194c6a31e4bb89c8f5`
- v0.70.1 tree：`0d4245527cad2550262f859d9eb039af1648329e`
- Tag：`v0.70.1`，解析目标为上述 v0.70.1 Release commit
- 回滚源码版本：`v0.70.0` / `3fe0fb15f5792ee6fb8d9893540dd996204a9462`
- Candidate CI：`31688646020`，通过
- Candidate dependency freshness：`31688646189`，通过
- Main CI：`31688909570`，通过
- Main dependency freshness：`31688909569`，通过
- Tag Release workflow：`31689159300`，通过
- Tag dependency freshness：`31689159432`，通过
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.70.1>，非 draft、非 prerelease，共 8 个附件

候选分支已由 Release workflow 自动清理；主线使用普通快进更新，没有强推或改写 v0.70.0/v0.70.1 历史。

## 自动化与隔离验收

- v0.70.1 完整 L3：Go 全量、race、vet、Web 86 个测试文件/608 项测试、i18n、TypeScript、生产构建、govulncheck、npm audit、Trivy 源码/镜像、双架构构建、受限容器、安装安全、应用生命周期和治理门禁全部通过。
- L3 日志 SHA-256：`c810ce27318c820452fb5f19c9c366fdc713ac8670f77168e5f4017045966af4`
- L3 证据：`/root/kpanel-release-v0701-9f124c4-r2`；本地副本：`C:\GitHub\_release-artifacts\v0.70.0\l3-v0701`
- 经典 Docker 29 `overlay2` 隔离实测：dangling 旧 KPanel 镜像删除；当前镜像、被停止容器引用的镜像和无关镜像保留，输出 `classic_docker_cleanup=pass`。
- 经典 Docker 实测日志 SHA-256：`e0b87d5683bd286cce6c1fcea300e578bbd4412cc4cf5c929fb0c813f7448407`
- v0.70.0 的独立正式 Chrome 四入口门禁全部通过；v0.70.1 没有 UI 业务差异，因此没有重复浏览器验收。结果文件 SHA-256：`635ed7bee616634e0878579ca5f05aff5fe39f2e768df81afb1aa8516c11641b`。

L3 首次增量 bundle 因缺少基线被隔离 clone 正确拒绝；随后改用包含完整历史的自包含 bundle 重跑通过。经典 Docker 首个夹具因删除最后一个 tag 时镜像被 Docker 直接回收，未形成 dangling 条件；改用升级覆盖同一 tag 的真实方式后重跑通过。这两项均为验收夹具问题，没有修改生产或绕过门禁。

## 不可变产物与应用市场

- OCI index（`0.70.1` 与 `latest`）：`sha256:ceae5b2ec62f7d93b96fdc0a0caff8817ced20e30887a4aef74a98da04f99813`
- Linux amd64 manifest：`sha256:1d0c92e9b204d9222f586abb2143bae9a836e0ebf722f7f9550e39ca94093e33`
- Linux arm64 manifest：`sha256:dee71ca66d434441fd6e398ac4397b6cf37ff570121a9b3e3f633a77cc13498d`
- 两个架构的 OCI version 为 `0.70.1`，revision 为 `9f124c42c175a8f1c2b2a2194c6a31e4bb89c8f5`。
- 154 从 Docker Hub 按不可变 OCI index 重新拉取并执行冷启动 E2E，输出 `image_e2e=pass`；日志 SHA-256：`d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。
- `kejilion/apps` commit：`c5cc79ce4acd7f7b373573952616dbabd2060b7d`
- apps 仅修改 `kpanel.conf`；其 Git blob 与 KPanel 冻结源文件一致，Linux 文件 SHA-256：`82f06ca32ce827ef8d0c9c72e65eed9180841a23cbc507237072b58a0807ef04`
- apps 生命周期由最终同一文件的 L3 lifecycle 覆盖，输出 `app_conf_lifecycle=pass`。

## 正式部署

### arena-154

- 端口：`8080`
- 上线后：版本 `0.70.1`，revision 精确匹配；Panel healthy、restart 0、OOM false；Agent active
- 数据：SQLite `quick_check`、JSON/JSONL 解析通过
- 升级前运行镜像：已清理
- 证据：`/root/kpanel-release-evidence/v0.70.1/arena154-20260813T100959Z`
- 备份：`/root/kpanel-backups/v0.70.1-preupgrade-arena154-20260813T100959Z.tar.gz`
- 备份 SHA-256：`b89baaa645ece8541edb8d8567e16a377ffb12de1ee20b75a29bcaba4b6e08a9`
- v0.70.0 镜像归档：`/root/kpanel-backups/v0.70.1-preupgrade-arena154-20260813T100959Z.image.tar`
- 镜像归档 SHA-256：`b5abfb0d602d63cf697c6f171b1397bfd2f112c0241f4158c984ad87740dfca9`

154 后置脚本首轮把有明确 `pre-0.28.0-*` 回滚标签的历史镜像也视为应删除对象而停止；这些镜像不属于本次升级遗留。验收随后收紧为精确核对本次升级前运行镜像并通过，产品和生产配置没有因此变更。

### production-108

- 端口：`5566`
- 上线后：版本 `0.70.1`，revision 精确匹配；Panel healthy、restart 0、OOM false；Agent active
- 数据：SQLite `quick_check`、JSON/JSONL 解析通过
- v0.70.0 升级前运行镜像、已知 v0.69 dangling 镜像以及其余未被容器引用的无 tag 官方 KPanel 镜像均已清理
- 最小健康核对：3/3 返回 `status=ok`、`version=0.70.1`
- 证据：`/root/kpanel-release-evidence/v0.70.1/prod108-20260813T101114Z`
- 备份：`/root/kpanel-backups/v0.70.1-preupgrade-prod108-20260813T101114Z.tar.gz`
- 备份 SHA-256：`137176b85642f5dace073a8f7e4df2dfe7ffb818151cddb7c04dfd25054aeafd`
- v0.70.0 镜像归档：`/root/kpanel-backups/v0.70.1-preupgrade-prod108-20260813T101114Z.image.tar`
- 镜像归档 SHA-256：`e916c25d6a0fbe95a97e616ddfb8d39ad1cb0fca3af0658704102d91990ace9b`

108 只执行停写一致性备份、标准应用市场更新、版本/健康/Agent/重启/OOM/数据完整性和镜像清理核对；未执行浏览器、功能、灰度、故障注入或长时间观察。

## 回滚方案

任一主机回滚时必须成套恢复，不允许只切镜像：

1. 停止 Panel 与 `kejilion-agent.service`，确认写入停止。
2. 校验该主机 `.tar.gz` 与 `.image.tar` 的 SHA-256。
3. 从归档恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 和 `kejilion-agent.service`。
4. `docker load` 对应 v0.70.0 镜像归档，并按备份中的 `old-image.id` 恢复 `docker.io/kjlion/kejilion-panel:latest`。
5. 执行 `systemctl daemon-reload`，启动 Agent，再用已恢复的 `.env` 与 Compose 执行 `docker compose up -d --force-recreate`。
6. 验证 v0.70.0、Panel healthy、Agent active、原端口、数据完整性与镜像 revision。

两台主机的备份均在停写状态生成，归档可读、镜像可加载、独立恢复目录 manifest/文件摘要和恢复后数据完整性已经通过。此次没有触发生产回滚。

## 遗留风险与工作流沉淀

- v0.70.1 仅修正更新后的镜像枚举，未改 API、数据格式、端口、Compose、Agent 权限或用户交互；没有新增业务兼容风险。
- 154 上被停止容器引用的浏览器候选镜像和显式历史回滚标签按安全边界保留，不属于“未被引用的更新遗留镜像”。
- 本轮没有新增长期项目规范；已有 `release-kpanel` v2.1、版本治理、环境策略、生命周期测试和验收模板已覆盖该流程。新增的经典 Docker 实测作为本次发布证据保留，没有另建重复工作流。
