<p align="center">
  <img src=".github/assets/kpanel-logo.png" alt="KPanel" width="520">
</p>

<p align="center">
  与 <code>kejilion.sh</code> 双向互通的现代 Linux 服务器管理面板
</p>

<p align="center">
  <a href="https://github.com/kejilion/KPanel/releases/latest"><img src="https://img.shields.io/github/v/release/kejilion/KPanel?display_name=tag" alt="Latest release"></a>
  <a href="https://github.com/kejilion/KPanel/actions/workflows/ci.yml"><img src="https://github.com/kejilion/KPanel/actions/workflows/ci.yml/badge.svg" alt="CI status"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/kejilion/KPanel" alt="Go version"></a>
</p>

<p align="center">
  <a href="#功能概览">功能概览</a> ·
  <a href="#快速开始">快速开始</a> ·
  <a href="docs/deployment.md">部署文档</a> ·
  <a href="docs/session-collaboration.md">协作说明</a> ·
  <a href="CHANGELOG.md">更新记录</a>
</p>

KPanel 将 `kejilion.sh` 的服务器运维能力带到 Web 端。脚本、SSH、Compose
和 KPanel 创建的真实资源可以互相发现、修改并继续管理，避免因为换了管理入口
就失去对现有网站、容器和配置的控制。

KPanel 可以安装在仅具备 systemd、Docker Engine 与 Compose v2 的干净 Linux
主机上，不要求预先运行 `kejilion.sh` 或存在 `/home/web`。接入可信
`kejilion.sh` 后，还可以复用其网站和应用业务。

## 功能概览

- **主机总览与系统管理**：查看 CPU、内存、磁盘、负载、网络、服务状态，管理主机名、
  SSH 端口、DNS、时区、Swap、软件源、内核预设、BBR、系统更新与安全重启。
- **网站管理**：发现现有站点、证书与 Nginx 状态，创建和管理静态站、PHP 站、
  反向代理、负载均衡、域名重定向及常用建站程序。
- **Docker 管理**：统一管理容器、镜像、网络和卷，支持日志、性能采样、更新检查、
  备份还原、IPv6 及脚本兼容的访问控制。
- **应用市场**：动态对齐 `app.kejilion.sh`，展示真实安装和容器状态，并为已审计应用
  提供后台安装、更新、卸载与失败回滚。
- **服务器体检**：运行 IP/解锁、网络线路、硬件性能和综合测评，持续显示脚本来源、
  资源影响、实时日志和历史结果。
- **审计与恢复**：记录管理变更，检测资源版本冲突；配置写入前校验，失败时回滚并明确
  报告未完成的清理项。

## 为什么是 KPanel

- **与脚本互认**：宿主机实际状态是事实来源，面板数据库不是资源的唯一真相。
- **不接管现有环境**：安装器不会修改 `kejilion.sh`、`/home/web`、Nginx、防火墙
  或现有站点。
- **权限边界清晰**：Web/API 进程无特权运行，不挂载 Docker Socket 或宿主机根目录；
  宿主机操作由本地 Unix Socket 上的结构化 Agent 执行。
- **来源不限制管理**：脚本、KPanel、Compose 或人工创建的资源，都可以按实际状态
  继续管理。

## 快速开始

支持 `amd64` 和 `arm64`。正式部署目标为使用 systemd 的 Debian/Ubuntu、
RHEL/Fedora、Arch/Manjaro 与 openSUSE/SLES；详细验收范围见
[宿主机系统兼容矩阵](docs/platform-support.md)。

1. 从 [最新 Release](https://github.com/kejilion/KPanel/releases/latest) 下载部署包、
   与主机架构匹配的 `kejilion-agent` 和 `SHA256SUMS`。
2. 使用 `SHA256SUMS` 校验下载文件，并从 Release 说明复制固定的镜像 digest。
3. 解压部署包，先运行只读预检：

   ```sh
   ./deploy/preflight.sh \
     --public-url https://panel.example.com \
     --network-subnet 172.29.255.240/28
   ```

4. 按[完整部署文档](docs/deployment.md)先执行安装器的 `--dry-run`，确认检查结果后
   再正式安装并配置 HTTPS 反向代理。

> [!IMPORTANT]
> KPanel 的 Agent 具备宿主机管理能力，因此不提供跳过校验的 `curl | bash`
> 安装方式。生产部署应使用已校验的 Agent、固定镜像 digest 和 HTTPS。

> [!NOTE]
> 当前安装器仅支持全新安装；发现既有 KPanel 文件、同名容器或同名网络时会拒绝继续。
> Alpine/OpenRC 暂不属于正式部署目标。

## 设计与安全

- 不修改、覆盖或 `source` 现有 `kejilion.sh`；需要复用脚本业务时，通过版本化的
  非交互协议调用对应动作。
- 登录、服务端 Session、CSRF、登录限速、路径约束与供应链校验属于基础能力。
- 资源来源、KPanel label、脚本 marker、人工修改、危险运行参数或 KPanel 自身身份，
  均不构成管理授权条件。
- 配置写入具备校验、审计和失败回滚；不可逆任务会明确标注并持久化进度与结果。

完整原则见 [PROJECT_RULES.md](PROJECT_RULES.md)、[生态互通基线](docs/ecosystem-parity.md)
与[操作边界审计](docs/operational-boundary-audit.md)。

## 当前边界

通用宿主机终端、Compose/`daemon.json` 通用编辑器、系统重装非交互适配器，以及部分
发行版的 DNS/换源适配器仍在规划中。这些是待实现能力；后续实现仍需经过鉴权、
结构化输入、路径约束、并发控制、审计和失败恢复。

## 项目文档

- 入门与部署：[构建、发布与部署](docs/deployment.md)、
  [宿主机系统兼容矩阵](docs/platform-support.md)
- 架构与安全：[架构与事实来源](docs/architecture.md)、
  [攻击面与操作边界](docs/security-model.md)、
  [性能、稳定性、资源与网络入侵安全开发规范](docs/development-quality-standard.md)
- 生态兼容：[kejilion.sh 兼容基线](docs/compatibility.md)、
  [网站业务分析](docs/legacy-site-contract.md)、
  [应用市场对齐](docs/application-market.md)
- 功能设计：[体检与第三方测试协议](docs/diagnostics.md)、
  [Docker 管理与互通边界](docs/docker-management-v0.18.md)
- 协作规范：[Codex 会话协作](docs/session-collaboration.md)
- 版本信息：[版本变更记录](CHANGELOG.md)
