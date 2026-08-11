# KPanel v0.64.0 发布验收记录

日期：2026-08-11

发布级别：L3

正式提交 / 标签：`7346670858932283d2c7d405f72400ba034ac998` / `v0.64.0`

上一稳定版本 / 回滚点：`v0.63.1` / `6010b0ad281ad0b586d888f7c4f9a5c7d20197af` / `sha256:94519b0bcfc539055a99b6a8ba91f3c691d9096e00dce489eba4ab5f4db050f8`

## 发布画像

- 新增独立“系统中心”入口，集中展示日常维护、基础配置、登录与安全、网络与流量、性能优化、危险操作六个分区。
- 概览保留 3×2 高频系统工具，并提供“更多设置”进入系统中心；桌面模式同步增加系统中心应用和窗口路由。
- 系统更新、系统清理、虚拟内存、SSH 防御、DNS、BBR 共 6 项显示“推荐”标记。
- 本版本只变更 `docs/` 与 `web/` 功能文件及版本元数据；没有新增或修改 Go API、数据库、端口、Compose、systemd、Agent 权限、Dockerfile、`kejilion.sh` 协议或应用市场契约。
- 页面只复用主线已有系统工具与一键调优实现，不自动执行宿主机写操作；风险等级为中等，主要风险集中在入口与响应式布局回归。

## 候选范围与净差异

- 冻结源分支为 `feature/system-center-page-v2`，基线 `9e664282e9107a59c4be0ddf781b1aefeb18a2f5`，源提交为 `3beb6d6138dd29d43d715056b480f5a39e9521c3`、`f59e5762d0fe573e9cb73a438dab97c33ca650cf`。
- 干净候选中的等价重放提交为 `b21c668006c5d5a5d93a2f92936ae0d4aa66ac8e`、`df66398de0f4da48d8a63246a328826ebe142261`；patch-id 分别为 `28b3cd001d4ba579f4e7cab42944c9576aa50a82`、`98ef6b11b920e95eff9dd1583038766290039ea3`。
- 功能净差异精确为 26 个 `docs/` / `web/` 文件；与源分支 HEAD 及旧版已完成浏览器验收的 `0ed36c2` 功能树逐字节一致。版本冻结提交为 `7346670`。
- 原实现中最新主线已经拥有的一键调优后端、API、`SystemTuningDialog`、Dockerfile 和脚本固定内容均为零差异，未重复纳入。
- 未整合其他未冻结工作树；源工作树保持 clean，发布候选分支已由 Release workflow 在成功后删除。

## 自动门禁与 L3

- Windows 冻结前验证：`npm ci` 为 0 vulnerabilities；Web 86 个测试文件、615 项测试通过；i18n 2122 条、typecheck、生产构建和 `git diff --check` 通过。
- 154 从 bundle 构建精确 detached 候选并执行完整 L3：`make verify-release`、Go 全量测试、核心 race、`go vet`、Linux amd64/arm64 Agent/Panel 构建、Web 全量测试与构建、Trivy 源码/镜像/密钥扫描、运行时约束和应用配置生命周期全部通过。
- L3 日志 `/root/kpanel-release-evidence/v0.64.0/l3-verify-release.log`，SHA-256 `e340dbc8ea3719a5fc4028aaee08a38195c18c404c282ac85348d2dc524640b4`；应用配置生命周期日志 SHA-256 `2e5731ab77ce82c4dc53d45bf14e09b46f522d7cca3da33f7d2fcf0cbc7a0c3f`。
- 候选 bundle：`C:\GitHub\_release-artifacts\v0.64.0\kpanel-v0.64.0-7346670.bundle`，SHA-256 `605846f4e45ef0c8fef053bf660d3ac9ca8979489096ee562bc8c62631edd32d`。
- 候选 CI `31502483493`、主线 CI `31502867642`、Release run `31503541295` 均成功；正式 Tag 只在候选与主线 CI 成功且三方 SHA 均为 `7346670` 后创建。

## 脚本与应用市场契约

- 镜像继续固定 `kejilion.sh` 提交 `28f89c1b34df4b25e6ef9b144c328fdea75dbac9`，公开 raw SHA-256 为 `0583f7cd5be1f0bb6ec48d92e2cf224bfabfafada5788658bda4414ba9561229`；L3、公开镜像 E2E 与生产安装文件均再次核对通过。
- KPanel 与 `kejilion/apps` 的 `kpanel.conf` blob 均为 `7289637a42b8209b301772139ff4404d08e196d2`；apps 主线保持 `e7f90760b71cfe69c8b05af40131ab89739eb0f5`，没有制造空提交。
- CI 与 Release workflow 相对上一稳定版未变，不存在发布门禁漂移。

## 浏览器与用户体验验收

- 冻结功能树已完成经典模式、桌面模式、深浅主题、中英文和 390px 手机竖屏检查；系统中心六分区、推荐标记精确 6 项、概览入口与桌面应用路由正常，无页面级横向溢出。
- 生产升级后经只读 SSH 隧道打开 154 登录页，标题为“登录 · KPanel”，用户名、密码、安全登录控件正常，控制台错误为 0。
- 未读取或传输生产凭据，因此没有在生产登录后重复执行系统中心工具；本轮也未在生产执行一键调优或任何系统工具写操作。生产浏览器摘要 SHA-256 为 `c9d17566a972975e5b10ab223363b2e88fdc9960f1133d213f4b10170e17c6bd`。

## Release 与公开镜像

