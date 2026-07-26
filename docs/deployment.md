# 构建、发布与部署

## 部署边界

KPanel 使用两个独立进程：

- `paneld` 以非 root Docker 容器运行，只接入专用 `internal` 网络，不发布宿主端口。
- `kejilion-agent` 以受限 systemd 服务运行，只接受本机 Unix Socket 上的类型化请求。

宿主机必须使用 systemd。当前发行版代码路径覆盖 Debian/Ubuntu、
RHEL/Fedora、Arch/Manjaro 和 openSUSE/SLES；具体实机验收层级见
[宿主机系统兼容矩阵](platform-support.md)。Alpine/OpenRC 尚不属于正式部署目标。

安装器只管理 `/etc/kejilion-panel`、`/opt/kejilion-panel`、
`/var/lib/kejilion-panel`、`/run/kejilion-panel`、
`/usr/local/libexec/kejilion-agent`、对应 systemd unit、专用
`kejilion-panel` 容器与 `kejilion-panel-internal` 网络。安装器还会把固定
digest 的 Panel 镜像拉入本机缓存。安装器不会执行或修改 `kejilion.sh`，也不会
改动 `/home/web`、现有 Nginx 配置、防火墙和站点。

v0.1 安装器只支持全新安装。发现任何既有 Panel 文件、同名容器或同名网络时
会拒绝继续，不会把未知资源当作可升级对象。后续版本必须在具备事务化升级和自动
回滚后再开放原地升级。

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
VERSION=0.15.0
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
  --network-subnet 172.29.255.240/28
```

预检不会调用 `docker info`、不会连接 Docker Socket，也不会启动 Docker。
Docker 服务必须由运维人员在评估现有容器后提前启动。预检会拒绝重叠路由、
非专用 Agent 组、既有 Panel 资源，以及缺失或不可读的 `/home/web`
根目录。目标机尚无 `/home/web/certs` 等子目录时，安全登录和主机监控仍可
工作；为避免把目录故障误报为空列表，相关网站发现能力会保持不可用，目录恢复
后自动恢复。失败项必须处理后再部署。

## 安装

根据目标机架构选择 Agent，并先在目标机核对摘要：

```sh
sha256sum kejilion-agent

sudo ./deploy/install.sh \
  --agent-binary ./kejilion-agent \
  --agent-sha256 <agent-sha256> \
  --image docker.io/<owner>/kejilion-panel@sha256:<manifest-digest> \
  --public-url https://panel.example.com \
  --network-subnet 172.29.255.240/28 \
  --dry-run
```

`--network-subnet` 只接受对齐的 RFC1918 IPv4 `/28`。安装器会用同一 CIDR
自动生成网关（网段基址加 1）、Panel 私网地址（网段基址加 2）和可信代理范围，
不提供可独立放宽的参数。`--dry-run` 会在连接 Docker Socket 前返回；确认后
去掉 `--dry-run`。正式安装只连接
`unix:///var/run/docker.sock`，并拒绝继承 `DOCKER_HOST` 或
`DOCKER_CONTEXT`。

安装成功后，一次性初始化 Token 仅保存在：

```text
/var/lib/kejilion-panel/panel/bootstrap.token
```

Token 不会写入日志或安装器输出。首次初始化成功后文件会被删除。

## HTTPS 入口

Panel 容器不发布宿主端口。需要在宿主机或 host-network Nginx 中把 Panel
域名反向代理到固定私网地址，例如默认网段对应
`http://172.29.255.242:8080`，并设置 `Host`、`X-Real-IP` 和
`X-Forwarded-Proto`。反向代理必须覆盖设置 `X-Real-IP`，不能沿用客户端提交
的同名请求头。反向代理来源必须显式加入 Panel 的可信代理 CIDR；不要信任整个
公网或所有私网。

默认 Compose 使用专用内部网段 `172.29.255.240/28`，并只信任 loopback 与
该网段。如果预检发现冲突，通过 `--network-subnet` 选择另一个对齐的私网
`/28`；安装器会同时写入网络、网关、Panel 私网地址和可信代理 CIDR，不能在
安装后手工改其中一项。Nginx 必须与 Panel 同处一台宿主机或能安全路由到该
内部网段；不要把 Panel 私网地址暴露到公网路由。

反向代理配置属于目标机业务配置，安装器不会自动写入。上线时应单独备份、新增
独立域名配置、执行 `nginx -t`，成功后才 reload；验证失败时不得 reload。

## 直接 IP + 端口

