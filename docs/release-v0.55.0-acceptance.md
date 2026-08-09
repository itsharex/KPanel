# KPanel v0.55.0 发布验收记录

日期：2026-08-09  
发布级别：L3  
生产目标：`arena-154`（`154.36.153.9:8080`）

## 发布范围

本版本包含以下已冻结内容：

- `4f072fa`：带 Web 地址的应用、网站和自定义 URL 统一进入桌面内置浏览器；
- `ae758be`：跨域内嵌页深浅主题、滚动条和加载背景修复；
- `7b0fad5`：浏览器多标签栏合并到桌面窗口标题栏；
- `794d0e3`：起始页支持网址和关键词搜索，中国大陆使用 Bing，其他已识别地区使用 Google；
- `2dd61d8`：新增 root 本地交互式管理员密码恢复，默认保留 TOTP、撤销旧 Session，不增加公网恢复 API；
- `7986ced`：统一版本字段并准备 KPanel `0.55.0`。

正式源码、标签和通过 CI 的提交均为：

```text
7986cedb63dad50b13f10103e7bbdc6f45a82527
```

## 发布前验证

在独立候选工作树与隔离 Linux/Docker 目录执行完整 L3：

- 生态策略、版本一致性和工作流 YAML 静态解析通过；
- `go test ./...`、`go vet ./...` 通过；
- `go test -race ./internal/panel ./internal/auth ./internal/dockerx` 通过；
- `govulncheck` 无可达漏洞，`npm audit --audit-level=high` 为 0；
- Trivy 源码和最终镜像扫描通过；
- Linux amd64/arm64 Panel、Agent、Node、kpctl 构建通过；
- 前端 73 个测试文件、507 项测试通过；
- TypeScript、1730 条国际化短语和生产构建通过；
- `app_conf_lifecycle=pass`；
- 候选镜像 `image_e2e=pass`；
- 候选镜像真实交互式密码恢复 E2E 通过：旧密码失效、新密码登录成功、数据哨兵保留。

远端门禁：

- 候选 CI：<https://github.com/kejilion/KPanel/actions/runs/31295543976>，成功；
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/31295684271>，成功；
- Release：<https://github.com/kejilion/KPanel/actions/runs/31295856876>，成功。

## 发布产物

GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.55.0>

附件已核对：

- `kejilion-agent-linux-amd64`
- `kejilion-agent-linux-arm64`
- `kejilion-panel-deploy-0.55.0.tar.gz`
- `SHA256SUMS`

Docker Hub：`docker.io/kjlion/kejilion-panel`

```text
0.55.0 / latest index:
sha256:849e157ecf39ec7cba6da8c854c51cec5c6618e8bd86de0457712e6e8f0b8ba7

linux/amd64:
sha256:20f8f4e2d22bff2731a0f41cfdb619bad011620e46692825f09702fe51ab5057

linux/arm64:
sha256:49f13b8e3e18d37a5b6b80de3bd15aad1f52b61904ebf86a45970242dd53a81b
```

版本镜像与 `latest` 索引摘要一致。重新拉取公开版本镜像后，通用镜像 E2E 与真实交互式密码恢复 E2E 均通过。

`packaging/kejilion-app/kpanel.conf` 未变更，与 `kejilion/apps@1f2740666a55ccbb3749ce83168e073c1ea08431` 一致，无需应用市场提交。

## 生产部署与验收

部署前生产版本为 `0.54.0`，容器 healthy、0 重启，Agent active、0 重启。通过标准应用市场更新入口部署：

```text
KJ_APP_NONINTERACTIVE=1 KJ_APP_ACTION=update /home/docker/kpanel/bin/kejilion.sh app kpanel
```

一致性备份：

```text
/root/kpanel-backups/v0.55.0-preupgrade-20260809T052241Z
```

归档校验：

```text
4d70a7dc410ff0c8924c7bfda0951bc590e98d000b14b5370d8eb75e89357a6a  kpanel-home.tar.gz
```

上线后验证：

- `/api/v1/health`：`status=ok`、`version=0.55.0`、`protocolVersion=v1alpha1`；
- Panel：running、healthy、0 重启，镜像索引为 `sha256:849e157e...b8ba7`；
- Agent：active、0 重启、`NeedDaemonReload=no`；
- `panel-state.json` JSON 校验通过；
- `ai.db` SQLite `PRAGMA integrity_check` 返回 `ok`；
- 近 10 分钟容器日志无 `panic`、`fatal` 或 `error`；
- 公开首页返回 HTTP 200；
- 2 分钟 60 次健康请求全部成功，版本始终为 `0.55.0`；
- 真实浏览器进入 `/login?redirect=/overview`，标题为“登录 · KPanel”；“忘记密码？”与展开后的恢复说明均可见，页面控制台无错误或警告。

未使用生产管理员凭据，因此未在生产账户中进入桌面工作区；桌面功能由冻结候选的全量测试、原任务视觉验收、正式镜像 E2E 和公开页面加载共同覆盖。

## 回滚点

源码回滚：`v0.54.0`。  
镜像回滚：

```text
docker.io/kjlion/kejilion-panel@sha256:6441741237a74a916902e0e2328e2ee29ded8bcdc935f073c140cd5bd26d93d5
```

普通代码回滚可把 Compose 镜像固定到上述摘要后重建 Panel。只有数据或配置同时需要回退时，才停止 Panel、校验备份 SHA-256，并恢复 `kpanel-home.tar.gz`。已经执行的管理员密码恢复属于兼容状态变更，代码回滚不会恢复旧密码或已撤销 Session。

## 遗留风险

- 目标网站仍可能通过 CSP 或 `X-Frame-Options` 拒绝 iframe；界面保留“用系统浏览器打开”回退；
- 搜索引擎未来若调整 iframe 策略，同样使用系统浏览器回退；
- 生产管理员登录后的桌面交互未在本次发布任务中直接操作，避免读取或传输生产凭据。
