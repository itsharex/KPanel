# KPanel v0.24.0 验收记录

## 交付范围

- LDNMP 环境在无备份或辅助列表读取失败时仍可正常显示和管理。
- 应用绑定域名后立即刷新详情，并提供公网 IP、IPv6 与 DNS 托管平台解析辅助。
- 已安装的脚本应用可通过固定应用编号打开 `kejilion.sh` 原生交互管理终端。
- 应用站点列表短暂失败时保留上次结果并重试，兼容等价 loopback 反代地址。
- 只读镜像更新检查自动处理容器资源版本竞态；写操作仍执行严格版本校验。
- 应用交互任务允许 `AF_NETLINK`，使 `iptables-nft` 可在宿主机上下文执行直连开关。

## 自动验收

- Linux：`go test ./...`、`go vet ./...` 全部通过。
- Web：TypeScript 检查、53 项 Vitest 和生产构建全部通过。
- Agent：linux/amd64、linux/arm64 静态编译通过。
- 部署：安装安全测试通过；Release 工作流中的脚本应用生命周期测试通过。
- 功能分支发布候选 CI：
  `https://github.com/kejilion/KPanel/actions/runs/30340724903`
- 主分支 CI：
  `https://github.com/kejilion/KPanel/actions/runs/30340872619`
- Release：
  `https://github.com/kejilion/KPanel/actions/runs/30341024472`

## 线上核查（2026-07-28）

- GitHub Release `v0.24.0` 已发布且不是草稿，包含 amd64/arm64 Agent、部署包和
  `SHA256SUMS`。
- Docker Hub `0.24.0` 与 `latest` 均指向
  `sha256:d9d54d7945d0a3510780134e7a65475560d1cc69e1fd20e1cb9a571b15829d02`，
  包含 linux/amd64 与 linux/arm64。
- linux/amd64 镜像 OCI 版本为 `0.24.0`，源码提交为
  `2561e542c4e712c52b7ccb40c1b6299887f6e679`，运行用户为 `65532:65532`，
  健康检查为 `CMD /paneld healthcheck`。
- 镜像固定脚本提交为 `cd4a97823e95f4029f6cb3a82249f2adf5d53763`，
  SHA-256 为 `dec802845150762a977c2dbf300a7ccf20e2e95135f9a4e4e751069d1a834259`。
- 发布附件全部通过 `SHA256SUMS`；linux/amd64 Agent 实际执行
  `version` 输出 `0.24.0 v1alpha1`。

## 回滚

- KPanel 镜像回滚到 `v0.23.0` /
  `sha256:e26a35860c36acb662eca918d20e4c0758875f4b1d285f04c9e8413316bd6414`。
- 本版本没有数据库格式迁移；回滚 KPanel/Agent 不删除 `/home/web`、应用容器、
  域名配置或环境备份。
- 目标主机升级后仍需实际执行一次应用“允许/阻止 IP+端口访问”，确认该机器的
  nftables/iptables 内核能力与宿主机策略没有额外限制。
