# KPanel 外联配置来源登记

本文件是 [`PROJECT_RULES.md`](../PROJECT_RULES.md)“外联配置直接复用硬规则”的强制登记表。
“目录相同”“脚本可发现”或“配置语义相似”不等于合规。状态只能使用：

- **已合规**：直接调用脚本入口、消费脚本同一权威模板，或双方调用同一共享生成器，并有双端证据；
- **不合规/冻结**：KPanel 仍维护自编模板或独立流程，禁止继续扩展，必须先迁移；
- **待审计**：尚未完成脚本来源与实际写入链路核对，不得据此宣称完全对齐。

## 当前登记

| ID | 业务与 KPanel 入口 | `kejilion.sh` 权威来源 | 当前方式 | 状态与发布要求 |
| --- | --- | --- | --- | --- |
| `website-nginx` | 静态站、PHP、域名反代、负载均衡、跳转；`internal/sites/managed_template.go` | `k web`；`html.conf`、域名反代及负载均衡模板 | KPanel `renderManagedConfig()` 自行拼接 | **不合规/冻结**：禁止继续修改或发布新增能力；先改为脚本同源模板/入口 |
| `reverse-proxy-ip-port` | IP+端口反向代理；网站页热门入口 | `k fd <domain> <host> <port>`；`ldnmp_Proxy` 与 `reverse-proxy-backend.conf` | Go 后台 PTY 任务直接执行本机可信脚本命令，域名和固定上游参数由面板传入，其余提示可交互输入；完成后发现 `/home/web` 产物 | **已合规（代码链路）**：发布前仍需目标机实测创建、脚本管理、面板管理与删除 |
| `wordpress-flow` | WordPress；网站页热门入口 | `k wp <domain>`；`ldnmp_wp`、LDNMP、证书、数据库、`wordpress.com.conf` 和脚本源码地址 | Go 后台 PTY 任务直接执行本机可信脚本命令，KPanel 不再先进入 `k web` 菜单，也不维护第二套 WordPress 安装器 | **已合规（代码链路）**：发布前仍需目标机实测创建、脚本管理、面板管理与删除 |
| `website-recipes` | Discuz、KodBox、MacCMS、独角数卡、Flarum、Typecho、LinkStack、AI Prompt | `k discuz <domain>` 等固定直达命令 | 后台 PTY 直接执行脚本命令并读取 `KPANEL_PROGRESS`；窗口关闭后任务继续，页面可恢复终端 | **已合规（代码链路）**：发布前仍需按目标脚本版本做实机闭环 |
| `application-market` | 应用安装、更新、卸载、域名与访问控制 | `/root/apps/*.conf`、动态应用目录及脚本非交互协议 | 部分直接脚本任务，部分 KPanel 适配器 | **待审计**：逐应用登记入口与来源后才能宣称完全对齐 |
| `system-dns` | 概览页 DNS 设置 | `set_dns` 与 `kpanel_set_dns_noninteractive` | Go 仅校验结构化 IP 并调用本机可信 `kejilion.sh dns`；最终配置、后端选择和回滚由脚本负责 | **已合规（代码链路）**：发布前仍需在 systemd-resolved、静态文件和受网络管理器接管的主机完成双端实机闭环 |
| `diagnostic-scripts` | 体检页 IP、线路、性能与综合测试 | `linux_test`、`kpanel_test_catalog`、`kpanel_run_remote_bash` 与 `kpanel_run_test_noninteractive` | Agent 从可信脚本读取固定目录，以 PTY 执行 `KJ_TEST_NONINTERACTIVE=1 k test run <selector>`；固定来源先下载再执行，保留 stdin 与 ANSI 颜色 | **已合规（代码链路）**：目录、拒绝未知 selector、终端偏移、输入保护、后台日志和失败状态已自动验证；各第三方来源的完整实机跑分需在目标服务器按需验收 |
| `managed-script-runtime` | SSH 防御、DNS、应用、建站、体检与 LDNMP 环境管理共同使用的宿主机脚本入口 | `kejilion/sh@4ee3f96591d4f6bd9a062fb574beee03107572fc`；SHA-256 `d470bb436a93036f238ffd25cbe0af8e0596d986f2c4c28f9a96ce66a9a874e5` | 镜像构建时按提交和摘要下载到 `/release/kejilion.sh`；安装/更新后以 root:root、0700 保存到 `/home/docker/kpanel/bin/kejilion.sh`，只继承既有可信脚本已明确接受的许可及区域、统计设置 | **已合规（代码链路）**：安装、旧版升级、摘要拒绝和回滚由生命周期测试覆盖；发布镜像仍须复核 OCI 标签与镜像内摘要 |
| `ldnmp-environment` | 网站 → 环境管理 | `kejilion.sh` 的 `k web env` 固定协议；安装、Fail2Ban/WAF/Cloudflare/DDoS、优化模板、镜像更新、`/home/web_*.tar.gz` 备份与卸载语义 | Agent 只调用固定动作并读取真实产物；所有远程模板继续使用脚本内 `gh_proxy` 与原上游地址，KPanel 不增加替代下载源 | **已合规（代码链路）**：Shell/Go/前端测试覆盖协议、枚举、任务凭据、资源版本、备份路径与敏感输入；发布前必须完成 LDNMP 实机矩阵 |
| `system-network` | 软件源、V4/V6、内核、BBR、防火墙 | `kejilion.sh` 系统工具对应函数和远程配置 | 多个 Go 适配器独立执行 | **待审计**：凡脚本已有外联模板/远程来源的项目必须迁移为同源 |
| `docker-environment` | Docker 安装、换源、维护、迁移、备份与还原 | `kejilion.sh` Docker 工具函数及其远程来源 | KPanel 固定动作适配器 | **待审计**：逐动作核对，不得新增自编外联配置 |

