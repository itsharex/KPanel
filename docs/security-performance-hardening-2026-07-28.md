# KPanel 安全与性能加固报告（2026-07-28）

## 1. 范围与结论

- 基线提交：`0e4b5da`
- 验证分支：`fix/security-performance-hardening`
- 覆盖范围：Go/Node 依赖、最终容器镜像、Dockerfile、HTTP 安全边界、认证峰值内存、Panel-Agent 读取链路、Docker 容器采集、静态资源传输。
- 不在本次范围：线上部署、业务数据写入、第三方应用自身漏洞、宿主机发行版全部软件包和内核 CVE。

结论：未发现 KPanel 调用路径、生产依赖或最终镜像中的已知 High/Critical 漏洞。需要持续关注 Node.js 2026-07 安全发布，以及旧宿主机上的 Docker/containerd/runc 版本；本次改动不要求停服迁移，不改变 `kejilion.sh`、`k fd`、应用管理和交互终端业务协议。

## 2. 已验证安全结果

| 检查 | 工具/版本 | 结果 |
|---|---|---|
| Go 调用路径 | `govulncheck v1.6.0` + 当前漏洞库 | 0 个可调用漏洞 |
| Node 生产及开发依赖 | `npm audit` | 0 个漏洞 |
| 最终 scratch 镜像 | Trivy，High/Critical，漏洞与 Secret | 0 |
| 源码和 Dockerfile | Trivy，High/Critical，漏洞、Secret、Misconfiguration | 0 |
| 静态分析 | `gosec v2.28.0` | 95 个启发式提示；16 个标为 High，逐项人工复核后未确认可利用漏洞 |
| 容器运行约束 | 隔离启动验证 | 非 root、只读根文件系统、全部 capability 删除、`no-new-privileges`、256 MiB、1 CPU、128 PIDs 可正常启动 |

`gosec` 启发式提示已由基线的 101 个/22 个 High 降至 95 个/16 个 High。

`govulncheck` 另报告依赖模块中存在 1 个未调用条目：`x/crypto/openpgp` 已停止维护。KPanel 不导入、不调用该包，因此不构成当前可达漏洞。

`gosec` High 提示的复核分类：

- `G704`：Panel Agent HTTP 客户端固定通过 Unix Socket 拨号，不能转向外部网络；公开路由另有固定 allowlist。
- `G703`：配置和凭据路径只接受本机绝对路径；环境备份 ID 使用固定正则并限定到备份目录。
- `G118`：应用安装、Docker 维护和公网信息刷新是明确设计为脱离浏览器请求继续运行的后台任务。
- `G122`：Docker 备份在 `Lstat` 后打开文件，并用 `file.Stat + os.SameFile` 复核 inode，替换竞态会被拒绝。
- `G115`：已补充文件大小、块大小、文件描述符、备份元数据和应用端口边界检查；剩余提示均位于明确边界后的整数转换。

## 3. 2025–2026 高危漏洞评估

### 需要持续关注

1. Node.js 已预告 2026-07-27 前后为 24.x 发布最高 High 级别安全修复；截至本次验收，官方最新 LTS 页面仍为 `24.18.0`。KPanel 已将构建器升级并固定到 `24.18.0` 镜像摘要。Node 只参与前端构建，不进入 scratch 运行镜像；新安全版发布后仍应及时重新固定摘要。
2. 2025–2026 年 containerd/runc 出现过 Critical/High 容器逃逸、宿主文件访问和 CRI 检查点恢复漏洞。KPanel 运行环境必须使用发行版已修补版本，不能只更新 KPanel 镜像。
3. Trivy 在 2026-03 曾发生供应链事件。扫描工具必须固定安全版本/镜像摘要，不使用 `latest` 或受影响的 `0.69.4–0.69.6`。

### 本次实机验证环境

- Docker Engine `29.6.2`
- containerd `2.2.6`
- runc `1.3.6`
- Linux `6.12.96+deb13-amd64`

上述 Docker 版本已包含 2026-07 官方安全修复；runc 版本高于 2025 年相关漏洞的修复版本 `1.3.3`。

官方依据：

