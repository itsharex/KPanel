# kejilion.sh 网站业务分析与兼容契约

> 历史分析说明：业务产物映射仍可参考，但文中的“漂移后只读”“未知模板只读”等限制
> 已由 [`PROJECT_RULES.md`](../PROJECT_RULES.md) 废止。文中允许 KPanel 自行生成 HTTP
> 模板的方案也已被“外联配置直接复用硬规则”废止，不能作为后续实现依据。

## 验证基线

- `kejilion.sh`：v4.5.2
- 脚本 SHA-256：
  `E45FB51E08E6536C9ECC890FE131D3E116D07103BFC8F2281FF0913299E598D3`
- `sh` 仓库提交：`74a80690d206a89abe9d11e8137671b6fa402688`
- `nginx` 模板提交：`05f5a2eac269967706f30dc3ff7985339e1f3ce4`
- 分析日期：2026-07-25

本分析只读取本地仓库，没有执行、`source` 或修改脚本。

## 实际产物

| 宿主机路径 | Nginx 容器路径 | 作用 |
| --- | --- | --- |
| `/home/web/nginx.conf` | `/etc/nginx/nginx.conf` | 全局配置 |
| `/home/web/conf.d/*.conf` | `/etc/nginx/conf.d/*.conf` | 网站配置 |
| `/home/web/certs/*_cert.pem` | `/etc/nginx/certs/*_cert.pem` | 证书链 |
| `/home/web/certs/*_key.pem` | `/etc/nginx/certs/*_key.pem` | 私钥 |
| `/home/web/html` | `/var/www/html` | 网站文件 |
| `/home/web/letsencrypt` | `/var/www/letsencrypt` | ACME Webroot |
| `/home/web/log/nginx` | `/var/log/nginx` | Nginx 日志 |

普通域名配置名为 `<domain>.conf`，IP 自定义端口配置可能为
`<ip>_<port>.conf`。证书名为 `<domain>_cert.pem` 和
`<domain>_key.pem`。`default.conf`、`map.conf` 是公共配置，不是网站。

依据：

- 目录初始化：`C:\GitHub\kejilion\sh\kejilion.sh:1312`
- Docker 挂载：`C:\GitHub\kejilion\docker\LNMP-docker-compose-10.yml:2`
- Nginx include：`C:\GitHub\kejilion\nginx\nginx10.conf:245`

## 脚本创建流程

静态站入口位于 `kejilion.sh:9725`。它下载 `html.conf`，替换域名，创建
`/home/web/html/<domain>`，可下载并解压用户提供的 ZIP，最后直接 reload
Nginx。模板同时声明 80、443 和 QUIC，固定引用域名证书并把 HTTP 跳转至
HTTPS。

IP + 端口反代入口位于 `kejilion.sh:3499`。它使用
`reverse-proxy-backend.conf`，生成随机 upstream 名称，写入目标
`IP:port`，覆盖 `map.conf` 后直接 reload。负载均衡入口位于
`kejilion.sh:3555`，可写入多个 upstream。域名反代位于
`kejilion.sh:9622`，使用另一套包含 SNI、缓存和内容替换的模板。

证书流程 `install_ssltls()` 位于 `kejilion.sh:1437`。它会停止 Nginx，
使用 Certbot standalone 或 IP 自签名证书，再复制到 `/home/web/certs`。
这一流程会改变线上服务状态，不能由首版 Panel 自动复刻。

## 已验证风险

- 域名、上游、端口和路径缺少严格输入校验。
- `repeat_add_yuming()` 遇到同名配置时不是返回冲突，而是调用
  `web_del()`；后者会删除配置、目录、证书和可能的同名数据库
  （`kejilion.sh:1618`、`:1786`）。
- 网站创建和手工编辑直接 reload，没有先执行 `nginx -t`。
- 现有“修改网站”是任意文本编辑，不具备备份、资源版本和回滚。
- 脚本站点列表主要按证书文件名和配置文件名拼接，不是完整 Nginx 语义解析；
  缺失证书、孤立证书、重复域名和文件名不一致都可能被误判。
- `html.conf` 的默认 root 没有尾部 `/`，而脚本修改嵌套 root 时搜索带尾部
  `/` 的文本，因此当前模板下该替换可能不生效。
- 代理模板依赖 `map.conf`；脚本的部分入口不会保证它存在。
- 脚本运行时下载远程 `main` 模板，业务形态可能独立于脚本版本变化。

以上是历史风险分析，不再构成“自行实现另一套模板”的理由。当前规则要求先为脚本补充稳定
非交互入口，或直接消费脚本使用的同源模板；KPanel 只能在外层增加输入校验、原子暂存、
`nginx -t`、备份和失败回滚。

## 历史 v0.9 双向呈现规则（写入方案已废止）

读取方向：

1. 每次查询扫描真实的 `conf.d`、`html`、`certs` 和 Docker Engine。
2. 保守解析 `server_name`、`listen`、`root`、`ssl_certificate`、
   `proxy_pass`、`upstream/server` 和 `fastcgi_pass`。
