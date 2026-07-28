# KPanel v0.24.2 验收记录

## 交付范围

- 停止应用后保持“已安装”状态与详情入口；从“运行中”筛选停止时自动切换到“已安装”。
- 对可信脚本配置和实际容器共同确认的应用开放更新、卸载及直接访问策略，不再依赖旧版
  `appno.txt` 标记作为能力开关。
- 更新、卸载和检查更新遇到页面资源版本过期时，刷新真实容器状态并重试一次；资源冲突与
  后台任务冲突使用不同错误码。
- 只有脚本安装标记但缺少实际容器时，仅提供恢复终端，不伪装成可更新或可卸载。
- 应用详情的域名访问和 IP + 端口访问统一改为上下排布。
- 概览发行版标志改为品牌色底板与单色本地矢量图标，兼容浅色和深色主题；未知发行版使用
  Linux 标志。

## 发布提交

- KPanel：`f82cb9e375621918e5e1a515db839423d5afdd8e`
- 标签：`v0.24.2`
- `kejilion.sh`：
  `9c19fc3f10e77dd352e5f6f7a0c53d8a1ba64761`
- 镜像内脚本 SHA-256：
  `30c5cfb8862f1896c96f71fbd8a1a692eacd968e90c04f7e2b93b57f2f106784`

## 自动验收

- 主分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30365199521>
- Release：
  <https://github.com/kejilion/KPanel/actions/runs/30365510582>
- Ubuntu 24.04 Runner 上的 Go 全量测试、`go vet`、Web 类型检查、62 项 Vitest、
  生产构建、安装安全测试、`kejilion.sh` 应用生命周期测试全部通过。
- `govulncheck v1.6.0` 与 `npm audit --audit-level=high` 通过。
- linux/amd64、linux/arm64 Agent 构建通过。
- 原生镜像在非 root、只读根文件系统、`cap-drop ALL`、`no-new-privileges` 和无网络条件下
  通过健康检查；OCI 版本、源码提交、脚本提交和脚本摘要检查通过。
- 双架构镜像推送、`latest` 提升、GitHub Release 发布、SBOM/Provenance 生成全部通过。

## 发布产物

- GitHub Release：
  <https://github.com/kejilion/KPanel/releases/tag/v0.24.2>
- 生产镜像：
  `docker.io/kjlion/kejilion-panel@sha256:060569d6e55277c0fdd7a85665a8e8bc1b7f28ef756df4529c5c3600db8cea1c`
- `0.24.2` 与 `latest` 已由发布工作流复核为同一 manifest digest。
- 发布附件已重新下载，并按附件中的 `SHA256SUMS` 独立校验：
  - `kejilion-agent-linux-amd64`：
    `42fd8ddba49a4ef2913042a5e74b36f87b1188a9137e678d2199362cef31ced5`
  - `kejilion-agent-linux-arm64`：
    `c08f5fb78b53e904a8a40bf06f5d38d9f27050f67f562fbbacf224c65e32e0be`
  - `kejilion-panel-deploy-0.24.2.tar.gz`：
    `aadea2050df913771a8b9f551c7352bf5f57b5dffe4262b4c50072d140ca6683`
- linux/amd64 Agent 实际执行 `version` 输出 `0.24.2 v1alpha1`；部署包中的 `VERSION`
  为 `0.24.2`。

## 回滚

- 代码回滚点：标签 `v0.24.1`。
- 镜像回滚点：
  `docker.io/kjlion/kejilion-panel@sha256:d6da7b6be360520a6847d7a39168520ed1e16db22c8c5c16ff4a82089b3e55ce`
- 本版本没有数据格式迁移。回滚 KPanel/Agent 不删除 `/home/web`、应用容器、域名配置或环境备份。

## 边界

- 本次发布完成 GitHub Release、Docker Hub 版本镜像和 `latest` 更新。
- 没有自动替用户升级任何生产主机；用户从应用市场更新 KPanel 时才执行原位升级与失败回滚。
