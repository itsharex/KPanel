# kejilion.sh 兼容基线

## 已验证输入

- 脚本：`kejilion.sh` v4.5.2
- 脚本 SHA-256：`E45FB51E08E6536C9ECC890FE131D3E116D07103BFC8F2281FF0913299E598D3`
- Nginx 模板仓库提交：`05f5a2eac269967706f30dc3ff7985339e1f3ce4`
- 分析日期：2026-07-25

面板不会按版本字符串盲目判断兼容性。Agent 会同时检查实际目录、配置语法、Docker 资源和挂载关系；布局不明确时保持只读。

## 事实来源映射

| 资源 | kejilion.sh 产物 | 面板行为 |
| --- | --- | --- |
| 网站配置 | `/home/web/conf.d/*.conf` | 解析并计算内容哈希，不保存影子配置 |
| 网站文件 | `/home/web/html/<domain>` | 展示真实目录；仅在确认式完整删除中按固定布局清理 |
| 证书 | `/home/web/certs/<domain>_cert.pem`、`<domain>_key.pem` | 只读取公钥状态，不返回私钥 |
| Web 运行环境 | `/home/web/docker-compose.yml` 与 Docker Engine | 联合识别 Nginx、PHP、MySQL 等服务 |
| 应用 | Docker/Compose 与 `/home/docker` | Docker 是运行状态真相，目录仅作归属证据 |
| 应用编号 | `/home/docker/appno.txt` | 兼容发现，不作为容器运行状态 |
| 应用端口 | `/home/docker/*_port.conf` | 兼容展示，最终端口以 Docker 映射为准 |

## 双向呈现规则

- `kejilion.sh` 或人工变更真实产物后，下一次 Agent 对账会更新 Web 列表、状态、来源和 `resourceVersion`。
- Web 只通过固定模板写入与脚本相同的目录和文件命名；脚本随后读取时可直接看到这些产物。
- Web 不生成仅存于面板数据库、但宿主机不存在的“网站”或“容器”。
- 外部变更与 Web 更新冲突时，Web 请求失败并要求刷新；不静默覆盖。
- 未经脚本改造，CLI 与 Web 无法共享同一把事务锁。因此首版承诺“真实产物双向可见 + 冲突拒绝”，不虚假承诺严格串行。

## 首版写入边界

- 允许：固定模板的 HTTP/已有证书 HTTPS 静态站、反向代理站安全创建和更新；
  WordPress 独立安装事务可创建脚本同款数据库、源码、证书与 Nginx 产物。
- 允许：对配置哈希、主域名、文件名和固定布局均可核验的站点执行 Nginx 解绑；
  管理员逐字确认主域名后，可按 `web_del()` 产物范围完整删除目录、证书和同名数据库。
- 允许：归属证据充分的 Kejilion 容器启动、停止、重启。
- 禁止：归属或布局不明的网站/证书/数据库/目录删除，任意 Nginx 文本，任意 Shell、公共
  Docker Exec、Compose 在线编辑。WordPress 事务内部仅允许固定 MySQL 命令，
  且只能回滚本事务刚创建并仍可核验的产物。
