# KPanel v0.60.1 发布验收记录

日期：2026-08-10

发布级别：L3

## 发布范围

本版本只包含两项兼容性修复及版本冻结：

- `1e43542`：修复 AI 工具 Schema 对不同模型提供商的兼容性，避免把仅部分提供商支持的约束直接下发；
- `0c5c74e`：保持桌面终端与应用脚本窗口的输入区在紧凑窗口内可见；该提交与原始需求提交 `950e343` 的 patch-id 一致；
- `dcf3c3c`：统一版本字段并准备 KPanel `0.60.1`。

正式源码、标签和发布镜像对应的源码提交均为：

```text
dcf3c3c4a66b8b9a56876d4f0447b3da37f32320
```

历史提交 `2c13a8e` 的等价主线实现为 `a4e26ec`，二者 patch-id 均为 `ebca1a8e77ac24cf5e30d0d5ea3e3b41f3f201ad`。候选继承并保留 `a4e26ec`，未合入旧分支上的撤销提交，随后再合入 `950e343` 的等价修复，因此两项桌面窗口行为同时存在且没有重复应用。

本版本没有 API、数据库、端口、Compose、Agent 权限、脚本协议或应用市场安装契约迁移。

## 未上线内容审计

从 `main@e27e691` 审计全部本地工作树和近期分支后，真正未上线且仍有效的净内容只有上述两项修复。候选相对基线共变更 17 个文件，192 行新增、232 行删除。

未纳入范围：

- 旧体检救援工作树的 3 个未提交文件，已被后续正式实现覆盖；
- 旧概览服务状态工作树的 2 个等价文件；
- 桌面浏览器分支中的未跟踪预览 HTML；
- 其他已发布的补丁等价分支和历史实验分支。

上述旧工作树均未被覆盖、提交或发布。

## 发布前验证

- 版本字段一致、`git diff --check` 和工作流 YAML 解析通过；
- AI 与桌面窗口定向测试共 17 项通过；
- 前端 81 个测试文件、587 项测试通过；
- i18n 检查通过，共 1891 条文案和 19 个按页加载语言包；
- TypeScript、生产构建和紧凑桌面窗口视觉验收通过；
- Linux L3 `make verify-release` 通过，覆盖 Go 测试、race、静态检查、依赖审计、漏洞扫描、双架构构建及最终镜像扫描；
- 应用配置生命周期输出 `app_conf_lifecycle=pass`；
- 公开版本镜像端到端检查输出 `image_e2e=pass`；
- GitHub Release 中 5 个二进制/部署归档均通过公开 `SHA256SUMS` 校验。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31394148862>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31394566447>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31394873451>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.60.1>

Release 为非 draft、非 prerelease，共 8 个附件。Docker 镜像索引如下：

```text
0.60.1 / latest:
sha256:c344b3519a07496fe1466be832845922f2e164c800a2326100912b7453f4e7c4

linux/amd64:
sha256:5505669c6ec03dbcae260d90d733bba4c26de777b2701a880d0fcd1c9d9153db

linux/arm64:
sha256:319e6e2edbfe536a02acfccf35c5943f9c2d33b2c5c020f0c3e886f3de905d70
```

`packaging/kejilion-app/kpanel.conf` 相对发布基线未变，本地、L3 和基线 blob 均为 `d49383667cea8c3b7294bf40ba1e272370a2cb87`。`kejilion/apps` 主线保持 `1f2740666a55ccbb3749ce83168e073c1ea08431`，`kejilion/sh` 主线保持 `3972217d4d4a51d473b7375f5b850870e066be92`，均无需额外提交。

## 生产部署与验收

部署前生产版本为 `0.60.0`，Panel healthy、Agent active。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

停写一致性备份：

```text
/root/kpanel-backups/v0.60.1-preupgrade-20260810T135900Z
```

归档校验：

```text
6fe4bc5df4840bc2454f2706b25bfdcc2f840f5acde0efbe7336452a65156d3c  kpanel.tar.gz
```

上线后验证：

- `/api/v1/health` 返回 `status=ok`、`version=0.60.1`；
- Panel 为 running、healthy，镜像索引为 `sha256:c344b351...e7c4`，OCI 源码提交为 `dcf3c3c`；
- Agent 为 active、0 重启、`NeedDaemonReload=no`，版本为 `0.60.1 v1alpha1`，宿主机二进制与正式镜像内 Agent 的 SHA-256 均为 `b46633a076290a9cb7052ecfcae59bc99aabab0517b92911e768239f7814d524`；
- 受管 `kejilion.sh` 仅继承 `permission_granted=true` 安装标记，其他内容与镜像内固定脚本一致；
- Panel 状态 JSON 可解析，SQLite 数据库 `PRAGMA integrity_check` 返回 `ok`；
- Agent Unix Socket 上的健康、系统摘要、公网网络、进程和监控历史 5 个只读接口均通过；
- 更新后 Panel 日志无 `panic`、`fatal` 或段错误，Agent 保持正常运行；
- 约 2 分钟内，本机与配置的权威外部入口各 60 次健康采样全部成功，版本始终为 `0.60.1`。

## 回滚点

源码回滚：`v0.60.0`（`6d6028a90c2912bc4c7c3f5d53b44e95687fb65b`）。

镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:09b0ed3be95b2db4c4a2dfa4aceb31fd369f6ea473e38e06d713d9ec8d6174af
```

本版本没有数据或配置迁移。普通回滚可将 Compose 镜像固定到上述摘要后重建 Panel，并从该镜像恢复 Agent 和受管脚本；仅当数据或配置也需要回退时，才停止 Panel/Agent、校验备份 SHA-256，并恢复停写归档。

## 遗留风险

- 未使用生产管理员凭据执行浏览器内逐按钮验收，避免读取或传输生产凭据；桌面窗口行为由定向组件测试、全量测试、本地视觉验收、L3、公开镜像 E2E 和生产健康采样覆盖；
- AI Schema 修复已通过单元测试、全量 Go 门禁和生产健康检查，但未使用真实第三方模型密钥逐提供商发起付费请求。
