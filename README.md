# KPanel

> 项目长期原则：KPanel 与 `kejilion.sh` 属于同一生态，脚本业务是首要真源，双端产物必须互认并可继续管理。
> 资源来源或操作风险不得成为禁用管理员功能的理由；登录、抗注入、路径完整性、供应链和回滚防护必须保留。
> 详见 [PROJECT_RULES.md](PROJECT_RULES.md)、[生态互通基线](docs/ecosystem-parity.md) 与
> [操作边界审计](docs/operational-boundary-audit.md)。

KPanel 是 `kejilion.sh` 的现代 Web 管理形态，目标是让脚本已有业务在 Web 端等价执行，
并让脚本、SSH、Compose 与 Web 产生的真实资源可以互相发现、修改和继续管理。

KPanel 本体可在仅具备 systemd、Docker Engine 与 Compose v2 的干净 Linux 主机上独立安装；
不要求预先运行 `kejilion.sh` 或存在 `/home/web`。未初始化网站环境时只禁用网站相关能力。

## 核心原则

- 不修改、覆盖或 `source` 现有 `kejilion.sh`；需要复用脚本业务时，通过版本化的非交互协议调用对应动作。
- 宿主机实际状态是事实来源，面板数据库不是系统资源的唯一真相。
- Web/API 进程无特权运行，不挂载 Docker Socket 或宿主机根目录。
- 所有宿主机操作通过本地 Unix Socket 连接到结构化 Agent。
- 资源来源、KPanel label、脚本 marker、人工修改、危险运行参数或 KPanel 自身身份不构成管理授权条件。
- 配置写入具备校验、审计和失败回滚；不可逆的软件包维护任务明确标注并持久化进度与结果。

## 首版范围

- 首次初始化、安全登录、服务端 Session、CSRF、登录限速。
- 与 `k info` 对齐的 CPU、内存、磁盘、1/5/15 分钟负载、连接数、累计流量、
  公网地址、ISP、位置、宿主机时间和服务状态。
- `/home/web` 现有站点、证书和 Nginx 状态发现。
- 与 `kejilion.sh` 产物布局一致的静态站、PHP 站、IP/端口反代、域名反代、
  负载均衡和域名重定向创建与更新；Web 使用直观卡片选择服务。
- 脚本、Panel、人工修改及孤立网站产物均可按真实资源 ID 管理和删除；完整删除处理
  实际站点目录、域名证书与同名数据库，Nginx 变更失败时回滚，数据库清理失败时明确告警。
- WordPress 一键成品站：对齐脚本源码、目录、数据库、Redis、TLS 与 Nginx
  产物，使用在线 ACME 验证、供应链校验、冲突拒绝和失败回滚保护现有网站。
- Discuz、Kodbox、MacCMS、独角数卡、Flarum、Typecho、LinkStack 和 AI Prompt
  通过 `kejilion.sh` 固定非交互协议在后台一键搭建，直接复用脚本业务分支和真实产物。
- Docker 环境、容器、镜像、网络、卷五分区管理；全部容器均按实时状态支持启动、
  停止、重启、强制删除、有界日志、性能采样、单次控制台与脚本兼容访问控制。
- 结构化创建容器、镜像拉取/更新/删除、网络成员关系、local 卷、分类/完整清理、
  脚本同源镜像组、Docker IPv6，以及 `/home/docker` 后台备份、覆盖式事务还原和
  SSH 密钥迁移。
- 与 `app.kejilion.sh` 动态对齐的应用目录、本地图标、脚本安装状态、容器状态、镜像更新检查、
  域名绑定/解绑和访问策略；已审计的标准应用支持持久化后台安装与实时进度，声明式应用额外支持
  更新、卸载与失败回滚。
- 主机名、SSH 新端口、DNS、时区、与 `kejilion.sh` 共用的 `/swapfile`、APT 镜像、IP 优先级、
  五种内存自适应内核预设、BBR 和确认式服务器重启的类型化管理。
- Debian/Ubuntu、RHEL/Fedora、Arch/Manjaro 和 openSUSE/SLES 的系统更新，
  以及不触碰 Docker、网站和备份的系统清理。
- 管理变更记录、审计、资源版本冲突检测，以及依赖不可用时的明确状态。

当前尚缺少宿主机终端、交互式 TTY、Compose/daemon.json 通用编辑器、系统重装非交互
适配器和部分发行版 DNS/换源适配器。这些是待实现能力，不是以“高风险”为由永久关闭的产品限制；
实现时仍须经过鉴权、结构化输入、路径约束、并发控制、审计和失败恢复。

## 文档

- [架构与事实来源](docs/architecture.md)
- [v0.1 范围与验收](docs/scope-v0.1.md)
- [攻击面与操作边界](docs/security-model.md)
- [操作护栏审计与适配缺口](docs/operational-boundary-audit.md)
- [kejilion.sh 兼容基线](docs/compatibility.md)
- [kejilion.sh 网站业务分析](docs/legacy-site-contract.md)
- [应用市场对齐](docs/application-market.md)
- [v0.17 业务对齐与加载策略](docs/business-alignment-v0.17.md)
- [v0.18 Docker 五分区管理与互通边界](docs/docker-management-v0.18.md)
- [构建、发布与部署](docs/deployment.md)
- [宿主机系统兼容矩阵](docs/platform-support.md)
- [版本变更记录](CHANGELOG.md)

详细设计见 [架构说明](docs/architecture.md)、[兼容基线](docs/compatibility.md)、[安全模型](docs/security-model.md) 和 [v0.1 范围](docs/scope-v0.1.md)。
