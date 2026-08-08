# KPanel v0.51.0 发布验收

## 发布范围

本版本只上线桌面模式的四项变更：

1. 已安装应用图标可直接打开应用详情。
2. 网站图标支持正确显示、重命名后的缓存刷新与失败回退。
3. 桌面可启动由脚本管理的应用。
4. 页面重载时不再短暂闪现经典模式外壳。

本版本不包含系统中心、治理文档、数据库结构、端口、Compose、Agent 权限、
`kejilion.sh` 或应用市场配置变更。

## 版本与产物

- 发布提交：`c7e074f22b10991448a921a51213599e0114ba96`。
- 标签：`v0.51.0`。
- 候选分支 CI：[31260756497](https://github.com/kejilion/KPanel/actions/runs/31260756497)，结论 `success`。
- 主分支 CI：[31260898225](https://github.com/kejilion/KPanel/actions/runs/31260898225)，结论 `success`。
- Release 工作流：[31261046144](https://github.com/kejilion/KPanel/actions/runs/31261046144)，结论 `success`。
- GitHub Release：[v0.51.0](https://github.com/kejilion/KPanel/releases/tag/v0.51.0)，已公开、非 prerelease，共 8 个发布资产。
- Docker OCI index：`sha256:9006758325cb941eeaed90f668a48bf03cf4ac6da288dd8723face887986aba4`；`0.51.0` 与 `latest` 一致。
- `linux/amd64`：`sha256:e7b1a82d7d4b667e5be134b2b0163212214b29dffdc3b1c2a3fc19a5d39dbf09`。
- `linux/arm64`：`sha256:ffb0d1fe6f9085476c0182016d5f68b7c24527279eff849a86704c6ccf8face6`。
- `kejilion/apps` 无契约变化；本地与远端 `main` 均为 `1f2740666a55ccbb3749ce83168e073c1ea08431`。
- `kejilion/sh` 无变更；本地与远端 `main` 均为 `dd26cf7eb962f985f94773c15f9b643677b4471c`。

## 上线前验证

- 干净安装 252 个前端依赖，npm 审计为 0 个漏洞。
- 前端 68 个测试文件、456 项测试通过；typecheck、1,660 条国际化短语校验和生产构建通过。
- 主入口 JavaScript 为 23.68 KiB gzip，Desktop 为 11.90 KiB，Apps 为 12.92 KiB，均在预算内。
- 候选分支、主分支和 Release 三层 GitHub Actions 门禁全部成功。
- 154 从 Docker Hub 按正式标签拉取公开镜像，在隔离端口 `18092` 完成冷启动 E2E，退出码为 0，输出 `image_e2e=pass`。

## 154 上线结果

- 升级前 Panel 为 `0.50.0`、容器健康且重启次数为 0，镜像摘要为
  `sha256:f54bd72c6d01465de294326329fc131e30c1c04603d9e12f97e5b4763560904c`。
- 升级前备份：`/root/kpanel-backups/v0.51.0-preupgrade-20260808T140943Z`，目录权限 0700。
  SQLite 在线备份 122,880 B，`integrity_check=ok`，SHA-256 为
  `5b05d3bfb21d628de4ab2fcba415961099faaf686358dd969705e8db014e3220`；完整应用归档
  21,550,999 B，SHA-256 为 `512e29c4e3bcec55f137265424d9c89e357e67da7542b3dd39a71f0cdc42cd12`。
- 使用 `KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update k app kpanel` 完成标准应用市场更新，实际拉取摘要与 Release OCI index 一致。
- 升级后健康接口连续 3 次返回 `status=ok`、`version=0.51.0`；容器为 healthy、重启 0、
  OOM=false，OCI `revision=c7e074f22b10991448a921a51213599e0114ba96`。
- Panel 继续使用 `65532:65532`、只读根文件系统、256 MiB、1 CPU、128 PID、
  `cap-drop ALL` 和 `no-new-privileges`。
- Agent 为 `0.51.0 v1alpha1`、active/enabled、重启 0、`NeedDaemonReload=no`；宿主机 Agent 与
  镜像内置二进制 SHA-256 均为 `32039a0ab83258d754c255c374515840646a4594dec2d3c88c882dc1030ce0c0`。
- 生产 SQLite 再次检查为 `integrity_check=ok`；Panel 与 Agent 最近 10 分钟
  `panic/fatal/error` 计数均为 0。
- `http://154.36.153.9:8080/` 返回 200，HTML 标题为 `KPanel · Linux 服务器管理面板`；
  公网健康接口返回 200、`version=0.51.0`。
- 浏览器自动化环境本次无法完成裸 IP 页面导航，因此没有把登录后的桌面交互标记为生产视觉验收通过；
  相关交互由候选分支布局、组件与全量回归测试以及上线前隔离镜像 E2E 覆盖。
- `/usr/local/bin/k` 修改时间仍为 2026-08-05，本次应用更新未改动宿主机脚本入口。

## 回滚

- 源码与标签回滚点：`v0.50.0` / `13da6b53b6efd79cb2e09b0a531b80131d61d745`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:f54bd72c6d01465de294326329fc131e30c1c04603d9e12f97e5b4763560904c`。
- 现场回滚使用上述旧镜像及升级前备份恢复 `/home/docker/kpanel`，再重启 Compose 与 Agent。
  本版本没有数据库结构迁移，可直接回滚并保留现有数据格式。
