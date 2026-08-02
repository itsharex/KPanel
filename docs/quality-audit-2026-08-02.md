# KPanel 质量审计记录（2026-08-02）

## 范围

- 审计基线：`fc89ee94d03a324ac325516c593c97cc9b9a1435`（`v0.38.2`）
- 回滚点：`fc89ee94d03a324ac325516c593c97cc9b9a1435`
- 验收机：Debian 13、Docker `29.6.2`、containerd `2.2.6`、runc `1.3.6`、
  Linux `6.12.96+deb13-amd64`
- 本轮只修改规范、自动门禁和版本元数据，不修改业务行为，不提交、不推送、不部署。

本报告记录当前证据，避免后续 Codex 把
[`runtime-performance-baseline.md`](runtime-performance-baseline.md) 中 `v0.24.0` 的历史阻断项
误判为当前状态。旧报告仍作为相同机器和方法下的比较基线。

## 结论

当前架构仍符合 Panel/Agent 权限分离、真实状态回读、固定动作 API、非 root 容器和有界资源原则。
未发现可达 High/Critical 漏洞或当前性能阻断项。快速迭代提高了回归风险，但现有业务基线健康；
本轮发现的主要缺口是“规范要求未全部自动阻断”，已通过统一脚本、CI 和 Release 门禁补齐。

## 自动测试与构建

在隔离副本执行 `make verify-release`：

- Go 全量测试通过；
- 前端 `32` 个测试文件、`214` 个测试通过；
- TypeScript 类型检查、`1411` 条文案/`14` 个懒加载语言目录和生产构建通过；
- `internal/panel`、`internal/auth`、`internal/dockerx` race 通过；
- `go vet ./...`、Shell 语法、版本元数据一致性、隔离安装安全测试通过；
- amd64/arm64 的 Panel、Agent、轻量节点和 `kpctl` 构建通过；
- 候选镜像构建和健康检查通过。

候选镜像在 `256 MiB / 1 CPU / 128 PID`、非 root、只读根、`cap-drop ALL`、
`no-new-privileges` 条件下健康运行，实际 inspect 结果为：

```text
user=65532:65532
memory=268435456
nano_cpus=1000000000
pids=128
readonly=true
cap_drop=["ALL"]
security=["no-new-privileges"]
```

## 性能复核

同一验收机、同一资源限制下：

| 指标 | 结果 | 判定 |
| --- | ---: | --- |
| 冷启动到健康 | `410 ms` | 通过，预算 `≤ 2 s` |
| Panel 预热后空闲 RSS | `27.04 MiB` | 通过，预算 `≤ 32 MiB` |
| `/api/v1/health`，20k/c10 | `4703.5 req/s`，P95 `4.710 ms`，0 错误 | 通过 |
| `/api/v1/auth/session`，10k/c10 | `4763.9 req/s`，P95 `4.617 ms`，0 错误 | 通过 |
| `/`，10k/c20 | P95 `17.854 ms` | 通过；较旧单次基线 `13.349 ms` 高 `33.7%`，需同条件复跑后才能定性回归 |
| 登录突发峰值 | 约 `137.5 MiB`，无 OOM | 通过；1 路执行、其余按设计 `429` |
| Agent 空闲/120 次混合读取峰值/结束后 | `16.7 / 27.1 / 21.8 MiB`，FD `10` | 通过 |
| 8 MiB 压缩/解压 | `31.52 / 28.85 ms` | 通过 |
| Cluster Noise summary | `1.299 ms/op` | 通过 |
| 7 天历史监控查询 | `86.82 ms/op`、约 `7.87 MiB/op`、`80,758 allocs/op` | 通过但需持续关注分配量 |

主入口 gzip `27.93 KiB`，低于 `70 KiB` 预算；最大业务懒加载路由为交互终端
`89.88 KiB`，低于 `120 KiB` 预算。

## 安全复核

- `govulncheck v1.6.0`：0 个可达漏洞、0 个导入包漏洞；依赖模块元数据有 1 条未调用记录。
- `npm audit --audit-level=high`：0 个漏洞。
- 固定摘要 Trivy `v0.72.0`：源码依赖、Secret、Dockerfile 配置和最终 Go 二进制均为
  0 个 High/Critical。
- Go 1.17+ 源码扫描跳过只保存历史校验和的 `go.sum`，仍扫描 `go.mod`，并由
  `govulncheck` 与最终二进制扫描交叉验证；没有添加漏洞 ID suppress。
- `gosec v2.28` 共 143 条启发式提示、20 条 High；结合固定动作入口、Unix Socket、路径边界、
  后台任务语义逐项抽查，未确认可利用 High。数量高于旧基线，后续应按规则分类而不是只比较数量。
- Host、Origin、CSRF、Cookie、方法限制和安全响应头测试存在并通过；未发现
  `InsecureSkipVerify`、通配 CORS 或 Panel 挂载 Docker Socket。

当前运行时版本已高于本轮核对的相关修复版本。后续复核只采用官方来源：

- Moby AuthZ：<https://github.com/advisories/GHSA-x744-4wpc-v9h2>
- containerd Releases：<https://github.com/containerd/containerd/releases>
- runc Releases：<https://github.com/opencontainers/runc/releases>
- Go Security：<https://go.dev/doc/security/>
- Trivy Go 扫描：<https://trivy.dev/docs/latest/guide/coverage/language/golang/>

## 已补门禁

- `scripts/check-version-consistency.sh`：阻止 `VERSION`、Go 和 Web 版本漂移。
- `scripts/security-scan.sh`：固定摘要执行源码和最终镜像 Trivy。
- `scripts/verify-deploy.sh`：隔离运行安装安全测试，不碰撞已部署 KPanel。
- `make verify-release`：统一覆盖全量测试、race、`govulncheck`、`npm audit`、Trivy、
  Linux 构建和最终镜像扫描。
- CI 增加核心 race 和源码扫描；Release 增加最终镜像扫描及生产资源限制断言。
- `.codex-workflows/quality-audit-kpanel.workflow.yaml`：为后续 Codex 提供可重放审计流程。

## 剩余风险

1. `internal/hostpty` 当前没有独立单元测试；交互终端已有上层测试和实机验证，但 PTY 生命周期、
   断连、重连和输入边界仍应补专门测试。
2. 仓库暂未引入 fuzz 测试；归档、路径、协议解析和终端输入适合作为后续优先候选。
3. 7 天历史监控查询的分配量较高；目前未超延迟或内存预算，增长数据量或并发前应复跑基准。
4. 首页 P95 的单次结果高于旧基线，但绝对值仍低；相同镜像至少复跑 3 轮后再决定是否优化。

这些风险不阻断当前质量门禁，但不得在发布说明中写成“零风险”。