<!-- external-config-debt:website-nginx:blocked -->

## KPanel 与 kejilion.sh 发布关系

- KPanel 与 `kejilion.sh` 按协议兼容，不要求版本号同步。
- KPanel 仅修改前端、Go 服务或不涉及脚本协议的功能时，继续固定上一份已验证脚本；
  不提交或发布新的 `kejilion.sh`。
- KPanel 新增或修改脚本协议时，先发布 `kejilion.sh`，再在 KPanel 固定脚本提交和
  SHA-256，并完成双端协议测试。
- `kejilion/apps/kpanel.conf` 不跟随普通 KPanel 或脚本版本发布；它从镜像 OCI 标签、
  `/release/VERSION` 和产物本身读取并交叉验证版本、源码提交、脚本提交及脚本摘要。
- 只有应用安装协议、镜像产物路径或 OCI 契约发生变化时，才修改应用市场配置。

### KPanel 应用市场自更新约束

- 用户入口固定为“应用市场 → KPanel → 更新”；底层增强不得另建第二套更新流程或改变原有确认、
  后台运行和任务日志体验。
- 更新必须把 Panel、Agent 与 KPanel 专用 `kejilion.sh` 作为同一事务校验和切换；宿主机不需要
  Go、Node.js 或其他编译环境，也不得覆盖系统的 `k` 命令。
- 更新前必须从正在运行的容器读取并校验当前镜像 ID 和宿主机端口；更新后继续保留原端口、
  `.env`、Panel 数据、应用、站点、域名和 `/home/web`。
- 新版本健康检查失败时，必须将 `latest` 本地标签恢复到更新前的精确镜像 ID，并同步恢复旧
  Agent、脚本、Compose、systemd unit 与 Agent 环境配置后重新验收。
- Panel 或 Agent 切换造成的短暂 API 不可用属于可重试状态；前端不得因此清除后台任务 ID。
  只有任务明确结束或服务确认返回任务不存在时，才结束任务跟踪。
- 发布门禁必须包含自定义端口保持、元数据/版本/脚本摘要拒绝、更新失败镜像回滚、任务重连和
  原安装/卸载体验不变的自动测试；正式发布仍需在真实 Docker 主机执行升级与回滚闭环。

## 每项迁移完成的证据

1. 脚本菜单/函数、模板 URL 或共享生成器的准确版本与 SHA-256；
2. KPanel 调用链证明没有内置第二份业务模板；
3. 相同输入下去除时间戳、随机凭证等易变字段后的有效配置对比；
4. 脚本创建 → KPanel 管理，以及 KPanel 创建 → 脚本管理的实机记录；
5. 更新、删除、失败回滚和脚本来源升级后的兼容测试。
