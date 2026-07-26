# KPanel

KPanel 是 `kejilion.sh` 的现代 Web 管理形态。首版聚焦安全登录、主机监控、现有网站发现、Docker 查看与受控生命周期管理。

## 核心原则

- 不修改、覆盖、执行或 `source` 现有 `kejilion.sh`。
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
- WordPress 一键成品站：对齐脚本源码、目录、数据库、Redis、TLS 与 Nginx
  产物，使用在线 ACME 验证、供应链校验、冲突拒绝和失败回滚保护现有网站。
- Docker 容器、镜像、网络、卷查看。
- 已识别 Kejilion 容器的启动、停止、重启和有界日志。
- 与 `app.kejilion.sh` 动态对齐的应用目录、本地图标、脚本安装状态、容器状态、镜像更新检查、
  域名绑定/解绑和访问策略；当前官方目录为 146 个应用，已审计应用支持声明式安装、更新、
  卸载与失败回滚。
- 主机名、SSH 新端口、DNS、时区、与 `kejilion.sh` 共用的 `/swapfile`、APT 镜像、IP 优先级、
  五种内存自适应内核预设、BBR 和确认式服务器重启的类型化管理。
- Debian/Ubuntu、RHEL/Fedora、Arch/Manjaro 和 openSUSE/SLES 的系统更新，
  以及不触碰 Docker、网站和备份的安全系统清理。
- 管理变更记录、审计、资源版本冲突检测和只读降级。

任意 Shell、Docker Exec、系统重装、磁盘格式化、应用市场远程脚本、第三方 Compose 和数据目录硬删除
等高风险能力不进入首版。

## 文档

- [架构与事实来源](docs/architecture.md)
- [v0.1 范围与验收](docs/scope-v0.1.md)
- [安全边界](docs/security-model.md)
- [kejilion.sh 兼容基线](docs/compatibility.md)
- [kejilion.sh 网站业务分析](docs/legacy-site-contract.md)
- [应用市场对齐与安全边界](docs/application-market.md)
- [构建、发布与部署](docs/deployment.md)
- [宿主机系统兼容矩阵](docs/platform-support.md)
- [版本变更记录](CHANGELOG.md)

详细设计见 [架构说明](docs/architecture.md)、[兼容基线](docs/compatibility.md)、[安全模型](docs/security-model.md) 和 [v0.1 范围](docs/scope-v0.1.md)。
