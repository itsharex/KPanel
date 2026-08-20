# KPanel v0.89.1 发布验收记录

日期：2026-08-20

发布级别：L3（补丁版本）

候选提交 / 标签：`89fb8aadea6c1eef970d8f72a3d7c4a16973db3d` / `v0.89.1`

上一稳定版本 / 回滚点：`v0.89.0` / `sha256:d04966d68fc7d1a7be5b68e12b32135455b54e52926ec6cec537da79e9690d19`

## 发布范围

- 明确一键跑分由服务器本机执行性能探针、由服务器出口执行网络探针，浏览器只启动任务和展示结果。
- 将综合结果重排为“综合评分 → 性能 → 网络”，区分下载/上传、磁盘读/写及出口网络信息。
- 统一结果卡片的字号、摘要对齐与窄屏收缩行为；发布候选验收中发现并修复 390px 下内部容器宽出 59px、右侧分数被裁切的问题。
- 没有改变跑分算法、Agent 协议、权限模型、数据库 schema、第三方依赖或应用市场契约；`kejilion/apps` 的 `kpanel.conf` 与候选一致，无需空提交。
- 未纳入旧功能分支的重复历史或其它工作树草稿；未连接、测试或部署 108。

## 自动门禁

- 最终 L3：arena-154 固定 Runner `sha256:b593c0ffe32e80a6a6ae8fcac38ed916587dbf698a6b6a55fa33887298737148`，标记 `L3 release verification completed`；日志 SHA-256=`da79ffef47b0415197612f0627aeaaedade01ca96b0b31831ea3ba88bd2fa5de`。
- 覆盖：Go 全包、`panel/auth/dockerx` 竞态、govulncheck 可达漏洞 0、npm audit 0、Trivy 源码/镜像 0、Web 106 文件/810 测试、i18n 2424/20、typecheck、production build、linux/amd64 与 linux/arm64、安装安全和应用生命周期。
- 候选 CI `32383176187`、候选依赖新鲜度 `32383176231`、main CI `32383518353`、main 依赖新鲜度 `32383518276` 均 success。
- Release workflow `32383975128` 与 Tag 依赖新鲜度 `32383975230` 均 success。

## 界面与真实链路验收

- 本地精确候选在正式浏览器中复核 1200×700 深/浅主题和 390×844 手机视口：结果最小可见字号 12px；综合评分、性能、网络层级清晰；页面及结果容器横向溢出均为 0；控制台 error/warning 为 0。
- 390px 首轮发现网络结果的 min-content 撑宽容器，已以聚焦 `min-width: 0` 约束修复并增加布局回归测试；最终复验分数、卡片和正文完整可见。
- arena-154 精确候选镜像真实执行 `native-comprehensive`：Panel→Agent→CPU/内存/磁盘/网络探针→结果回读成功，progress=100、parser=`kpanel-native-v1`，Panel healthy、Agent active、restart=0、OOM=false；结果 SHA-256=`2b2f29d433dff714b961b1d148a2494631e26e50bad78b87dc1ad3b1978b32fb`。
- 正式公开 OCI 独立回拉后再次执行同一真链路并成功；结果 SHA-256=`13567c05e548e386108b3ffbda42e58ae5f2d3e8e1f4a8810fb492dd4317007e`。
- 外部探测点或 IP 风险接口将来不可用时仍可能产生明确降级结果；不得把上游可用性等同于 KPanel 算法或 UI 故障。

## 发布产物

- GitHub Release：[v0.89.1](https://github.com/kejilion/KPanel/releases/tag/v0.89.1)，非 draft、非 prerelease；Tag 解引用到 `89fb8aadea6c1eef970d8f72a3d7c4a16973db3d`。
- Docker `0.89.1` 与 `latest` OCI index：`sha256:755cad031673e9bc44dc8b43ed7b62012a240d708cdf8d14c6c593810e1a73d8`。
- `linux/amd64`：`sha256:f420978372527289b8387e82a9c95a661f807aa4bd83410b1f37fd356b5ca3e3`；`linux/arm64`：`sha256:70fe16cab5fcc4f7b346b13b4754d5141d2c5d4e53f782e3b837f0936e63e3d1`。
- 公开镜像 version=`0.89.1`、revision=`89fb8aa...`、script revision=`6fa7bcc...`、script SHA-256=`534a7a18...`；Release `SHA256SUMS` asset digest=`sha256:8a7c3bdecbd210a339175e83203e022fbee0878d285328a39a2d84fd6ebed2fa`。

## 生产部署与回滚

- 生产目标仅 arena-154。部署前为 v0.89.0，Panel healthy、Agent active、restart=0、OOM=false。
- 停写一致性备份：`/root/kpanel-backups/v0.89.1-preupgrade-arena154-20260820T151442Z`；状态包 SHA-256=`3dc03df76a2364b28b25f9df6d549d705e439adb56666a14c6f4458c60f5f69c`，旧镜像归档 SHA-256=`ac89e9bd7db0eab2e1247d6823e7e5b1420a7cda85d638e66182c5e3542daa0e`。
- 备份已独立解包、关键文件逐字节对比、SQLite integrity、Compose 解析、旧镜像加载及原版本恢复健康核验。
- 标准更新入口：`KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update bash /home/docker/kpanel/bin/kejilion.sh app kpanel`，完成到 `KPANEL_PROGRESS 100`。
- 部署后 Panel v0.89.1 healthy、Agent active、restart=0、OOM=false，公网健康正常；三次资源采样 CPU 0.01%～0.03%、内存 74.08 MiB/256 MiB；SQLite integrity=ok，近 15 分钟无 panic/fatal/OOM/协议错误。
- `.env`、`agent.env` 与 `/root/apps/kpanel.conf` 升级前后哈希一致，现有生产配置未被覆盖。生产验收结果 SHA-256=`17fabda777e6c3b9e52e63f2f191f84c21fa3b6a16612faad15ad8e6078e37d0`。
- 回滚时停写，加载上述旧 OCI，成套恢复备份中的 `/home/docker/kpanel`、应用市场配置和 systemd unit，执行 `systemctl daemon-reload` 后启动 Agent/Panel，并复核 v0.89.0、SQLite、Compose、restart/OOM 与公网健康；禁止只替换镜像或单独恢复配置。

## 交付节奏数据

<!-- kpanel-release-metrics:start -->
- 首个纳入提交时间：2026-08-20T22:20:38+08:00
- 候选冻结时间：2026-08-20T22:41:53+08:00
- 生产完成时间：2026-08-20T23:16:12+08:00
- 提交到生产用时：0.93 小时
- 是否回滚、紧急热修复或重复发布：否
- 若发生失败，发现时间、恢复时间和逃逸门禁：不适用
<!-- kpanel-release-metrics:end -->

<!-- kpanel-release-process-metrics:start -->
- 已记录发布流程异常或无效证据拦截次数：2
- 其中生产写操作开始后异常次数：0
<!-- kpanel-release-process-metrics:end -->

两次拦截均在生产前闭环：首次浏览器验收发现手机端内部裁切并重新冻结 SHA；首次重建 bundle 未包含业务事实基线 Tag，治理门禁按设计停止，补齐完整 Tag 证据后从头重跑。没有门禁绕过、数据损坏、生产回滚或 108 操作。
