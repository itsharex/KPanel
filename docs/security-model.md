# 安全模型

## 信任边界

- 浏览器只连接 `paneld`；生产环境必须位于 HTTPS 反向代理后。
- `paneld` 以非 root 用户运行，只持有面板数据库、Agent Socket 的组访问权和只读 Token。
- `kejilion-agent` 是唯一宿主机特权入口，只监听 Unix Socket，不监听 TCP。
- `kejilion.sh`、Docker、Nginx 配置和 `/home/web` 是外部事实来源，不受面板数据库支配。

## 首版防护

- 无默认账户；一次性 Bootstrap Token 成功使用后立即失效。
- 密码使用 Argon2id；Session 随机生成并只保存摘要。
- Cookie 为 `HttpOnly`、`SameSite=Strict`，公开环境强制 `Secure`。
- 所有状态变更要求 CSRF Token，并校验 `Origin` 与配置的公开地址。
- 登录失败按来源和账户分别限速：单 IP 使用配置阈值，账户阈值为其 10 倍，
  避免少量匿名请求锁死唯一管理员；正确登录会重置该来源和账户的既往失败计数。
  认证及管理操作写入脱敏审计日志。
- Agent 请求使用独立 Bearer Token，通过文件权限和 Unix Socket 组权限双重限制。
- 公共 API 不接受 Shell、命令行、绝对目标路径、Docker Exec 或原始 Nginx 配置。
- Docker 资源默认只读；只有归属证据充分的 Kejilion 资源允许生命周期操作。
- Agent 无法识别宿主机布局、缺少操作前置条件、检测到漂移或验证失败时，
  对应 capability 降级为只读。
- 系统写入只接受主机名、端口、IP、IANA 时区、大小和固定预设等类型化字段；
  变更前在 Agent 专属目录保存快照，失败时恢复配置并再次加载。
- 系统更新与清理只接受固定策略枚举，由独立 systemd transient service
  执行固定 APT/journalctl 参数。API 不接受包名、命令、路径或 Shell；
  维护任务拥有软件包升级所需的宿主机写权限，因此与普通 Agent 沙箱分离。
- `/swapfile` 调整同样由固定参数的一次性 systemd 事务执行；输入仅为
  0 或 256–65536 MiB 整数。事务拒绝符号链接，只接管 `/swapfile` 与旧版
  KPanel Swap，内存安全门和失败恢复通过后才报告成功。

## 与 kejilion.sh 并存

- 面板不执行、`source`、修改或替换 `kejilion.sh`。
- 每次读取都从真实产物重新发现状态，并计算 `resourceVersion`。
- 写入前再次核对资源版本；发现外部变化时返回冲突，不覆盖脚本或人工修改。
- 写文件使用同文件系统临时文件、同步和原子替换；Nginx 或系统配置校验、
  reload、监听探测或状态回读失败时恢复原文件。
- 现有脚本尚未使用 Agent 资源锁，因此无法宣称跨 CLI/Web 的严格串行；首版以冲突检测保护数据，后续只有在脚本显式接入 `kpctl` 后才能形成同一事务入口。

## 明确不支持

- 任意终端、Docker Exec、容器/镜像删除、Prune 或系统重装。
- 任意包安装/卸载、非 Debian/Ubuntu 软件包升级、清空日志与临时目录、
  Docker 清理，以及自定义更新/清理命令。
- SSH 旧端口删除、任意 DNS/软件源 URL、任意 sysctl 参数和任意脚本执行。
- 网站、数据库、证书和目录硬删除。
- 在线编辑任意 Nginx 文本或 Docker Compose 文件。
- 保存 Docker 环境变量、私钥、数据库密码或 Cookie 到日志和审计。
