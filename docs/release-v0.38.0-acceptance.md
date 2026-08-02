# KPanel v0.38.0 发布验收

发布日期：2026-08-02

## 发布范围

- 完成当前公开业务页面的 `zh-CN` / `en-US` 覆盖，英文业务词典按路由懒加载。
- 修复语言切换后动态弹窗、提示、占位文本和运行中任务状态未同步的问题。
- 非面板 Linux 主机接入命令改用 `https://kejilion.sh` 官方短入口。
- 根目录 `kejilion.sh` 与 `cn/kejilion.sh` 使用同一轻量节点协议，并兼容缺少 `useradd` 的精简 systemd 主机。
- KPanel 镜像固定新的脚本提交与 SHA-256，不改变节点权限、出站上报和二进制摘要校验边界。

## 源码与自动化

- `kejilion.sh` 提交：`e9f1670d0a89b06bf3e690509deb184e4804d6dd`
- 根脚本 SHA-256：`d9143d1aa0f02fce0bccc4b8e66d24964325eb0bf6d818b6f553081da23a748b`
- KPanel 短入口提交：`0813363`
- 完整多语言提交：`1cc1a19`
- 发布准备提交：`54229128dee06790b3993ef9dcb2c05a5445b187`
- 标签：`v0.38.0`
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30743682576> — 成功
- Release：<https://github.com/kejilion/KPanel/actions/runs/30743759966> — 成功
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.38.0> — 已公开

## 功能、性能与安全验收

- Shell：根脚本与中文脚本均通过 `bash -n` 和轻量节点安装器 smoke test；除 `canshu="default"` / `canshu="CN"` 外内容一致。
- Web：31 个测试文件共 211 项测试通过，类型检查和生产构建通过。
- 多语言：覆盖检查确认 14 个按需词典共 1409 条文案；后续轻量节点新增的 HTTPS 提示已补齐英文资源。
- Go 与 Linux：主线 CI 完成全量测试、`go vet`、双架构构建和部署测试。
- 安全：Release 完成 Go 调用路径漏洞扫描、Node 高危依赖审计、脚本应用生命周期测试、镜像运行契约和脚本摘要验证。
- 性能：中文用户不下载英文业务词典；切换语言不重复请求业务 API，未增加后台常驻服务或新数据存储。

## 发布产物

- `kejilion-agent-linux-amd64`：`0ac4aac1c6390a6839b7200c074e1571ad19883fd4eb5d6b34815cf058efad55`
- `kejilion-agent-linux-arm64`：`d1fe1c2fb62f85663b45221c758d359c4fb28cdfab68e82c3d427d009d889069`
- `kejilion-node-linux-amd64`：`614c271c957ee469560d6ebf01c687d9aebf2b67123415517e6c730fdfe69d8b`
- `kejilion-node-linux-arm64`：`44369c3e8ee1b02d8187120cbe01711c6e7ef441ce2ab659116924e758eb4303`
- `kejilion-panel-deploy-0.38.0.tar.gz`：`002320c07dde8688f5da6d0e0ec6b64c9d27de9dc9f1651a178dc922301fcc56`

## Docker Hub

- `docker.io/kjlion/kejilion-panel:0.38.0`
- `docker.io/kjlion/kejilion-panel:latest`
- 两个标签由 Release 流水线逐一回读并确认指向同一 OCI index：
  `sha256:d5eb792b6d49d9d79c92b9d5561b1245fcf73a44ea34db1f6343955badcd01fc`
- 当前本地网络访问 Docker Hub API 超时，因此不重复宣称本机直连复核；发布结论以成功的 Release 回读门禁和公开 Release 摘要为准。

## 兼容与回滚

- 兼容现有 KPanel 数据、端口、反向代理、语言偏好、集群配置和已接入轻量节点，无需迁移。
- 回滚源码与标签：`v0.37.1`
- 回滚镜像：`docker.io/kjlion/kejilion-panel:0.37.1`
- 回滚只切换 KPanel 镜像和程序版本，不删除用户、站点、容器、文件、集群配置或节点身份数据。
