# KPanel v0.42.1 发布验收

## 发布范围

v0.42.1 修复 v0.42.0 双网络迁移后，宿主机 Nginx 通过 HTTPS 域名访问时可能返回
`421 host_validation_failed` 的问题。根因是宿主机到已发布 Panel 端口的连接来自
`kejilion-panel-egress` 网关，而更新器只将 `kejilion-panel-internal` 网段写入
`KPANEL_TRUSTED_PROXY_CIDRS`。

修复同时覆盖全新安装和更新：保留内部代理网段，并仅加入出口网络的宿主机网关单地址
（IPv4 `/32`、IPv6 `/128`），不信任整个出口网段。

## 版本与制品

- KPanel 提交：`89d8ec9d0d4735beaee89df2a831c548be87eeb3`
- 标签：`v0.42.1`
- 应用市场提交：`61f645d`
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.42.1>
- Docker 多架构摘要：`sha256:8269a8a25693814b50ec9ca5b38d12fb60550109e91771da9d384bea99db5eae`
- linux/amd64：`sha256:30b55e4eab6687c623cd99ff9cf838f8c858b59b862879843cf74e3759463187`
- linux/arm64：`sha256:09dd99430c29a464e415daadb9ed0f9cb491b667d5be7dcd1af11240e2cc17f4`
- `0.42.1` 与 `latest` 指向同一多架构摘要。
- 公开应用市场 `kpanel.conf` SHA-256：
  `b1e23371da402ebfcb61d56a377f77b6b78c44a454e23b99119bab5aaf895d0f`

## 自动化验收

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/30874071055>，成功。
- main CI：<https://github.com/kejilion/KPanel/actions/runs/30874192470>，成功。
- Release：<https://github.com/kejilion/KPanel/actions/runs/30874311010>，成功。
- 前端 typecheck、生产构建成功；Vitest `38` 个文件、`243` 项测试通过。
- `packaging/tests/app-conf-lifecycle.sh` 在固定 Bash 容器通过，覆盖安装、更新、失败回滚及双网络可信代理迁移。
- Release 门禁通过 Go test/vet、版本一致性、源代码与依赖安全扫描、应用生命周期、
  CGO-free amd64/arm64 构建、原生镜像扫描和运行时镜像合约。

## 真机回归

在 154 Debian 13 的 v0.42.0 双网络实例上，修复前使用域名 Host 和
`X-Forwarded-Proto: https` 请求返回 `421`；加入出口宿主机网关 `/32` 后返回 `200`，
IP 直连仍为 `200`，Panel 保持 `healthy`。

随后使用公开应用市场 `kpanel.conf` 完成 v0.42.0 → v0.42.1 更新，验收结果：

- Agent：`0.42.1 v1alpha1`
- Panel 健康接口：`version=0.42.1`、`status=ok`
- 域名 HTTPS 转发路径：`200`
- IP 直连路径：`200`
- 更新后的可信代理包含内部网段及出口宿主机网关 `/32`，未被更新流程覆盖。

## 回滚与待执行项

更新器在失败时恢复旧镜像、Agent、Compose、systemd 单元和 `.env`；业务数据目录不迁移。
报告故障的 `kpanel.kejilion.eu.org` 主机尚未由本次任务直接变更：当前工作环境保存的 SSH
密钥对该机无权限。该机再次执行 KPanel 应用市场更新后即可获取 v0.42.1 修复，随后需从
公网复核首页、登录、AI 页面和健康接口。
