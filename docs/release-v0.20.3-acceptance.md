# KPanel v0.20.3 问题修复与发布验收清单

状态说明：`已实现` 仅代表代码完成；只有验证项全部通过并核对线上产物后才能标记为 `已上线`。

| # | 用户反馈 | 归属判断 | 修复/验收要求 | 当前状态 |
| --- | --- | --- | --- | --- |
| 1 | 概览发行版图标显示字母占位 | KPanel 前端 | Ubuntu、Debian、CentOS、Rocky、Alma、Fedora、Alpine、Arch、openSUSE 使用真实 SVG 品牌图标；未知系统使用 Linux 图标 | 已实现，Rocky Linux 品牌图标已视觉复核 |
| 2 | 概览国旗显示异常 | KPanel 前端 | 使用随面板发布的本地轻量 SVG 国旗资源，不依赖 emoji 字体或远程图片源 | 已实现，本地香港国旗已视觉复核 |
| 3 | DNS 设置不可用 | `kejilion.sh` 协议 + Agent 权限 | 使用 `KJ_DNS_NONINTERACTIVE=1`；升级安装时同步受信脚本；Agent 增加静态 DNS 所需 `CAP_LINUX_IMMUTABLE`；覆盖 resolved 与 static 两种模式 | 生命周期验收通过，待目标机器升级后实机复核 |
| 4 | 应用提示更新脚本、接受许可后仍无法安装 | `kejilion.sh` 协议 + KPanel 升级链路 | KPanel 安装/更新同步固定提交和 SHA-256 的脚本，只从本机可信且已授权脚本继承许可，不静默代替用户授权 | 安装、更新、回滚与卸载生命周期验收通过 |
| 5 | 脚本安装的内置/第三方应用不能更新、卸载 | 主要由新版 `kejilion.sh` 解决 | 使用脚本原生应用状态、更新、卸载入口；面板不以“非 KPanel 创建”为由禁用 | 新版脚本已提供协议，待双端验收 |
| 6 | 应用安装被“已有任务运行”永久阻塞 | KPanel 后端 | 自动核对 systemd 单元；已消失的 queued/running 任务标记为中断并释放互斥锁；返回中文可操作错误 | 已实现并补单元测试 |
| 7 | 应用域名反代、第三方域名显示和打开方式不正确 | `kejilion.sh` 建站协议 + KPanel 前端 | 域名绑定复用脚本反代入口；应用详情展示匹配域名；打开应用优先有效域名和 HTTPS，否则回退 IP+端口 | 现有实现已复核，4 项域名优先测试通过 |
| 8 | 系统更新实际成功却显示“上次执行失败” | KPanel 后端 | systemd 单元已回收时，`Result=success` 且 `ExecMainStatus=0` 必须判定成功 | 已实现并补回归测试 |
| 9 | 无 LDNMP 环境时不能创建站点、IP+端口反代失败 | `kejilion.sh` 建站协议 + KPanel 任务链 | 站点入口保持可点击；WordPress/反代由脚本按需准备 LDNMP/Nginx；域名和上游由面板校验后传入；证书失败可在终端重试或导入 | 已实现，Linux 单元/PTY 测试通过，待线上实机验收 |
| 10 | 站点删除只显示 `Validation failed` | KPanel 前端/API 适配 | 删除前刷新缺失或陈旧的 `resourceVersion`；优先显示后端字段级中文错误；保留失败回滚 | 已实现并补 API 回归测试 |
| 11 | 体检提示协议不可用 | 主要由新版 `kejilion.sh` 解决 | 使用固定体检目录和 `KJ_TEST_NONINTERACTIVE=1`，升级后同步脚本；禁止自定义 Shell | 新版脚本协议和升级同步生命周期验收通过 |
| 12 | 体检卡片墙体验差 | KPanel 前端 | 桌面端改为左侧命令选择、右侧实时终端；移动端纵向排列；保留分类、历史与任务状态 | 已实现，左右布局与终端区域已视觉复核 |
| 13 | Docker 管理排版不统一、默认页不合适 | KPanel 前端 | 默认进入容器页；统一导航、工具栏、表格列宽与操作按钮分组，避免按钮散乱换行 | 已实现，默认容器页与统一布局已视觉复核 |
| 14 | KPanel 更新后宿主脚本能力仍旧、缺少回滚 | KPanel 安装/更新链路 | 镜像内同时携带 Agent 与固定摘要脚本；安装、旧版升级、失败回滚、权限和许可继承均纳入生命周期测试 | CI 隔离容器生命周期验收通过 |
| 15 | 建站调用脚本但看不到过程和具体失败原因 | KPanel Agent/API/前端 + `kejilion.sh` | 面板先收集域名和固定参数；后台 PTY 实时输出；终端支持后续输入；安全进度时间线保留具体脚本失败阶段；终端受登录、CSRF 和输入长度限制保护 | 已实现，Linux PTY、Go 和前端测试通过 |
| 16 | GitHub CI/Release、Docker Hub 和线上产物未闭环 | 发布链路 | 全量 Go/前端测试、生命周期测试、双架构镜像、OCI 标签、镜像内脚本摘要、Docker Hub `latest`、GitHub Release、线上配置同步逐项验证 | v0.20.3 已发布；目标机器仍运行 0.20.2，待用户升级后复核 |

## 发布门槛

- [x] `gofmt`、`git diff --check`
- [x] `go test ./...`
- [x] `go vet ./...`
- [x] 前端类型检查、34 项单元测试、生产构建
- [x] Linux `go test ./...`、`go vet ./...` 和 PTY 测试
- [x] `packaging/tests/app-conf-lifecycle.sh`
- [x] Linux amd64/arm64 构建
- [x] 镜像非 root、健康检查、版本/提交/脚本 OCI 标签
- [x] 镜像内 `kejilion.sh` SHA-256 与固定值一致
- [x] GitHub CI 成功
- [x] GitHub Release 成功
- [x] Docker Hub 版本标签与 `latest` 摘要一致
- [x] `kejilion/apps` 中 `kpanel.conf` 与发布版本一致
- [ ] 线上升级后实机复核 DNS、应用、站点、体检和 Docker 页面

## 发布证据

- KPanel 提交：`1f1227c49d54ac197ee4e3cb9d60c8b7bf008af9`
- `kejilion.sh` 提交：`5361f8f9ba7133e8ce64f32adfa27b22d6247484`
- 脚本 SHA-256：`40289bf18f1550afa4cd2cad58c2d1f515b99189692b80ffffb0a7e3c540290b`
- Release 工作流：`30278137621`
- Docker `0.20.3` / `latest` 摘要：`sha256:9cd64a516984f51df3580976638db3d30504cef721d11075b542a147b05959ca`
- `kejilion/apps` 同步提交：`8abee67eb9435b9e6cf860fbd280dc787995cceb`
- 目标机器公开健康接口在发布后仍报告 `0.20.2`；SSH 公钥认证不可用，因此未冒充已完成实机升级。
