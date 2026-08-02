# KPanel v0.37.0 发布验收

发布日期：2026-08-02

## 发布范围

- 集群新增非 KPanel Linux 主机接入能力，保留原有 KPanel 节点完整配对方式。
- 非面板主机仅需 `curl`/`wget`、systemd 和常见 Linux 用户态工具，不依赖 Docker、Go 或 Node.js。
- 中心端生成一次性接入命令；轻量节点仅主动通过 HTTPS 上报只读遥测，不开放远程 Shell、文件或 Docker 管理。
- 上报覆盖 CPU、内存、磁盘、网络、系统、地区和运行时间等摘要，沿用集群列表与排序体验。
- 节点自动更新固定发布版本与 SHA-256，下载或校验失败时保留旧二进制，不把失败更新判定为成功。

## 源码与自动化

- `kejilion.sh` 协议与安装器提交：`60b7982f5622c5be958a0c8197d3077783505494`
- KPanel 功能提交：`5804922`
- 发布准备提交：`280bcd66f869709f4a7475347de3112763f79402`
- 标签：`v0.37.0`
- 候选分支 CI：<https://github.com/kejilion/KPanel/actions/runs/30736616586> — 成功
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30736694515> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30736769222> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.37.0>

GitHub Release 已公开并包含 8 个附件：amd64/arm64 Agent、amd64/arm64 轻量节点、部署归档、`SHA256SUMS`、许可证和第三方声明。

## 功能、性能与安全验收

- Shell：索引中的 `kejilion.sh` 与 `cn/kejilion.sh` 通过 `bash -n` 和轻量节点安装器 smoke test；远端脚本与固定提交的 SHA-256 一致。
- Go：集群、轻量节点和 Panel 集群路径测试通过；对应包 `go vet` 通过。
- Web：类型检查、30 个测试文件共 207 项测试和生产构建通过。
- 构建：Panel、Agent、轻量节点和 `kpctl` 的 linux/amd64、linux/arm64 交叉构建通过。
- 安全：一次性接入令牌、HMAC 请求签名、时间窗口、重放保护、节点身份绑定、只读遥测边界及 systemd 沙箱约束均有实现与自动化覆盖。
- 性能：节点使用固定采样周期和有界响应，不部署额外数据库、容器运行时或常驻脚本解释器。

Windows 本机执行 `go test ./internal/panel` 时仍会触发现有 Linux 绝对路径和静态资源嵌入差异；本次改动相关定向测试通过，发布结论以成功的 Linux CI 为准。

## 网络中断完整性

- 上报目标不可达时，轻量节点保持运行并按退避周期重试，不退出、不写入成功凭据，也不影响中心端既有节点数据。
- 更新器在完全断网环境下载失败时，旧二进制内容保持不变，未残留 `.new` 或 `.previous` 文件。
- 新二进制只在下载完成并通过 SHA-256 后原子替换；替换失败路径保留旧版本，可由 systemd 继续启动。
- 安装与更新状态以完成凭据和产物复核共同判定，不以进程退出码或网络连接断开单独判定成功。

## 发布产物

- `kejilion-node-linux-amd64`：
  `1715a97a5dd259c5cac5598c6e446679aba923b6a71954851f460bf94d1687b3`
- `kejilion-node-linux-arm64`：
  `52359eed4bc41ce61385471b7ee1b11e53fbb4729bfc9abc6b6b83fa1ceb6b3a`
- amd64 轻量节点实际执行返回：`0.37.0 light-v1`

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.37.0`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签的 OCI index 摘要一致：
  `sha256:1e1caa036f32a29705012ff36b1eec78f51ec652a877672f163daacd39fc5350`
- linux/amd64：
  `sha256:22204cb63e46b7d5c49b303d0332542c596bbe4d45dcb509930f9c2873f4caf7`
- linux/arm64：
  `sha256:ef5d6f60da8d8f0ab0270fb2f4345d495374787ddfa488b4d0886897ec53d1e0`

清单中的 `unknown/unknown` 项为 provenance/SBOM attestation，不是缺失架构。

## 154 实机状态

- v0.37.0 的 GitHub Release、Docker Hub 镜像和 `kejilion.sh` 固定提交已经公开上线。
- 发布后尝试通过保存的 SSH Key、生产域名、生产 IP、容器网络和浏览器连接 154，均未建立可用管理链路；容器侧访问域名和 IP 均在 10 秒连接超时。
- 因无法进入 154，本次未执行 `k app kpanel`，也未改写其生产 Compose、端口、数据目录、反向代理或业务数据。
- 154 升级及真实 systemd 轻量节点接入属于待网络恢复后的 L3 补验项；在补验完成前，不宣称 154 已运行 v0.37.0。

## 回滚

- 源码与标签：`v0.36.0`
- 镜像：`docker.io/kjlion/kejilion-panel:0.36.0`
- 镜像摘要：
  `sha256:0bba49179d9ea0787836154c8bbea0f59d666a418620b61c9ddf8bfd5f8fb736`

回滚只切换 KPanel 镜像和程序版本，不删除用户、站点、容器、文件、集群配置或轻量节点身份数据。