- [GitHub Release v0.64.0](https://github.com/kejilion/KPanel/releases/tag/v0.64.0) 发布成功，非 draft，包含四个 Linux 原生二进制、部署归档、`SHA256SUMS`、LICENSE 和第三方许可清单，共 8 个附件；8 个公开下载入口均返回 HTTP 200。
- `docker.io/kjlion/kejilion-panel:0.64.0` 与 `latest` OCI index 均为 `sha256:bbe9c148a70f7362dc9b78f0de084cf65e202edeb7cacffaf2104d7ef91dcfb3`。
- `linux/amd64`：`sha256:3334d5594963392af6d4280f9ff08aa7ca7b76731adb26d0c5a2f6c103f4cad1`；`linux/arm64`：`sha256:7460b4dd09f7738215a53896957067db15bd184146606baa85f171429d1fb146`。
- 154 从公开仓库重新拉取后，版本、源码修订、双架构、脚本修订/摘要、只读根文件系统、CPU/内存/PID 限制及健康检查均通过，输出 `image_e2e=pass`；日志 SHA-256 `8e43ecd7bd46dd931fd9119e1655f348fd63a59634671770fdd9b9bd7cf8fb93`。
- `SHA256SUMS`：Agent amd64 `a2dae70f...`、Agent arm64 `92c841da...`、Node amd64 `aa4607be...`、Node arm64 `0e826230...`、部署归档 `324aeb4e...`；完整值以 Release 附件为准。

## 生产部署与观察

- 生产目标为 `arena-154` 的 `http://154.36.153.9:8080`；`kp.kejilion.pro` 是另一实例，不作为本次生产真源。
- 升级前版本为 `v0.63.1`，Panel running/healthy/0 重启/OOM false，Agent active；运行镜像精确为上一稳定 index 摘要。
- 停写一致性备份目录：`/root/kpanel-backups/v0.64.0-preupgrade-20260811T145747Z`；归档 `/root/kpanel-backups/v0.64.0-preupgrade-20260811T145747Z.tar.gz`，SHA-256 `50a03e984c1a491e90bfce3d2c2e69c3285903db19317a1e99862c74b4ece01c`；manifest SHA-256 `3e1739bb99011791bafc82d9b03d1bd268a629aa54262d26a88087029070c17c`。SQLite quick check、JSON 解析、逐文件清单和独立解包恢复校验均通过，旧版恢复健康后才开始升级。
- 使用标准入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel` 升级成功，未触发应用回滚。
- 升级后本机与公网健康接口均为 `0.64.0/ok`；Panel running/healthy/0 重启/OOM false；Agent `0.64.0 v1alpha1`、active/running、`NeedDaemonReload=no`、`ExecMainStatus=0`。
- 镜像、源码、固定脚本摘要、SQLite/JSON 完整性、`.env`、Agent 配置/token、AI/集群身份密钥和 apps 配置均通过核对；最近日志未发现 panic、fatal、segmentation fault 或 OOM。
- 60 次、2 秒间隔持续采样全部通过，时间为 `2026-08-11T14:59:55Z` 至 `2026-08-11T15:02:07Z`；采样 TSV SHA-256 `41b00a366ddc238b8610d4fce895c785824a9ce52c3db84ee006c89e554dc5fe`。
- 生产证据目录：`/root/kpanel-release-evidence/v0.64.0/production-20260811T145747Z`；上线后验证摘要 SHA-256 `6e8cc50609078157164fa65f73cb749b419e0da4f626de7e59a6a345e60708f8`。

## 回滚

- 源码/Tag：`v0.63.1` / `6010b0ad281ad0b586d888f7c4f9a5c7d20197af`。
- 镜像：OCI index `sha256:94519b0bcfc539055a99b6a8ba91f3c691d9096e00dce489eba4ab5f4db050f8`，amd64 子清单 `sha256:65a58abe1322ccb34eafd197a024edb3ba119b730c21df94ba909e1af69b3149`。
- 停止 Panel 与 Agent，将 Compose 镜像固定回上一版不可变摘要；只有数据或配置也需回退时，才在停写状态从已验证备份恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 和 systemd 单元/symlink，执行 `systemctl daemon-reload` 后启动 Agent 与 Panel。
- 回滚后复核版本、容器健康/重启/OOM、Agent 状态、脚本摘要、SQLite/JSON、稳定身份文件和公网健康接口。GitHub Tag/Release 保持不可变；如需恢复公共默认更新通道，必须通过新的规范发布，而不是改写 `v0.64.0`。

## 节奏、限制与沉淀

- 首个源功能提交：2026-08-11T22:24:05+08:00；最终候选冻结：2026-08-11T22:30:41+08:00；生产持续采样完成：2026-08-11T23:02:07+08:00。源功能提交到生产完成约 38 分钟，冻结到生产完成约 31 分钟。
- 发布流程中没有应用回滚。公开镜像 E2E 首轮仅因验收脚本在 Docker 健康状态仍为 `starting` 时过早断言而失败；镜像直接健康检查已成功。脚本增加状态收敛等待后重跑通过，产品和镜像未改写。
- 未验证风险：生产登录后的系统中心视觉未重复截图；原因是没有读取或传输生产凭据。冻结功能树的完整浏览器验收、Release 精确源码修订、公开镜像 E2E 和生产健康验证共同构成本版准入依据。
- 本轮复用项目现有 `release-kpanel v1.7` 与版本治理流程，没有新增重复工作流；验收事实已沉淀为本文件。