测试主机可以不依赖 Nginx，使用附加 Compose 文件直接发布端口：

```sh
KEJILION_PANEL_PUBLIC_URL=http://154.36.153.9:8080
KEJILION_PANEL_SECURE_COOKIE=false
KEJILION_PANEL_BIND_ADDRESS=0.0.0.0
KEJILION_PANEL_PORT=8080

docker --host unix:///var/run/docker.sock compose \
  --project-name kejilion-panel \
  --env-file /opt/kejilion-panel/.env \
  -f /opt/kejilion-panel/compose.yml \
  -f /opt/kejilion-panel/direct-port.yml up -d
```

`KEJILION_PANEL_PUBLIC_URL` 必须与浏览器访问的来源完全一致。直接 HTTP 会禁用
Secure Cookie，仅建议用于受控测试环境。覆盖文件会把原来的专用 Panel 网络改为
可发布端口的普通 bridge；Panel 仍只加入这一张网络，避免 `kejilion.sh` 的端口访问
策略读取到多个容器 IP。正式公网环境仍建议使用 HTTPS。

直接端口部署完成后，可以使用 `kejilion.sh` 的标准反代入口：

```sh
k fd panel.example.com 127.0.0.1 8080
```

来自显式可信代理 CIDR 的请求可以使用代理传递的 HTTPS `Host` 和
`X-Forwarded-Proto`，无需把 `KEJILION_PANEL_PUBLIC_URL` 从直连地址改成域名。
该路径会自动启用 Secure Cookie，并从 `X-Real-IP` 或安全解析后的
`X-Forwarded-For` 恢复客户端地址；非可信来源不能利用这些请求头绕过 Host/Origin 校验。

## 验收

```sh
systemctl is-active kejilion-agent
sudo /usr/local/libexec/kejilion-agent healthcheck
docker --host unix:///var/run/docker.sock compose \
  --project-name kejilion-panel \
  --env-file /opt/kejilion-panel/.env \
  -f /opt/kejilion-panel/compose.yml ps
docker --host unix:///var/run/docker.sock compose \
  --project-name kejilion-panel \
  --env-file /opt/kejilion-panel/.env \
  -f /opt/kejilion-panel/compose.yml \
  exec -T panel /paneld agent-healthcheck
curl --noproxy '*' --fail --silent http://172.29.255.242:8080/api/v1/health
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

v0.1 不执行原地升级，因此没有“恢复旧版 Panel”的路径。全新安装在启动阶段
失败时，安装器会尝试停止并禁用本次 Agent，并只在 Compose project/service
标签同时匹配时停止 Panel 容器；随后复核容器运行态、Agent `ActiveState` 和
`UnitFileState`。无法确认时会输出 `CRITICAL`，此时不得重试或启动相关服务。
安装器不会自动删除数据、日志或镜像。先保留现场并检查：

```sh
journalctl -u kejilion-agent --no-pager
docker --host unix:///var/run/docker.sock logs kejilion-panel
systemctl show kejilion-agent.service -p LoadState -p FragmentPath -p DropInPaths
docker --host unix:///var/run/docker.sock inspect kejilion-panel \
  --format '{{json .Config.Labels}}'
```

只有确认 unit 的 `FragmentPath` 是
`/etc/systemd/system/kejilion-agent.service`，且容器标签同时包含
`com.docker.compose.project=kejilion-panel` 与
`com.docker.compose.service=panel` 后，才可执行以下全新安装恢复步骤：

```sh
docker --host unix:///var/run/docker.sock compose \
  --project-name kejilion-panel \
  --env-file /opt/kejilion-panel/.env \
  -f /opt/kejilion-panel/compose.yml down
systemctl disable --now kejilion-agent.service
rm -f -- /etc/systemd/system/kejilion-agent.service \
  /usr/local/libexec/kejilion-agent
rm -rf -- /etc/kejilion-panel /opt/kejilion-panel /var/lib/kejilion-panel
systemctl daemon-reload
```

`kejilion-panel` 组默认保留；只有能证明它由本次失败安装创建、没有显式成员且
没有用户以其为主 GID 时，才可单独删除。恢复完成后重新执行只读 preflight，
不得绕过 fresh-install 检查直接重试。

任何回滚都不得删除、恢复或覆盖 `/home/web`、`kejilion.sh`、站点、数据库、
证书和其他容器。生产部署前应记录 `kejilion.sh` 相关文件哈希和现有 Docker
资源清单，回滚后再次比对，证明现有业务未变化。
