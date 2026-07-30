# KPanel v0.31.0 发布验收

## 范围

- 文件管理增加 Windows 式双击、`Ctrl/Cmd`、`Shift`、复制、剪切和粘贴交互。
- 代码编辑器按需加载 CodeMirror 6 和语言解析器，大文件自动使用轻量模式。
- 修复 systemd 沙箱导致 `/home` 上传和新建目录误报只读。
- 文件传输增加并发、空闲及硬超时，目录读取增加服务端搜索和稳定分页。
- 左侧分类增加弱网预取、即时加载反馈和瞬时失败重试。
- 桌面左侧栏支持收起、展开及偏好记忆，移动端保持完整抽屉导航。

## 自动验收

- [x] Linux Go 1.26.5 环境全量测试与 vet。
- [x] Web 全量测试、类型检查和生产构建。
- [x] 生态规则、部署安装安全测试和应用生命周期测试。
- [x] Linux amd64/arm64 的 `paneld`、`kejilion-agent` 和 `kpctl` 交叉构建。
- [x] 已知漏洞扫描和 Node 高危依赖审计。
- [x] Release L3 镜像构建、运行时健康检查和镜像契约检查。
- [x] 主线 CI。
- [x] Release 工作流、多架构镜像、GitHub Release 及 Docker Hub 摘要。

## 发布证据

- 发布提交：`e829f6a5154a4e21ff53547cda4a3790691b353c`
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30575188641>
- Release：<https://github.com/kejilion/KPanel/actions/runs/30575479951>
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.31.0>
- Docker Hub `0.31.0` / `latest`：
  `sha256:1d9a73d105a11059e6b909433a1927fe6400ca4198c7211faa0786205ef45c0a`
- 平台清单：`linux/amd64`、`linux/arm64`

## 性能、安全与兼容结论

- 文件流使用固定缓冲，不把完整文件载入 Panel 或 Agent 内存；上传并发 2、下载并发 4，
  45 秒无数据即中断，单次请求硬上限 2 小时。
- 目录单页最多 500 项、服务端最多扫描 20,000 项；复制预算按整批累计
  10,000 个条目或 10 GiB。
- 文件根继续由 Go `os.Root` 固定在 `/home`，KPanel 目录、路径穿越和符号链接逃逸规则不变。
- CodeMirror、语言解析器和分类页面均按需加载，不进入初始页面主包。
- 本次不修改 `kejilion.sh`，不改变应用、网站、Docker、体检、集群和环境管理协议。

## 人工验收

- [x] 文件列表双击、右键、`Ctrl/Cmd` 与 `Shift` 选择逻辑。
- [x] 代码高亮懒加载和大文件轻量模式。
- [x] `/home` 上传、新建目录及写入错误路径复核。
- [x] 弱网分类切换延迟模拟、加载反馈和失败重试。
- [x] 左侧栏深色、浅色、收起、展开和刷新状态记忆。

## 回滚

- 发布前回滚点：`v0.30.0` / `0b708dc`。
- 回滚面板镜像与 Agent 不会迁移或删除 `/home` 用户文件。
- 浏览器中的剪贴板、选择和侧栏偏好仅为本地状态，不产生服务端迁移。
