# Changelog

本项目遵循语义化版本。所有日期均使用 `YYYY-MM-DD`。

## [0.1.0] - 2026-07-25

首个可部署版本。

### Added

- 一次性初始化、安全登录、Argon2id 密码哈希、服务端 Session、CSRF/Origin/Host 校验和登录限速。
- 通过受限 Unix Socket Agent 读取 Linux 主机状态、`/home/web` 网站产物和 Docker 资源。
- 现有静态站、反向代理站、证书及未知 Nginx 配置的只读发现与资源版本展示。
- 固定模板静态站和内网反向代理站的安全创建、更新、`nginx -t`、原子替换、reload 与失败回滚。
- 经归属验证的 Kejilion 容器启动、停止、重启、有界日志与一次性资源统计。
- 管理变更记录、结构化审计、敏感字段脱敏和 Agent 离线只读降级。
- 非 root 多架构 Panel 镜像、双架构 Agent、校验和、SBOM/Provenance 发布流程及隔离安装器。

### Compatibility

- 兼容基线为 `kejilion.sh` v4.5.2 与 Nginx 模板提交 `05f5a2eac269967706f30dc3ff7985339e1f3ce4`。
- Panel 不修改、执行或 `source` `kejilion.sh`；宿主机真实产物始终是事实来源。
- 脚本侧或人工修改会在下一次发现时呈现；Web 侧仅写入脚本既有路径和命名约定。

### Known limitations

- 首版不提供网站、证书、数据库、目录或 Docker 资源删除。
- 首版不提供任意 Shell、任意 Docker Exec、Compose 在线编辑、系统重装或应用市场远程脚本。
- 未改造的 `kejilion.sh` 与 Web 不共享事务锁；外部并发变更通过资源版本和再次校验拒绝覆盖。
