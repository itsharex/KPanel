# KPanel v0.60.2 发布验收记录

日期：2026-08-10

发布级别：L3

## 发布范围

本版本是在 `v0.60.1` 候选冻结后新增的独立体检页补丁：

- `c73cddf`：已成功或失败完成的体检项目显示“已测”标记，并移除重复的最近体检区块，让工作区集中展示当前项目和终端输出；
- `ec73b71`：修复紧凑桌面窗口中体检交互终端挤压脚本输入区的问题；
- `e1501e0`：统一版本字段并准备 KPanel `0.60.2`。

正式源码、标签和发布镜像对应提交：

```text
e1501e00747d9b7f89edb2cdac943f12f22f3178
```

两项原始本地提交 `058728e`、`f7d86e6` 均产生于 `v0.60.1` 候选冻结之后，已按相同 patch 独立迁移到最新远端主线；没有带入管理工作区的分叉历史。本版本没有 API、数据库、端口、Compose、Agent 权限、脚本协议或应用市场安装契约迁移。

## 发布前验证

- 变更边界共 11 个文件，版本字段一致，`git diff --check` 和工作流 YAML 解析通过；
- 定向 3 个测试文件、18 项测试通过；
- 前端 81 个测试文件、587 项测试通过；
- i18n 检查通过，共 1889 条文案和 19 个按页加载语言包；
- TypeScript 和生产构建通过；
- Linux L3 `make verify-release` 通过，覆盖 Go 测试、race、静态检查、依赖审计、漏洞扫描、双架构构建及最终镜像扫描；
- 应用配置生命周期输出 `app_conf_lifecycle=pass`，配置 blob 仍为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`；
- 公开版本镜像端到端检查输出 `image_e2e=pass`；
- GitHub Release 中 5 个二进制/部署归档均通过公开 `SHA256SUMS` 校验。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31397949963>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31398350730>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31398725204>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.60.2>

Release 为非 draft、非 prerelease，共 8 个附件。Docker 镜像索引如下：

```text
0.60.2 / latest:
sha256:508d5eefdb43e05b8e8a42b18d6758eb99387116de978e6b2da82c8a782bf982

linux/amd64:
sha256:59c9f499c315250e337b454be4c79cc82bbeb5b5a308bf20baefb7c8c2932116

linux/arm64:
sha256:604b1bfc4d9c5b7e008fab52ab71df8ffede37cebbabdf296d5e1aa7216bb3a4
```

`kejilion/apps` 和 `kejilion/sh` 均无需额外提交。

## 生产部署与验收

部署前生产版本为 `0.60.1`，Panel healthy、Agent active。通过标准应用市场更新入口部署。

停写一致性备份：

```text
/root/kpanel-backups/v0.60.2-preupgrade-20260810T143900Z
```

归档校验：

```text
3739b0e7d121f9a9e4dbd8c8ebaa477bdde311fe6565f3f35416af82a9f3dcd4  kpanel.tar.gz
```

上线后验证：

- 健康接口返回 `status=ok`、`version=0.60.2`；
- Panel 为 running、healthy，镜像索引为 `sha256:508d5eef...bf982`，OCI 源码提交为 `e1501e0`；
- Agent 为 active、0 重启、`NeedDaemonReload=no`，版本为 `0.60.2 v1alpha1`，宿主机与正式镜像内 Agent 的 SHA-256 均为 `9efea7939c5a00f80550a3f1c4f3bbab213fb6b9815c286cbbe75c27a7a2ee4e`；
- 受管 `kejilion.sh` 仅继承 `permission_granted=true` 安装标记，其他内容与镜像内固定脚本一致；
- Panel 状态 JSON 可解析，SQLite 数据库完整性检查返回 `ok`；
- Agent Unix Socket 上的健康、系统摘要、公网网络、进程和监控历史 5 个只读接口均通过；
- 更新后日志无 `panic`、`fatal` 或段错误；
- 约 2 分钟内，本机与配置的权威外部入口各 60 次健康采样全部成功，版本始终为 `0.60.2`。

## 回滚点

源码回滚：`v0.60.1`（`dcf3c3c4a66b8b9a56876d4f0447b3da37f32320`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:c344b3519a07496fe1466be832845922f2e164c800a2326100912b7453f4e7c4
```

本版本没有数据或配置迁移。普通回滚可固定上述镜像后重建 Panel，并从该镜像恢复 Agent 和受管脚本；仅当数据或配置也需要回退时，才停止 Panel/Agent、校验备份并恢复停写归档。

## 遗留风险

- 未使用生产管理员凭据执行浏览器内逐按钮验收，避免读取或传输生产凭据；体检页行为由定向测试、全量测试、L3、公开镜像 E2E 和生产持续健康采样覆盖；
- 未在生产主动启动会消耗流量或 CPU 的第三方体检任务，因此“已测”标记与紧凑输入区未通过生产写操作复验。
