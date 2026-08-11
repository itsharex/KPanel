# KPanel v0.65.0 发布验收记录

日期：2026-08-12

发布级别：L3

正式提交 / 标签：`f6c78c4d70473cd393e02e427cf01281968132b1` / `v0.65.0`

上一稳定版本 / 回滚点：`v0.64.0` / `7346670858932283d2c7d405f72400ba034ac998` / `sha256:bbe9c148a70f7362dc9b78f0de084cf65e202edeb7cacffaf2104d7ef91dcfb3`

## 发布画像

- 终端和诊断页在手机竖屏下改用可展开抽屉承载连接、会话、筛选与详情区域，避免主工作区被侧栏挤压。
- 经典模式下统一终端类工作区的页面结构、标题、间距和滚动边界；桌面模式保持既有独立窗口行为。
- 纳入依赖生命周期治理：统一依赖源凭据隔离、补齐依赖新鲜度监控及治理门禁。
- 本版本不包含嵌入式浏览器核心提交 `062ac9e`。该功能仍缺 Relay、共享密钥、双 Origin、回滚和真实兼容性闭环，判定为生产 No-Go。
- 版本按新增用户可见能力定义为 MINOR，版本号 `0.65.0`。

## 候选范围与净差异

- 候选基线为 `10460af9`，终端/诊断源提交为 `a78bf16`、`8f88b56`；干净候选中的等价重放提交为 `b3b3d53`、`77f4850`，patch-id 一致。
- UI 净差异精确为 9 个 `web/` 文件；重放候选与冻结源功能树逐字节一致。
- 治理提交 `95849ec`、`37e62d0`、`10460af` 已在基线主线中，未重复制造提交。
- 版本冻结提交为 `f6c78c4`；候选工作树在冻结、推送和发布前均为 clean。
- 未拼入其他未冻结工作树；嵌入式浏览器 Phase 1 明确排除。

## 自动门禁与 L3

- Windows 候选验证：`npm ci` 安装 252 个包且 0 vulnerabilities；Web 86 个测试文件、618 项测试全部通过；i18n 2127 条、20 个按页加载语言包通过；typecheck、生产构建和 `git diff --check` 通过。
- 主入口 JavaScript 为 96.41 KiB，gzip 33.88 KiB。
- 154 使用精确候选 bundle 在隔离验证器中完成 L3：Go 全量测试、核心包 race、`go vet`、Web 全量测试与构建、安装安全检查、`govulncheck`、`npm audit`、Trivy 源码/镜像扫描、Linux amd64/arm64 Agent/Panel 构建、最终镜像、固定脚本契约和应用配置生命周期全部通过。
- 154 宿主最初没有 Go，改用固定 `golang:1.26.5-alpine` 与 `node:24.18.0-alpine` 组成的隔离验证器；首个验证器缺 Buildx 后补齐并重跑。两次均为验证环境修正，没有修改产品代码或候选提交。
- 候选 bundle：`C:\GitHub\_release-artifacts\v0.65.0\kpanel-v0.65.0-f6c78c4.bundle`，SHA-256 `f02a6c30c915bb1032493b08e0b8b1c4c5d4f3dd48fee628b580e7a4036541c8`。
- L3 日志：`/root/kpanel-release-evidence/v0.65.0/l3-verify-release.log`，SHA-256 `16d8a53b6954de42d98372beb43b51b43aa54a57e82f2346f3744e6f552aae93`。
- 候选 CI `31520246675`、候选 Dependency freshness `31520246747`、主线 CI `31520583719`、主线 Dependency freshness `31520583754`、Release run `31520891887` 全部成功。

## 脚本与应用市场契约

- 镜像继续固定 `kejilion.sh` 提交 `28f89c1b34df4b25e6ef9b144c328fdea75dbac9`，公开 raw SHA-256 为 `0583f7cd5be1f0bb6ec48d92e2cf224bfabfafada5788658bda4414ba9561229`；L3、公开镜像 E2E 和生产安装文件均再次核对通过。
- KPanel 与 `kejilion/apps` 的 `kpanel.conf` blob 均为 `7289637a42b8209b301772139ff4404d08e196d2`；apps 主线保持 `e7f90760b71cfe69c8b05af40131ab89739eb0f5`，本轮没有 apps 变更或空提交。

## 浏览器与用户体验验收

- 冻结源功能树已完成 390x844 手机竖屏真实交互：终端选择器抽屉可打开、操作和关闭；经典模式终端工作区布局一致，无页面级横向溢出。
- 9 个 UI 文件在源功能树与正式候选中逐字节一致；候选的 618 项测试、生产构建和完整 L3 均通过。
- 尝试通过 SSH 隧道分别使用 Chrome 扩展和应用内浏览器复验隔离候选，但两种连接均在打开页面时超时；相关隧道、候选容器和临时数据已清理。因此本记录不将候选浏览器重检标记为通过，准入依据为原冻结树真实浏览器证据、文件同一性及候选自动化/L3。
- 生产环境没有执行需要凭据的写操作，也没有读取或传输生产登录凭据。

## Release 与公开镜像

