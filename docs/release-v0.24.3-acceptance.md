# KPanel v0.24.3 验收记录

## 交付范围

- 应用详情改为紧凑纵向布局，1280×720 视口下完整显示状态、操作、域名、解析辅助、IP + 端口访问及维护区，无内部滚动。
- 停止应用仍可打开详情并执行启动、检查更新、更新和卸载；只有脚本标记存在但缺少容器的应用，才提供固定编号的交互式“脚本管理”恢复入口。
- 域名绑定、解绑、检查更新及直接访问切换遇到资源版本过期时，刷新真实状态并重试一次；后台应用任务运行期间禁用冲突操作。
- 应用打开链接优先使用已绑定域名，否则只使用服务器公网 IP + 应用端口，不再拼接 KPanel 反代域名与其他应用端口。
- 应用安装、更新、卸载和访问策略任务改用 `systemctl show ActiveState` 同步状态；`activating` 长任务及暂时无法查询的任务不再被提前标记为中断。
- 离线应用目录增加 KPanel 本地 128×128 WebP 图标。
- `kejilion.sh` 的 Docker Hub 更新检测改为摘要优先，并兼容 `docker.io/` 镜像前缀。

## 发布提交

- KPanel：`04ba26510e66c1ddca462113d741ba39fb0a170b`
- 标签：`v0.24.3`
- `kejilion.sh`：`4ee3f96591d4f6bd9a062fb574beee03107572fc`
- 镜像内脚本 SHA-256：`d470bb436a93036f238ffd25cbe0af8e0596d986f2c4c28f9a96ce66a9a874e5`

## 自动与本地验收

- 主分支 CI：<https://github.com/kejilion/KPanel/actions/runs/30371691746>
- Release：<https://github.com/kejilion/KPanel/actions/runs/30371915609>
- Linux Go 全量测试、`go vet`、`govulncheck v1.6.0`、linux/amd64 与 linux/arm64 Panel/Agent/kpctl 构建全部通过；未发现可达漏洞。
- Web 类型检查、66 项 Vitest、生产构建和 `npm audit --audit-level=high` 全部通过；npm 未发现漏洞。
- 安装安全测试、Shell 语法、生态规则、`kejilion.sh` 10 组 smoke tests 和 KPanel 应用安装/更新/回滚/卸载生命周期测试全部通过。
- 使用真实 systemd transient unit 复现：`systemctl is-active` 在 `activating` 时输出正确但退出码为 3；新实现 `systemctl show ActiveState` 返回 `activating` 且退出码为 0。
- 浏览器验证停止的 CLIProxyAPI 可重新打开详情；直接访问链接为 `http://159.54.161.187:11451`，没有拼接 KPanel 域名；1280×720 下详情主体 `clientHeight` 与 `scrollHeight` 均为 547。
- 最终镜像在非 root、只读根文件系统、`cap-drop ALL`、`no-new-privileges` 和无网络条件下通过健康检查；OCI 版本、源码提交、脚本提交及脚本摘要均通过检查。

## 线上产物

- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.24.3>
- 生产镜像：`docker.io/kjlion/kejilion-panel@sha256:ed2d2a792c95f8e1d3d9075db138d9eaa7c6a204ea238c8bb88953f98f0e93d7`
- `0.24.3` 与 `latest` 为同一 manifest digest，包含 linux/amd64、linux/arm64 及对应 SBOM/Provenance 证明清单。
- 从 Docker Hub 重新拉取的 `0.24.3` 镜像已独立通过运行时契约和内置脚本摘要检查。
- 发布附件已重新下载，并按附件中的 `SHA256SUMS` 独立校验：
  - `kejilion-agent-linux-amd64`：`d360f38b8ad241ca56ceaf8f7c4d5d5e421e8d468d550c9f304df454b6193c90`
  - `kejilion-agent-linux-arm64`：`39223b300b019a1acdd607118c78a07616a812bc783bf2050be1cc1678210b3a`
  - `kejilion-panel-deploy-0.24.3.tar.gz`：`beb9647b52c155bb8ad60ed5d199280d696dec5dc58c64311d832d25bbee600b`

## 回滚与边界

- 代码回滚点：标签 `v0.24.2`。
- 镜像回滚点：`docker.io/kjlion/kejilion-panel@sha256:060569d6e55277c0fdd7a85665a8e8bc1b7f28ef756df4529c5c3600db8cea1c`。
- 本版本没有数据格式迁移。回滚 KPanel/Agent 不删除 `/home/web`、应用容器、域名配置或环境备份。
- 本次发布完成 GitHub Release、Docker Hub 版本镜像和 `latest` 更新；没有自动替用户升级生产主机，用户从应用市场更新 KPanel 时才执行原位升级与失败回滚。
