# 构建、发布与部署

## 部署边界

Kejilion Panel 使用两个独立进程：

- `paneld` 以非 root Docker 容器运行，只绑定 `127.0.0.1:18443`。
- `kejilion-agent` 以受限 systemd 服务运行，只接受本机 Unix Socket 上的类型化请求。

安装器只管理 `/etc/kejilion-panel`、`/opt/kejilion-panel`、
`/var/lib/kejilion-panel`、`/run/kejilion-panel`、
`/usr/local/libexec/kejilion-agent` 和对应 systemd unit。安装器不会执行或
修改 `kejilion.sh`，也不会改动 `/home/web`、现有 Nginx 配置、防火墙和站点。

## 发布产物

版本发布应包含：

- `docker.io/<owner>/kejilion-panel:<version>` 的 `linux/amd64`、`linux/arm64`
  多架构镜像；
- `kejilion-agent-linux-amd64`；
- `kejilion-agent-linux-arm64`；
- `kejilion-panel-deploy-<version>.tar.gz`；
- 上述文件的 `SHA256SUMS`；
- 镜像 manifest digest。生产部署只使用
  `docker.io/<owner>/kejilion-panel@sha256:<digest>`，不使用可漂移标签。

仓库的 `Release` 工作流仅接受精确的 `v<semver>` 标签。启用前配置：

- Repository variable `DOCKERHUB_IMAGE`：`owner/repository`；
- Repository variable `DOCKERHUB_USERNAME`：Docker Hub 用户名；
- Repository secret `DOCKERHUB_TOKEN`：仅具备目标仓库写权限的访问令牌。

工作流会先执行前后端验证，再构建双架构 Agent、带 SBOM/Provenance 的双架构
镜像，并把固定镜像 digest 写入 GitHub Release。生产部署使用 Release 中的
digest 与校验和，不直接使用 `latest`。

本地验证和交叉编译：

```sh
make test
make build-linux
sha256sum dist/linux-amd64/kejilion-agent dist/linux-arm64/kejilion-agent
```

推送 Docker Hub：

```sh
VERSION=0.1.0
IMAGE=docker.io/<owner>/kejilion-panel

docker login
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg "VERSION=$VERSION" \
  --provenance=mode=max \
  --sbom=true \
  --tag "$IMAGE:$VERSION" \
  --tag "$IMAGE:latest" \
  --push .

docker buildx imagetools inspect "$IMAGE:$VERSION"
```

发布前应把输出的 manifest digest 记录到发布说明；部署时必须使用该 digest。

## 宿主机预检

先在目标机执行只读预检。该命令不会连接 Docker Socket，避免意外启动已停止
的 Docker：

```sh
./deploy/preflight.sh \
  --public-url https://panel.example.com \
  --port 18443
```

预检告警不一定阻止部署。目标机尚无 `/home/web/certs` 时，安全登录和主机监控仍可工作；
为避免把目录故障误报为空列表，网站发现会保持不可用，目录恢复后自动恢复。失败项必须处理后再部署。

## 安装

根据目标机架构选择 Agent，并先在目标机核对摘要：

```sh
sha256sum kejilion-agent

sudo ./deploy/install.sh \
  --agent-binary ./kejilion-agent \
  --agent-sha256 <agent-sha256> \
  --image docker.io/<owner>/kejilion-panel@sha256:<manifest-digest> \
  --public-url https://panel.example.com \
  --port 18443 \
  --dry-run
```

确认 dry-run 后去掉 `--dry-run`。安装成功后，一次性初始化 Token 仅保存在：

```text
/var/lib/kejilion-panel/panel/bootstrap.token
```

Token 不会写入日志或安装器输出。首次初始化成功后文件会被删除。

## HTTPS 入口

Panel 容器只监听回环地址。需要在现有 HTTPS 入口中把 Panel 域名反向代理到
`http://127.0.0.1:18443`，并透传 `Host`、`X-Real-IP` 和
`X-Forwarded-Proto`。反向代理来源必须显式加入 Panel 的可信代理 CIDR；不要
信任整个公网或所有私网。

默认 Compose 使用专用内部网段 `172.29.255.240/28`，并只信任 loopback 与
该网段。如果它与目标机已有网段冲突，应在首次启动前同时修改
`KEJILION_PANEL_NETWORK_SUBNET` 和
`KEJILION_PANEL_TRUSTED_PROXY_CIDRS`，保持两者一致。

反向代理配置属于目标机业务配置，安装器不会自动写入。上线时应单独备份、新增
独立域名配置、执行 `nginx -t`，成功后才 reload；验证失败时不得 reload。

## 验收

```sh
systemctl is-active kejilion-agent
docker compose --env-file /opt/kejilion-panel/.env \
  -f /opt/kejilion-panel/compose.yml ps
curl --fail --silent http://127.0.0.1:18443/api/v1/health
curl --fail --silent --show-error https://panel.example.com/api/v1/health
```

还需人工确认：

- 登录、注销、失效 Session 和 CSRF 拒绝；
- `kejilion.sh` 已有站点与容器只读发现正常；
- Web 创建的测试站点产物可被脚本侧列表识别；
- 脚本侧新增测试站点后，刷新 Web 能显示实际配置；
- 未识别容器保持只读；
- Agent 离线时 Web 降级且所有宿主机写操作禁用。

## 回滚

安装器会把被替换的 Panel 文件备份到
`/var/backups/kejilion-panel/<UTC 时间>`。回滚只恢复 Panel 自身二进制、
unit、Compose 和 `.env`，然后重启 Agent 与 Panel 容器；不得回滚或覆盖
`/home/web`、`kejilion.sh`、站点、数据库、证书和其他容器。
