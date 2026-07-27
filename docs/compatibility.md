# kejilion.sh 兼容基线

## 已验证输入

- 脚本：`kejilion.sh` v4.5.2
- 脚本 SHA-256：`E45FB51E08E6536C9ECC890FE131D3E116D07103BFC8F2281FF0913299E598D3`
- Nginx 模板仓库提交：`05f5a2eac269967706f30dc3ff7985339e1f3ce4`
- 分析日期：2026-07-25

面板不会按版本字符串盲目判断兼容性。Agent 同时读取实际目录、配置语法、Docker 资源和挂载关系；
布局不明确时仍展示真实产物和已实现动作，并明确指出缺少哪一种结构化适配器。

## 事实来源映射

| 资源 | kejilion.sh 产物 | 面板行为 |
| --- | --- | --- |
| 网站配置 | `/home/web/conf.d/*.conf` | 解析并计算内容哈希，不保存影子配置 |
| 网站文件 | `/home/web/html/<domain>` | 展示真实目录；完整删除按实际发现路径在共享根目录内清理 |
| 证书 | `/home/web/certs/<domain>_cert.pem`、`<domain>_key.pem` | 只读取公钥状态，不返回私钥 |
| Web 运行环境 | `/home/web/docker-compose.yml` 与 Docker Engine | 联合识别 Nginx、PHP、MySQL 等服务 |
| 应用 | Docker/Compose 与 `/home/docker` | Docker 是运行状态真相，目录和 marker 用于生态发现而非管理授权 |
| 应用编号 | `/home/docker/appno.txt` | 兼容发现，不作为容器运行状态 |
| 应用端口 | `/home/docker/*_port.conf` | 兼容展示，最终端口以 Docker 映射为准 |

## 双向呈现规则

- `kejilion.sh` 或人工变更真实产物后，下一次 Agent 对账会更新 Web 列表、状态、来源和 `resourceVersion`。
- Web 外联写入必须通过脚本非交互协议、脚本同一权威模板或双方共享生成器完成；仅写入相同目录
  和文件名不再视为兼容。
- Web 不生成仅存于面板数据库、但宿主机不存在的“网站”或“容器”。
- 外部变更与 Web 更新冲突时，Web 请求失败并要求刷新；不静默覆盖。
- 未经脚本改造，CLI 与 Web 无法共享同一把事务锁。因此首版承诺“真实产物双向可见 + 冲突拒绝”，不虚假承诺严格串行。

## 写入与适配边界

- 一键成品站已使用脚本非交互协议。静态站、PHP、反向代理、负载均衡、重定向和 WordPress
  仍存在 KPanel 自编流程，已在 `external-config-sources.md` 登记为冻结迁移债务，不得宣称
  使用 `k web` 同一套配置。
- 脚本、Web、人工修改和孤立站点均可按实际资源 ID 删除；完整删除不要求逐字确认主域名，
  但仍将路径限制在共享网站根目录并保留 Nginx 失败回滚。
- 所有 Docker 容器均可按实时状态启动、停止、重启、删除、查看日志/性能、进入单次控制台
  和变更外部访问，不以归属证据授权。
- 复杂 Nginx 结构若尚无可靠补丁适配器，会明确返回“结构更新适配器未实现”；不得描述为
  “外部资源只读”。宿主机终端、交互式 TTY 和 Compose 通用编辑器同样属于待实现能力。