- [GitHub Release v0.65.0](https://github.com/kejilion/KPanel/releases/tag/v0.65.0) 发布成功，非 draft、非 prerelease，共 8 个附件：Agent/Node 的 amd64 与 arm64 二进制、部署归档、`SHA256SUMS`、LICENSE 和第三方许可清单。
- annotated tag object 为 `9232080a5cd11d56703c41ee352ee96131b360c3`，peel 后精确指向 `f6c78c4d70473cd393e02e427cf01281968132b1`。
- `docker.io/kjlion/kejilion-panel:0.65.0` 与 `latest` OCI index 均为 `sha256:8947e7aa281df47612d03a29cd78556b7c60b00a59214fd542a80b58eff57c22`。
- `linux/amd64` 子清单为 `sha256:73afda66298cfa6e46d655e8c574ee15dc0e045fe6ba2299be074021a5ef2173`；`linux/arm64` 子清单为 `sha256:fa06659353caed0951423324c007eb2d0d74f40a47a04d08a6963a5b68d492c5`。
- 154 从公开仓库拉取并执行不可变镜像 E2E，输出 `image_e2e=pass`；证据日志 SHA-256 `d6918030f72af42d7c376ceb7a1f902705a8473b11ff4ccacb39706e81322eb9`。

## 生产部署与观察

- 生产真源为 `arena-154` 的 `http://154.36.153.9:8080`；`kp.kejilion.pro` 不属于本次目标。
- 升级前为 `v0.64.0`，Panel healthy、0 restarts、OOM false，Agent active；运行镜像为上一稳定 index 摘要。
- 停写一致性备份目录：`/root/kpanel-backups/v0.65.0-preupgrade-20260811T181413Z`；归档：`/root/kpanel-backups/v0.65.0-preupgrade-20260811T181413Z.tar.gz`；SHA-256 `6c0dd4aeecbd9843eb1880ff3ef682901d3e092af56d44135cbe53c9dcc1fa2e`。源数据与独立解包后的 SQLite、JSON、JSONL 完整性及清单哈希均通过，恢复旧版健康后才执行升级。
- 使用标准入口 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel` 升级成功，未触发应用回滚；更新日志 SHA-256 `2f32eeaa6c47ddeb51fd633953c655cf3294dd1b3c31e12965eabf88ae9a6e95`。
- 升级后本机与公网健康接口均为 `0.65.0/ok/v1alpha1`；Panel running/healthy、0 restarts、OOM false；Agent `0.65.0 v1alpha1`、active，`NeedDaemonReload=no`、`ExecMainStatus=0`、`NRestarts=0`。
- 容器源码标签为 `f6c78c4`，镜像 index 精确匹配正式摘要。宿主 Agent 与镜像内 Agent SHA-256 均为 `8dab6125c84670b0886b92e98e059598ac7fc6af622d7eb863bae906d35ba451`。
- 安装后的 `kejilion.sh` 因更新器按规范继承 `permission_granted`、`ENABLE_STATS`、`canshu`，其文件 SHA-256 为 `d73231f146f7398d7b50133695faf2116134fbfe33a7b94068e277cc7b82df55`；将这三个宿主配置归一为默认值后，与固定 raw 摘要 `0583f7cd...` 精确一致。
- `.env`、Agent 配置、Compose、systemd、token、AI secret、集群身份和 apps 配置哈希与备份一致；SQLite quick check、5 个 JSON 和 14621 个 JSONL 文件解析通过；最近日志没有 panic、fatal、segmentation fault 或 OOM。
- 60 次、2 秒间隔持续采样从 `2026-08-11T18:19:38Z` 到 `18:21:44Z` 全部通过：本机/公网版本与健康状态正确，Panel 无重启/OOM，Agent 始终 active。采样 SHA-256 `91717cbc2c1b449d949ae4a8ff297ada71fdc38aaca96af8ba9c05d0969cd350`。
- 生产证据目录：`/root/kpanel-release-evidence/v0.65.0/production-20260811T181413Z`；上线验证摘要 SHA-256 `c1d8b1e0f48be0324e331720af452009810fd75cf3e9cb4a93b58277252e9d4e`。

## 回滚

- 源码/Tag：`v0.64.0` / `7346670858932283d2c7d405f72400ba034ac998`。
- 镜像：OCI index `sha256:bbe9c148a70f7362dc9b78f0de084cf65e202edeb7cacffaf2104d7ef91dcfb3`；amd64 子清单 `sha256:3334d5594963392af6d4280f9ff08aa7ca7b76731adb26d0c5a2f6c103f4cad1`；arm64 子清单 `sha256:7460b4dd09f7738215a53896957067db15bd184146606baa85f171429d1fb146`。
- 停止 Panel 与 Agent，将 Compose 镜像固定回上一稳定不可变摘要；仅当数据或配置也需回退时，才在停写状态从已验证备份恢复 `/home/docker/kpanel`、`/root/apps/kpanel.conf` 及 systemd 单元/symlink，执行 `systemctl daemon-reload` 后启动 Agent 与 Panel。
- 回滚后复核版本、容器健康/重启/OOM、Agent 状态、脚本摘要、SQLite/JSON、稳定身份文件和公网健康接口。GitHub Tag/Release 保持不可变；公共默认更新通道只能通过新的规范发布恢复，不能改写 `v0.65.0`。

## 节奏、限制与沉淀

- 首个本版功能提交：2026-08-11T23:40:38+08:00；候选冻结：2026-08-12T01:37:37+08:00；Tag：02:05:06+08:00；Release 发布：02:09:42+08:00；生产持续采样完成：02:21:44+08:00。
- 功能提交到生产完成约 2 小时 41 分钟，冻结到生产完成约 44 分钟；没有应用回滚、产品热修或重复版本。验证器环境有两次可审计修正，候选代码未变化。
- 已知限制为候选浏览器连接超时；嵌入式浏览器核心继续保留为未上线项，需完成生产级设计与真实兼容性验收后另立版本。
- 本轮复用项目现有 `release-kpanel v1.7` 与版本治理流程，没有新增重复工作流；验收事实沉淀在本文件中。
