# KPanel v0.24.4 验收记录

## 交付范围

- 对使用 `docker_app`、`docker_app_plus` 或单端口声明式适配器的应用，将安装端口输入前置到
  KPanel 安装弹窗；专属、多端口或无法安全识别端口的安装器继续使用原生交互终端。
- 安装前同时检查 Docker 已发布端口和宿主机 TCP/UDP 监听端口，并在任务提交和声明式容器创建
  前再次校验；冲突时不启动安装任务。
- KPanel 通过 `KJ_APP_PORT` 把已校验端口传给 `kejilion.sh`，脚本在实际安装前继续复核端口；
  SSH 直接执行脚本时仍保留原端口输入顺序。
- 应用市场中的运行中交互任务增加“结束任务”入口；只优雅停止可安全中断的交互任务，普通
  声明式安装、更新任务不开放强制终止。
- `kejilion/apps` 的 KPanel 配置同步一次性更新回滚保护；后续常规 KPanel 发版仍使用
  `latest` 镜像和镜像内发布契约，不需要逐版本修改应用配置。

## 发布提交

- KPanel：`b7bf3f093549769b69729b8f08092edb1b16e0c9`
- 标签：`v0.24.4`
- `kejilion.sh`：`229355b1b6e1ce405cc97019b3755e36f2b814c1`
- `kejilion/apps`：`e20a79ccb8982346fa79debe172d56aa3e38403f`
- 镜像内脚本 SHA-256：`12736ab5a0c331e4752d1bfd444c49500f2736cc5c19c8595be7ad9c2d37aef0`

## 自动与本地验收

- 主分支 CI：<https://github.com/kejilion/KPanel/actions/runs/30379871504>
- Release：<https://github.com/kejilion/KPanel/actions/runs/30380389858>
- Linux Go 全量测试、`go vet`、`govulncheck v1.6.0`、linux/amd64 与 linux/arm64
  Panel/Agent/kpctl 构建全部通过；未发现可达漏洞。
- Web 类型检查、70 项 Vitest、生产构建和 `npm audit --audit-level=high` 全部通过；
  npm 未发现漏洞。
- 安装安全测试、应用安装/更新/回滚/卸载生命周期测试和 `kejilion.sh` 应用协议 smoke test
  全部通过。
- 浏览器实际验证默认端口可用时显示绿色状态并允许提交；模拟 8080 被占用时显示明确冲突并禁用
  安装按钮；临时端口检查失败后仍可由提交动作重试。
- 本地发布镜像在非 root、只读根文件系统、`cap-drop ALL`、`no-new-privileges` 条件下通过
  健康检查；OCI 版本、源码提交、脚本提交及脚本摘要均通过检查。

## 线上产物

- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.24.4>
- 生产镜像：
  `docker.io/kjlion/kejilion-panel@sha256:4de5e75c3fc43331de7a4f4aa8907c960a271cf38a62282bd5569ad9e5aaf395`
- `0.24.4` 与 `latest` 为同一 manifest digest，包含 linux/amd64、linux/arm64 及对应
  SBOM/Provenance 证明清单。
- 从 Docker Hub 重新拉取的 `0.24.4` 镜像已通过运行时 E2E；Agent 自报
  `0.24.4 v1alpha1`，镜像内脚本摘要与登记值一致。
- 发布附件已重新下载，并按附件中的 `SHA256SUMS` 独立校验：
  - `kejilion-agent-linux-amd64`：`3d02686b19adaed3603f6a2a31045e6c2a88a63c7888ac73b75bbf64aa196b1c`
  - `kejilion-agent-linux-arm64`：`ae88f3dce1fec9fe2ede0756c74833f0cc47151fac3d92fcc034f4062c56345e`
  - `kejilion-panel-deploy-0.24.4.tar.gz`：`aff0ff599ef860cc6aec74a0b8db3b39e17d7b266fedf685b3cc49c75ae5aeb3`

## 回滚与边界

- 代码回滚点：标签 `v0.24.3`。
- 镜像回滚点：
  `docker.io/kjlion/kejilion-panel@sha256:ed2d2a792c95f8e1d3d9075db138d9eaa7c6a204ea238c8bb88953f98f0e93d7`。
- 本版本没有数据格式迁移。回滚 KPanel/Agent 不删除应用容器、网站、数据库、域名配置或环境备份。
- 发布不会自动替用户升级生产主机；用户从 `kejilion.sh` 应用市场更新 KPanel 时执行原位升级，
  保留当前访问端口，并在失败时恢复原镜像、Agent、脚本和服务配置。
