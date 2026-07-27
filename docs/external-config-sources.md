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
| `reverse-proxy-ip-port` | IP+端口反向代理；网站页热门入口 | `KJ_WEB_RECIPE=23`；`ldnmp_Proxy` 与 `reverse-proxy-backend.conf` | Go 后台任务执行本机可信 `kejilion.sh web` 固定非交互分支，完成后发现 `/home/web` 产物 | **已合规（代码链路）**：发布前仍需目标机实测创建、脚本管理、面板管理与删除 |
| `wordpress-flow` | WordPress；网站页热门入口 | `KJ_WEB_RECIPE=2`；`ldnmp_wp`、LDNMP、证书、数据库、`wordpress.com.conf` 和脚本源码地址 | Go 后台任务执行本机可信 `kejilion.sh web` 固定非交互分支，KPanel 不再维护第二套 WordPress 安装器 | **已合规（代码链路）**：发布前仍需目标机实测创建、脚本管理、面板管理与删除 |
| `website-recipes` | Discuz、KodBox、MacCMS、独角数卡、Flarum、Typecho、LinkStack、AI Prompt | 本机可信 `kejilion.sh web` 非交互协议 | 直接执行脚本并读取 `KPANEL_PROGRESS` | **已合规（代码链路）**：发布前仍需按目标脚本版本做实机闭环 |
| `application-market` | 应用安装、更新、卸载、域名与访问控制 | `/root/apps/*.conf`、动态应用目录及脚本非交互协议 | 部分直接脚本任务，部分 KPanel 适配器 | **待审计**：逐应用登记入口与来源后才能宣称完全对齐 |
| `system-network` | DNS、软件源、V4/V6、内核、BBR、防火墙 | `kejilion.sh` 系统工具对应函数和远程配置 | 多个 Go 适配器独立执行 | **待审计**：凡脚本已有外联模板/远程来源的项目必须迁移为同源 |
| `docker-environment` | Docker 安装、换源、维护、迁移、备份与还原 | `kejilion.sh` Docker 工具函数及其远程来源 | KPanel 固定动作适配器 | **待审计**：逐动作核对，不得新增自编外联配置 |

<!-- external-config-debt:website-nginx:blocked -->

## 每项迁移完成的证据

1. 脚本菜单/函数、模板 URL 或共享生成器的准确版本与 SHA-256；
2. KPanel 调用链证明没有内置第二份业务模板；
3. 相同输入下去除时间戳、随机凭证等易变字段后的有效配置对比；
4. 脚本创建 → KPanel 管理，以及 KPanel 创建 → 脚本管理的实机记录；
5. 更新、删除、失败回滚和脚本来源升级后的兼容测试。