- Go 发布记录：<https://go.dev/doc/devel/release>
- Node.js 2026-07 安全预告：<https://nodejs.org/en/blog/vulnerability/july-2026-security-releases>
- Node.js 24.18.0：<https://nodejs.org/en/blog/release/v24.18.0>
- Docker Engine 29：<https://docs.docker.com/engine/release-notes/29/>
- containerd advisories：<https://github.com/containerd/containerd/security/advisories>
- runc GHSA-9493-h29p-rfm2：<https://github.com/opencontainers/runc/security/advisories/GHSA-9493-h29p-rfm2>
- Trivy 供应链通告：<https://github.com/aquasecurity/trivy/security/advisories/GHSA-69fq-xp46-6x23>

## 4. 本次优化

### 认证内存

- 保持 Argon2id 的 64 MiB、3 次迭代和最多 4 线程参数，不降低密码安全强度。
- 默认并发密码哈希从 2 收紧为 1；并发请求快速返回 429，不把容器推到 OOM 边缘。

8 个并发正确密码登录的隔离测试：

| 指标 | 优化前基线 | 优化后 |
|---|---:|---:|
| 256 MiB cgroup 峰值 | 达到 256 MiB 上限 | 151,560,192 B（144.5 MiB） |
| `memory.events.max` | 增加 242 | 0 |
| OOM / OOM kill | 0 / 0 | 0 / 0 |
| 返回 | 2 个执行、其余 429 | 1 个 200、7 个 429 |

峰值下降约 43.5%，仍保留约 111 MiB cgroup 余量。

### Panel-Agent 与 Docker 采集

- 相同路径、相同查询参数的并发只读请求只执行一次；请求完成后不缓存，避免状态过期及 `resourceVersion` 冲突。
- Docker 容器详情从逐个串行 `inspect` 改为最多 4 路有界并发。
- 单元回归验证：12 个同时读取者只触发 1 次 Agent 读取；12 个容器详情读取的最大并发为 4。

### 静态/API 传输

- Vite 构建阶段预生成 `.gz`，Panel 直接发送预压缩文件，不在请求时消耗 CPU。
- `/assets/` 指纹文件使用 `Cache-Control: public, max-age=31536000, immutable`。
- 大于等于 1 KiB 的 Agent JSON 使用快速 gzip；下载、Range、HEAD 和不可压缩内容保持原行为。

隔离 HTTP 验证：

| 资源 | 原始 | gzip | 减少 |
|---|---:|---:|---:|
| 主入口 JS | 144,073 B | 54,063 B | 62.5% |
| 概览页 JS | 约 478.8 KB | 约 89.3 KB | 81.3% |

### HTTP 与供应链

- HTTPS 或可信本机 `k fd` 反代链路增加 HSTS。
- 增加 `Cross-Origin-Opener-Policy` 与 `Cross-Origin-Resource-Policy`。
- 保留 CSP、Host 校验、Origin/CSRF、Secure/SameSite Cookie 和 Agent Unix Socket 边界。
- Node 构建器升级到 `24.18.0` 并固定镜像摘要。
- CI 增加固定版本 `govulncheck` 和 `npm audit --audit-level=high`。

## 5. 业务兼容与回滚

- `k fd`：可信本机反代的 `Host + X-Forwarded-Proto: https` 继续有效，Secure Cookie 和 Origin 校验回归通过；隔离实测绑定域名返回 200 并携带 HSTS，非 HTTPS 转发返回 421。
- `kejilion.sh`、应用安装/更新/卸载、站点、环境管理、后台任务和交互 Shell 协议未修改。
- 无响应缓存；只合并正在进行的相同只读请求，不会引入陈旧状态。
- Docker `inspect` 并发固定最多 4，不会无界占用 Docker daemon。
- 回滚点：`0e4b5da`；回滚只需恢复上一镜像/提交，业务数据目录无需改变。

## 6. 发布前门槛

1. `go test ./...`
2. `go test -race ./internal/panel ./internal/auth ./internal/dockerx`
3. `npm test`、`npm run build`、`npm audit`
4. `govulncheck ./...`
5. 使用摘要固定的 Trivy 扫描源码和最终镜像
6. scratch 镜像隔离启动、HTTP 头、Host/路径/方法拒绝、gzip 和缓存验证
7. `k fd` 可信 HTTPS 反代回归
8. 正式发布前再执行项目 `verify-release`
