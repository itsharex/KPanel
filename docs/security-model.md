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
- `k info` 公网信息只访问 Agent 内置的 IPinfo IPv4/IPv6 HTTPS 端点，不接受
  用户 URL；使用地址族校验、短超时、响应大小限制、同主机重定向限制和
  30 分钟缓存。可通过环境变量关闭，查询失败不会阻塞本地主机监控。
- Docker 资源默认只读；只有归属证据充分的 Kejilion 资源允许生命周期操作。
- Agent 无法识别宿主机布局、缺少操作前置条件、检测到漂移或验证失败时，
  对应 capability 降级为只读。
- 系统写入只接受主机名、端口、IP、IANA 时区、大小和固定预设等类型化字段；
  变更前在 Agent 专属目录保存快照，失败时恢复配置并再次加载。
- 系统更新与清理只接受固定策略枚举，由独立 systemd transient service
  按 `/etc/os-release` 白名单执行固定 APT、DNF/YUM、Pacman、Zypper 和
  journalctl 参数。API 不接受包名、命令、路径或 Shell；维护任务拥有软件包
  升级所需的宿主机写权限，因此与普通 Agent 沙箱分离。
- `/swapfile` 调整同样由固定参数的一次性 systemd 事务执行；输入仅为
  0 或 256–65536 MiB 整数。事务拒绝符号链接，只接管 `/swapfile` 与旧版
  KPanel Swap，内存安全门和失败恢复通过后才报告成功。
- 内核调优只接受五个场景枚举或还原操作，参数由 Agent 内置生成；Web 不能提交
  sysctl 键值。脚本合法产物按标识和场景双重识别，未知同名文件拒绝覆盖。

## 与 kejilion.sh 并存

- 面板不 `source`、修改或替换 `kejilion.sh`。应用安装只允许 Agent 通过固定的
  `app <数字编号或 token>` 参数和非交互环境变量调用支持 KPanel 协议的本机脚本；
  API 不接受脚本路径、动作、命令或其他参数。
- 只有脚本已记录 `permission_granted="true"` 时才开放安装；首次许可必须由用户在
  终端运行 `k` 明确接受，Web 不模拟输入或代替同意。
- 脚本安装在独立 systemd 临时服务中运行，任务状态和有界日志保存为 `0600`；
  安装器生成的初始应用凭据可能出现在该日志中，因此仅向已认证管理员显示，
  不写入 Panel 审计记录。
- 每次读取都从真实产物重新发现状态，并计算 `resourceVersion`。
- 写入前再次核对资源版本；发现未知外部变化时返回冲突。只有结构和标识均符合
  当前 `kejilion.sh` 的内核预设允许在用户确认后切换，并先保存完整快照。
- 写文件使用同文件系统临时文件、同步和原子替换；Nginx 或系统配置校验、
  reload、监听探测或状态回读失败时恢复原文件。
- `k fd` 可以把域名反代到 KPanel 的直连端口。只有立即来源位于显式可信代理
  CIDR 且 `X-Forwarded-Proto` 严格为单一 `https` 时，Panel 才接受代理传入的
  HTTPS Host/Origin、启用 Secure Cookie，并解析真实客户端地址；公网客户端伪造
  转发头不会改变安全边界。
- 现有脚本尚未使用 Agent 资源锁，因此无法宣称跨 CLI/Web 的严格串行；首版以冲突检测保护数据，后续只有在脚本显式接入 `kpctl` 后才能形成同一事务入口。

## 明确不支持

- 任意终端、公共 Docker Exec、容器/镜像删除、Prune 或系统重装。
- 任意指定包安装/卸载、白名单外发行版升级、清空日志与临时目录、Docker
  清理，以及自定义更新/清理命令。
- SSH 旧端口删除、任意 DNS/软件源 URL、任意 sysctl 参数，以及应用安装协议之外的任意脚本执行。
- 网站、数据库、证书和目录硬删除。
- 在线编辑任意 Nginx 文本或 Docker Compose 文件。
- 保存 Docker 环境变量、私钥或 Cookie 到日志和审计；应用安装器主动输出的初始凭据
  只进入受限任务日志。
