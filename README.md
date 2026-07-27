# KPanel

KPanel 是 `kejilion.sh` 的现代 Web 管理形态。首版聚焦安全登录、主机监控、现有网站发现、Docker 查看与受控生命周期管理。

KPanel 本体可在仅具备 systemd、Docker Engine 与 Compose v2 的干净 Linux 主机上独立安装；
不要求预先运行 `kejilion.sh` 或存在 `/home/web`。未初始化网站环境时只禁用网站相关能力。

## 核心原则

- 不修改、覆盖或 `source` 现有 `kejilion.sh`；只有固定白名单动作可通过已更新脚本的
  KPanel 非交互协议执行。
- 宿主机实际状态是事实来源，面板数据库不是系统资源的唯一真相。
- Web/API 进程无特权运行，不挂载 Docker Socket 或宿主机根目录。
- 所有宿主机操作通过本地 Unix Socket 连接到白名单式 Agent。
- 无法证明安全的资源保持只读。
- 配置写入具备校验、审计和失败回滚；不可逆的软件包维护任务明确标注并持久化进度与结果。

## 首版范围

- 首次初始化、安全登录、服务端 Session、CSRF、登录限速。
- 与 `k info` 对齐的 CPU、内存、磁盘、1/5/15 分钟负载、连接数、累计流量、
  公网地址、ISP、位置、宿主机时间和服务状态。
- `/home/web` 现有站点、证书和 Nginx 状态发现。
- 与 `kejilion.sh` 产物布局一致的静态站、PHP 站、IP/端口反代、域名反代、
  负载均衡和域名重定向创建与安全更新；Web 使用直观卡片选择服务。
- 对结构可确认且未漂移的脚本/Panel 站点提供配置解绑和确认式完整删除；完整删除按
  `k web` 布局处理站点目录、域名证书与同名数据库，并在 Nginx 校验失败时回滚。
- WordPress 一键成品站：对齐脚本源码、目录、数据库、Redis、TLS 与 Nginx
  产物，使用在线 ACME 验证、供应链校验、冲突拒绝和失败回滚保护现有网站。
- Discuz、Kodbox、MacCMS、独角数卡、Flarum、Typecho、LinkStack 和 AI Prompt
  通过 `kejilion.sh` 固定非交互协议在后台一键搭建，直接复用脚本业务分支和真实产物。
- Docker 环境、容器、镜像、网络、卷五分区管理；已识别 Kejilion 容器支持启动、
  停止、重启、安全删除、有界日志、性能采样、受审计单次控制台与脚本兼容访问控制。
- 结构化创建容器、镜像拉取/更新/删除、网络成员关系、local 卷、分类/完整清理、
  脚本同源镜像组、Docker IPv6，以及 `/home/docker` 后台备份、冲突保护还原和
  SSH 密钥迁移。
- 与 `app.kejilion.sh` 动态对齐的应用目录、本地图标、脚本安装状态、容器状态、镜像更新检查、
  域名绑定/解绑和访问策略；已审计的标准应用支持持久化后台安装与实时进度，声明式应用额外支持
  安全更新、卸载与失败回滚。
- 主机名、SSH 新端口、DNS、时区、与 `kejilion.sh` 共用的 `/swapfile`、APT 镜像、IP 优先级、
  五种内存自适应内核预设、BBR 和确认式服务器重启的类型化管理。
- Debian/Ubuntu、RHEL/Fedora、Arch/Manjaro 和 openSUSE/SLES 的系统更新，
  以及不触碰 Docker、网站和备份的安全系统清理。
- 管理变更记录、审计、资源版本冲突检测和只读降级。

宿主机 Shell、外部/危险容器 Exec、交互式 TTY、任意 Compose/daemon.json 文本、
系统重装、磁盘格式化、密码式远程命令，以及无法确认归属和布局的数据目录硬删除等
高风险能力不开放。

## 文档

- [架构与事实来源](docs/architecture.md)
- [v0.1 范围与验收](docs/scope-v0.1.md)
- [安全边界](docs/security-model.md)
- [kejilion.sh 兼容基线](docs/compatibility.md)
- [kejilion.sh 网站业务分析](docs/legacy-site-contract.md)
- [应用市场对齐与安全边界](docs/application-market.md)
- [v0.17 业务对齐与加载策略](docs/business-alignment-v0.17.md)
- [v0.18 Docker 五分区管理与互通边界](docs/docker-management-v0.18.md)
- [构建、发布与部署](docs/deployment.md)
- [宿主机系统兼容矩阵](docs/platform-support.md)
- [版本变更记录](CHANGELOG.md)

详细设计见 [架构说明](docs/architecture.md)、[兼容基线](docs/compatibility.md)、[安全模型](docs/security-model.md) 和 [v0.1 范围](docs/scope-v0.1.md)。
