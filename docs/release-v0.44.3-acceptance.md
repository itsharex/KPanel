# KPanel v0.44.3 发布验收

## 发布范围

v0.44.3 上线 Docker 日常右键菜单，覆盖容器、镜像、网络和存储卷，并补齐容器暂停/恢复动作。登录控制台过渡优化提交
`fabf0948cd3660b0c786b05f05e51a2781407cc0` 已在 v0.44.2 主线中，本次继续纳入回归验证，不重复合入。

旧系统中心及相关实验分支没有进入本次发布；本次没有新增系统中心导航、页面或宿主管理协议。

## 版本与制品

- Docker 功能提交：`70f427b8b073719215d56ba2ab5bc0a4566c601f`。
- 发布提交：`f48b5ba1a16c5f68b97f6f28f5a0fea9eba8e47e`。
- 标签：`v0.44.3`。
- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30966591433>。
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30966743764>。
- Release 工作流：<https://github.com/kejilion/KPanel/actions/runs/30966928067>。
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.44.3>。
- Docker OCI index：`sha256:1c5a0ebdd40023d63a7ff484de7872e7589ee855d7c73521b2bb4fc76a714586`。
- linux/amd64：`sha256:c683f3a3a1ad25e12bfbd5808f9dd1b2f7f844671e86235ca5ecdca09229b8e8`。
- linux/arm64：`sha256:1bc33393738b9138850634b423cc7a47ae7d3a71326bed27ca7a36b9a568ad4a`。

Release 已公开且不是 prerelease，Agent、Node、部署归档、许可证、第三方声明和 `SHA256SUMS` 共 8 个附件完整。
`0.44.3` 与 `latest` 指向同一 OCI index；额外 `unknown/unknown` 清单为 amd64/arm64 的 SBOM/Provenance attestation。

## 自动化与 154 隔离验收

- 154 在精确发布提交执行完整 L3：前端 39 个文件共 256 项测试、1,622 条 i18n、typecheck、生产构建和
  `npm audit` 全部通过。
- 全量 Go 测试、特权核心包 race、`go vet ./...`、`govulncheck`、Trivy 源码与镜像扫描全部通过；源码和镜像
  HIGH/CRITICAL 结果为 0。
- 应用市场安装、更新、失败回滚和卸载生命周期输出 `app_conf_lifecycle=pass`；本地候选镜像与从 Docker Hub
  重新拉取的公开 `0.44.3` 镜像均输出 `image_e2e=pass`。
- 隔离候选实例使用临时 Panel 数据并只读连接 154 Agent。登录从 `/login` 正常进入 `/overview`；容器、镜像、
  网络、存储卷四类真实资源均能通过右键打开对应 `role=menu` 菜单，未点击任何资源动作，页面错误日志为 0。
- Docker API 新增动作仅为固定 `pause` / `unpause`，继续经过管理员会话、CSRF/Origin、固定 Agent 路径、
  resourceVersion 和审计保护，不接受任意 Shell。
- `packaging/kejilion-app/kpanel.conf` 相对 v0.44.2 无变化，且与 `kejilion/apps` 远端 `origin/main` 的 Git blob
  `0a603abfe77beb045c4e7648dd60f5e4a1876e4d` 完全一致，因此 apps 仓库无需新提交。

## 154 生产上线

- 升级前 Panel 为 `0.44.2/ok`，容器 healthy、重启 0、OOM=false，Agent active、重启 0；旧镜像摘要为
  `sha256:396c854af8b796924bb87db4c3dbf7adf9dc65bb5cf31bf8116a8cd5a3f3b3e8`。
- 一致性备份：`/root/kpanel-backups/v0.44.3-preupgrade-20260805T014530Z`。目录权限 0700、文件权限 0600；
  SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为
  `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整 KPanel 归档 20,750,896 B，
  SHA-256 为 `856309c575004f27a2478aad12331d4bca4ed684d3e9d43846225bfadb473d2a`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取正式摘要
  `sha256:1c5a0ebdd40023d63a7ff484de7872e7589ee855d7c73521b2bb4fc76a714586`，输出
  `KPanel 更新完成 / Update Complete`。
- 升级后 Panel 与 Agent 均为 `0.44.3`，本机和公网直连 `http://154.36.153.9:8080/api/v1/health` 返回
  `0.44.3/ok`；容器保持 `65532:65532`、只读根文件系统、非特权、`cap-drop ALL`、256 MiB 和 128 PID。
- 升级后 `ai.db integrity_check=ok`，`ai.db` 与 `ai-secrets.key` 权限保持 0600。
- 从升级完成起观察约 5 分钟，立即检查及后续 9 次采样全部为 healthy、Panel/Agent 重启 0、OOM=false；
  CPU 0.01%–0.05%，内存约 11.04–72.91 MiB，PID 7–8，Panel/Agent 的 panic、fatal、error 计数均为 0。

## 回滚与已知边界

- 公共镜像回滚点为 v0.44.2，摘要
  `sha256:396c854af8b796924bb87db4c3dbf7adf9dc65bb5cf31bf8116a8cd5a3f3b3e8`；现场回滚使用上述成对备份恢复
  KPanel 目录、SQLite、Compose、Agent 和旧镜像，不覆盖其他业务容器、网站、证书或数据库。
- `kp.kejilion.pro` 当前经 Cloudflare 指向的公网入口仍返回 `0.44.2`，且根路径与 `/docker` 返回 404；该入口
  不是本次已授权的 154 直连目标。本次未越权修改 DNS、Cloudflare 或其他源站，需单独确认其实际源站后再处理。