3. 以配置、证书和 HTML 目录的并集发现资源。
4. 配置重复、symlink、未知语法和目录读取失败必须显式显示为
   `ambiguous`、`unsupported` 或错误，不能显示成“空列表”。
5. 证书只读取公钥文件，校验有效期、hostname 和对应私钥文件是否存在；
   私钥内容不进入 API、日志、摘要或审计。

写入方向：

1. 允许固定、内置、无 TLS 的 HTTP 模板创建静态站、PHP 动态站、
   IP/端口反代、域名反代、负载均衡和域名重定向。
2. 产物仍使用 `/home/web/conf.d/<domain>.conf` 与
   `/home/web/html/<domain>`，因此脚本侧列表能发现。
3. 已存在任何同名配置、目录或域名归属时返回 `409`，绝不调用删除。
4. 只有带 Panel 管理标记且仍匹配固定模板的站点允许 Web 更新；脚本和人工
   创建的配置默认只读，避免覆盖自定义指令。
5. 输入只接受结构化域名、别名、PHP 版本、跳转状态码和 HTTP(S) origin；
   负载均衡与脚本一致使用 2–8 个 HTTP upstream。不接受 Shell、任意
   Nginx 文本或绝对目标路径。
6. 写入使用同文件系统临时文件、`fsync` 和原子替换。
7. 固定执行 `nginx -t`，成功后才 reload；失败时在确认没有外部并发改写后
   恢复原产物。
8. 更新必须携带由实际配置和证书计算出的 `resourceVersion`；外部变化返回
   `409`，不静默覆盖。
9. 删除只对配置文件名与主域名一致、结构已识别且未漂移的站点开放；可只移除
   Nginx 入口，或在逐字确认域名后清理 `html/<domain>`、域名证书和同名数据库。

旧脚本不遵守 Panel 的资源锁，所以 v0.9 能保证真实产物双向可见和冲突拒绝，
不能宣称 CLI/Web 严格串行。只有未来让脚本显式通过 `kpctl` 调用 Agent，才能
形成同一事务入口。

## v0.9 新建站点服务矩阵

| Web 选择 | `kejilion.sh` 对应业务 | Panel 产物与限制 |
| --- | --- | --- |
| WordPress | `ldnmp_wp` / `KJ_WEB_RECIPE=2` | 后台执行脚本固定非交互分支；源码、数据库、Redis、TLS 与 Nginx 产物均由脚本生成 |
| 静态网站 | `k static-site <domain>` / `KJ_WEB_RECIPE=30` | 面板先收域名；脚本终端继续接收 ZIP 下载地址和 `index.html` 入口路径 |
| PHP 网站 | `k php-site <domain>` / `KJ_WEB_RECIPE=20` | 面板先收域名；脚本终端继续接收源码、入口、PHP 版本、扩展与数据库配置 |
| IP / 端口反代 | `ldnmp_Proxy` / `KJ_WEB_RECIPE=23` | 后台执行脚本固定非交互分支，并对账脚本生成的官方反代配置 |
| 域名反代 | `k domain-proxy <domain>` / `KJ_WEB_RECIPE=24` | 脚本终端继续接收目标域名并生成官方 SNI 反代模板 |
| 负载均衡 | `k loadbalance-site <domain>` / `KJ_WEB_RECIPE=28` | 脚本终端继续接收后端节点并生成官方 upstream 配置 |
| 域名重定向 | `k redirect-site <domain>` / `KJ_WEB_RECIPE=22` | 脚本终端继续接收跳转目标并生成官方重定向模板 |

所有类型仍写入脚本可发现的 `/home/web/conf.d/<domain>.conf`；静态和 PHP
目录映射保持 `/home/web/html/<domain>` → `/var/www/html/<domain>`。
所有新建任务统一持久化到
`/var/lib/kejilion-panel/site-recipe-jobs`。Agent 通过独立 systemd 后台单元执行
脚本原生命令，允许脚本按自身流程安装缺失的 Docker、Certbot 或 LDNMP 环境；
KPanel 不再为新建站点维护第二套 WordPress、静态站、PHP、反向代理、负载均衡或
重定向模板。Nginx Stream 不属于域名站点创建，继续由脚本专用入口管理。

历史结构化编辑仍保留旧模板兼容，但处于冻结状态；新建链路不再调用该生成器。

## 固定模板摘要

用于兼容回归的当前模板 SHA-256：

| 模板 | SHA-256 |
| --- | --- |
| `html.conf` | `A06D62750F641A47114194A105855B3008F0087CDDD191EC728C49FDD32EACE7` |
| `reverse-proxy-backend.conf` | `EA20107B5809C53BB4408968896E297AC1167E36CE99E0430E097D627D3B3F6D` |
| `reverse-proxy-domain.conf` | `A6A3B054A3C38CDAA980DB44260BFE042C4F55E60894456AF359DD02AAC7E105` |
| `map.conf` | `C35A546A15A40633AF6FA8D5F77E5426F03C545E88AEB915BD3A41DE9A5FF684` |

Panel 不在运行时下载这些远程模板。未知的新模板只进行保守发现并保持只读。
