# KPanel v0.88.1 发布验收记录

日期：2026-08-20

发布级别：L3（补丁）

候选提交 / 标签：`46981a51f1836d77b94be0826971a1cbde749b0c` / `v0.88.1`

上一稳定版本 / 回滚点：`v0.88.0` / `sha256:81b76fe59b15ceaed985421e79f81afd744eb9776fbdb2566b1464814cecfb99`

## 发布画像

- 业务域：AI 助手 Provider 请求历史重建。
- 变更面：仅调整 Panel 内部多工具调用与工具结果的消息批次重建，并补充回归测试、设计说明和补丁版本元数据；没有数据库、API、权限、前端或部署契约变化。
- 受影响用户旅程：同一模型响应包含多个 tool call、多个 tool result，随后继续向严格校验 Provider 发起下一轮请求。
- 未变化契约：隐藏 reasoning 继续只在内部保存；不会作为普通文本输出；单工具调用、非工具消息、会话数据格式及 Agent 协议不变。
- 风险等级及理由：低；改动局限于 AI 请求历史适配层，严格 Provider 合同由自动化模拟覆盖，未使用生产 AI 凭据执行写验收。

## 发布范围与未纳入内容

- 用户可见结果：多工具调用会话的后续请求不再因同一 Provider 响应被拆成多个 assistant message 而触发严格消息序列校验失败。
- 精确提交清单：`51c383b`（产品修复，等价来源提交 `1eed0b7`）、`0c1c399`（v0.88.1 版本准备）、`46981a5`（仅提高一个既有 512 图标压力测试的超时预算，产品逻辑未变）。
- 明确未纳入的分支、文件或后续事项：未合入其它工作树、未提交草稿或无关实验；没有 apps 空提交；没有连接或操作 108。

## 多维质量结论

| 维度 | 状态 | 证据 | 未验证风险或不适用依据 |
| --- | --- | --- | --- |
| 业务正确性与双端互通 | 已验证 | AI 请求历史定向回归、Go 全量测试、L3、候选/main CI | 未使用真实严格 Provider 凭据做在线 E2E；合同由确定性测试模拟 |
| 网络入侵与供应链安全 | 已验证 | govulncheck 可达漏洞 0、npm audit 0、Trivy 源码/镜像 0、OCI/受管脚本固定契约 | 不适用：没有新增网络入口、权限或依赖 |
| 稳定性、失败恢复与兼容 | 已验证 | Go 全量/AI 核心竞态/vet、Web 105 文件/798 测试、公开镜像 E2E、生产健康采样 | 第三方 Provider 未来若改变私有序列规则仍需新增相应契约夹具 |
| 性能与资源预算 | 已验证 | 生产三次采样 CPU 0.02%、内存 73.42 MiB/256 MiB，0 restart/OOM | 本补丁无常驻进程或高负载路径变化，不执行长时间 soak |
| 用户体验与可访问性 | 不适用 | 没有前端产品代码差异；Web 全量测试和生产构建通过 | 不新增浏览器人工视觉验收 |
| 数据、配置与迁移 | 已验证 | SQLite `integrity_check=ok`、Compose parse、停写备份独立解包/比对/旧镜像恢复核验 | 不适用：无 schema 或配置迁移 |

## 自动门禁

- 定向测试及结果：多工具调用/结果批次重建、严格消息序列、隐藏 reasoning 保留边界全部通过；既有 512 图标压力测试在完整套件中两次超过默认 5 秒，定向连续通过后仅将该测试预算提高为 10 秒，最终 Web 全量 105 文件/798 测试通过。
- `make verify-release` 环境和结果：arena-154 Debian 13、Docker 29.6.2、固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`；Go 全量/AI 核心竞态/vet、Web、i18n 2422/20、typecheck/build、双架构、安装安全、应用生命周期全部通过；证据 `/root/kpanel-release-evidence/v0.88.1-final`，日志 SHA-256=`b5ded0e6a99ad5d7aaccc6855b134cdc3fb7f501abc496e7a9a637ba1f0fa88a`。
- 候选 CI：run `32347000487` success；Dependency freshness run `32347000548` success。
- 主线 CI：run `32347338256` success；Dependency freshness run `32347338257` success。
- Release workflow：run `32347782549` success；Tag Dependency freshness run `32347782523` success。
- 安全扫描、镜像契约、SBOM/provenance：govulncheck、npm audit、Trivy 源码/镜像和 OCI 标签契约通过；双架构 provenance/SBOM attestation 保留。

## 依赖与技术栈变化

- `make dependency-report` 生成时间及检测源完整性：2026-08-20 Release L3；依赖策略、治理一致性和每日新鲜度门禁通过。
- 最近每日安全通告审计、EOL 复核状态及证据：候选、main、Tag 的 Dependency freshness 均 success；govulncheck 可达漏洞 0，npm audit 0，Trivy 0。
- 本版采用的依赖、工具链、基础镜像、Action、扫描器或受管脚本候选：没有新增或升级第三方依赖；沿用 Go 1.26.6、Node 24.18.0、现有固定 Actions/扫描器和受管脚本。
- 版本/锁文件/Action SHA/镜像 digest/脚本提交与摘要：Node 基础镜像 `sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd`；Go 基础镜像 `sha256:af8d6740070b8906d12eae1c3e3ea0957fb63f492051ea05e354c38ef9fe88df`；脚本 revision=`6fa7bcc7c2d15fe09d829cb9664941ff40bf4aaf`、SHA-256=`534a7a1866a7ab4f36229571b74492cb333c183a4375968141d3d40a9802b8d9`。
- 暂缓或拒绝候选、证据、负责人、复核日期和退出条件：不适用；没有待升级依赖候选。
- 升级后的兼容、安全、构建、性能资源和回滚结论：均通过；旧 OCI 和完整状态备份可执行成套回滚。

## 隔离真机与浏览器验收

- 主机/发行版/架构/运行时版本：arena-154 Debian 13、x86_64、Docker 29.6.2；发布构建覆盖 linux/amd64 与 linux/arm64。
- 环境策略 ID 与允许用途：`arena-154`，candidate-validation、performance-validation、production-safety-check 和 production-deploy。
- 使用的精确候选或公开产物：源码 `46981a51f1836d77b94be0826971a1cbde749b0c`；L3 验证镜像 index `sha256:a8a4d52c3848b7ee5015a8069b97f861ea0e6a691c1e55fd25b6c7b8518f2941`；公开 OCI `sha256:0710dce2c8272f657bc171c4abf576abe4ec968d267f209c93ccce5d6c645660`。
- 后台作业 ID、终态、退出码、超时、证据目录、命令规格路径及 SHA-256：L3 同步作业 exit 0；证据目录 `/root/kpanel-release-evidence/v0.88.1-final`，L3 日志 SHA-256=`b5ded0e6a99ad5d7aaccc6855b134cdc3fb7f501abc496e7a9a637ba1f0fa88a`。
- 测试窗口/循环数及风险依据（无 soak 时写不适用依据）：生产 3 次健康/资源采样；本补丁没有常驻服务或资源模型变化，不执行长时间 soak。
- 受影响用户旅程、视口、100%/125%/200% 缩放、最小计算字号、主题、键盘/焦点、语言和失败态：不适用；无前端产品代码差异，Web 全量与生产构建覆盖现有界面。
- 宿主机写入、失败注入、重启恢复和回滚结果：停写备份完成独立恢复核验；正式升级前旧 v0.88.0 Panel/Agent 已原位恢复 healthy/active。
- 未执行场景及原因：未调用生产 AI Provider 或读取 AI 凭据；真实严格 Provider 在线 E2E 由管理员后续正常使用观察，避免产生计费、泄露或外部写操作。

## 发布产物与公开仓库复核

- GitHub Release：[v0.88.1](https://github.com/kejilion/KPanel/releases/tag/v0.88.1)，非 draft、非 prerelease。
- Docker 版本与 `latest` OCI index：两者均为 `sha256:0710dce2c8272f657bc171c4abf576abe4ec968d267f209c93ccce5d6c645660`。
- `linux/amd64`、`linux/arm64` digest：amd64 `sha256:c36eb558a63b3e41e5d72dab96e48c7000b403a0d90dfe3c904e7f3e38423eec`；arm64 `sha256:ef5e6be7c2ac08125b8239a733a29de4366333f4552e5106ec9377e34461914d`。
- 附件及 `SHA256SUMS`：Agent/Node 双架构、部署包、LICENSE、THIRD_PARTY_NOTICES 与 SHA256SUMS 齐全；SHA256SUMS asset digest=`sha256:0c523457f445bf368a98cd61b6672e69c3f89ee406adc684bfd66a87321b7c87`。
- 公开镜像 `image_e2e=pass`：arena-154 隔离端口 `18098` 从公开仓库回拉通过；日志 `/root/kpanel-release-evidence/v0.88.1-final/public-image-e2e.log`，SHA-256=`dc3687111741bf7506f94d7fd188b756dd02dc1d74fc7acf25b26ad89d8ca74e`。
- `kejilion/apps` / `kejilion.sh` 契约结论：`packaging/kejilion-app/kpanel.conf` 与 apps main 归一化内容一致，无需空提交；公开受管脚本 revision/hash 与 OCI 标签和镜像内 `/release/kejilion.sh` 一致。

## 生产部署安全核对

- 生产目标和部署授权范围：仅 `arena-154`；用户已授权 v0.88.1 正式升级。
- 验证/灰度环境（必须来自 `environment-policy.json`，不得包含 `prod-108`）：`arena-154` 隔离容器和回环端口。
- 正式部署环境（默认 `arena-154`；不得包含 `prod-108`）：`arena-154`。
- `prod-108`：禁用全部 KPanel 操作；确认本次未连接、未备份、未部署、未升级、未核对：已确认。
- 部署前版本、健康、备份位置及摘要：v0.88.0，Panel healthy、Agent active、restart=0、OOM=false；备份 `/root/kpanel-backups/v0.88.1-preupgrade-arena154-20260820T082653Z`；状态包 SHA-256=`840fdd9fe2da4087c4d8086805fd2e2848739d0fb7d8d3c4170f9e4dc4898d7d`，旧镜像 tar SHA-256=`a6c1391ac918058ce1d3b9d382693fbc74301e8b54cca5e4a3fde77a85354074`。
- 部署命令/入口：标准应用市场更新入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，进度到 `KPANEL_PROGRESS 100`。
- 部署后版本、Panel/Agent 状态、重启、日志、数据完整性和公网入口：Panel/Agent 0.88.1，Panel healthy、Agent active、双方 restart=0、OOM=false；三次健康/资源采样稳定，SQLite integrity=ok，近 15 分钟 panic/fatal/OOM/协议错误 marker 为 0，HTTPS 健康入口返回 v0.88.1。
- 生产已执行写操作：停写一致性备份、旧服务原位恢复核验、标准应用市场升级；运行时脚本只按既有安装契约设置 `permission_granted=true`，归一化后与固定脚本逐字节一致。
- 仅在隔离真机执行、未在生产执行的场景：公开镜像 E2E、构建/安全门禁；未在生产调用真实 AI Provider。

## 回滚

- 源码/tag：`v0.88.0` / `14962aeff7186e846f52db3c989d5220a854869f`。
- 镜像 digest：`sha256:81b76fe59b15ceaed985421e79f81afd744eb9776fbdb2566b1464814cecfb99`。
- 数据/配置备份：`/root/kpanel-backups/v0.88.1-preupgrade-arena154-20260820T082653Z`，含 Compose、`.env`、Agent/脚本、Panel/Agent 数据、apps 配置、systemd unit 和旧镜像 tar；独立解包、关键文件 cmp、Compose parse、SQLite integrity、摘要和 `docker load` 均通过。
- 回滚步骤和回滚后复核：停写；从备份加载旧镜像并成套恢复 `kpanel` 目录、apps 配置和 systemd unit；`systemctl daemon-reload` 后启动 Panel/Agent；核对 v0.88.0、health、restart/OOM、SQLite、Compose 和公网入口。
- 回滚后生产实际版本与健康状态：未执行回滚；当前 v0.88.1 healthy。
- GitHub Latest、Docker `latest` 与标准更新入口实际指向：均为 v0.88.1 / `sha256:0710dce2c8272f657bc171c4abf576abe4ec968d267f209c93ccce5d6c645660`。
- 公共默认更新通道决策：不适用；所有生产门禁通过，无需恢复上一稳定默认版本。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-20T15:25:50+08:00
- 候选冻结时间：2026-08-20T15:51:41+08:00
- 生产完成时间：2026-08-20T16:29:56+08:00
- 提交到生产用时：1.07 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：11
- 其中生产写操作开始后异常次数：3
<!-- kpanel-release-process-metrics:end -->

流程异常均与产品载荷无关。生产写入前包括三个候选审计/Runner 上下文修正、两次既有压力测试超时、一次远端循环转义产生无效证据、一次 Windows 主机 npm 可选原生绑定缺失和一次远端清理命令转义；最终均在同一冻结源码或仅含测试预算修正的新冻结 SHA 上重跑并通过。生产写入后，一次备份脚本错误依赖未安装的 `sqlite3` 命令、一次安装脚本归一化格式写错、一次 PowerShell JSON 断言转义失真；三次均 fail-closed，旧服务或当前服务保持健康，随后以 Python 标准 SQLite 读取和独立只读验收脚本完成同等门禁。没有产品失败、数据损坏、回滚或门禁逃逸。

## 遗留风险与后续准入

- 未验证风险：没有使用真实严格 Provider 凭据执行在线多工具 E2E；第三方 Provider 未来可能调整私有消息序列规则。
- 已实现待实机准入：管理员可在正常 AI 助手使用中观察真实多工具调用；若出现 Provider 特有错误，应保留脱敏请求形态并新增确定性合同夹具。
- 不阻断本版的理由：根因和修复均位于确定性历史重建层，严格 Provider 合同模拟、Go 全量/竞态、L3、CI、公开产物和生产健康全部通过；在线调用会产生外部计费或凭据风险，不属于生产健康验收。
- 后续应进入的自动门禁或专项工作流：继续保留同一 Provider 响应多 tool call/tool result 合批、隐藏 reasoning 不外泄和严格消息序列回归。
