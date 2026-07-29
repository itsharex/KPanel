# KPanel v0.26.1 验收记录

## 交付范围

- 修复已安装应用在停止、重启或暂停后无法再次打开详情的问题。
- Agent 对无端口容器固定返回 `ports:[]`；前端同时兼容旧 Agent 的
  `ports:null`，避免详情渲染时访问 `null.find`。
- 详情弹窗在 `running`、`exited`、`paused` 及确定性
  `restarting + ports:null` 用例中保持可打开、可关闭且不残留页面滚动锁。
- 新增真实浏览器真机生命周期工作流，并加固独立工作树发布流程。
- 本版本未修改 `packaging/kejilion-app/kpanel.conf`，不需要同步
  `kejilion/apps`。

## 发布提交

- 应用详情与端口契约修复：`1d7a705fe3e944eacc4a88e946fc995fed11e7cc`
- 版本准备：`8fd24ae`
- 发布与真机工作流加固：`7463438`、`7da8a6d`、`32fe6ab`、
  `7da6bd4`、`295d100`
- 标签提交：`295d100c44de4ad8986df3093460e0102980e18e`
- 标签：`v0.26.1`

## 自动与 L3 验收

- 候选分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30440173941>
- 主分支 CI：
  <https://github.com/kejilion/KPanel/actions/runs/30440287508>
- Release：
  <https://github.com/kejilion/KPanel/actions/runs/30440422453>
- 精确提交 `295d100c44de4ad8986df3093460e0102980e18e` 通过远端 Git bundle
  校验后在 154 真机独立目录执行 L3。
- Go 全量测试、`go vet`、生态规则、安装安全测试、Shell 应用生命周期、
  前端类型检查、14 个前端测试文件共 80 项 Vitest、生产构建全部通过。
- Panel、Agent、kpctl 的 Linux amd64/arm64 交叉编译和 Docker 镜像构建通过。
- `govulncheck v1.6.0` 未发现调用链漏洞，`npm audit` 漏洞数为 0。
- 本地候选镜像和从 Docker Hub 重新拉取的公开镜像均通过
  `image_e2e=pass`；公开镜像版本、源码 revision、非 root、只读根文件系统、
  healthcheck、固定 `kejilion.sh` revision 与摘要均通过核对。

## 154 真实浏览器生命周期验收

- 使用公开镜像
  `docker.io/kjlion/kejilion-panel:0.26.1` 启动全新数据目录的隔离 Panel，
  仅绑定远端 `127.0.0.1:18082`，经 SSH 隧道由 Codex in-app Browser
  操作真实 DOM。
- 为覆盖用户现场的旧响应，隔离 Panel 只读复用生产 `0.26.0 Agent`
  的 socket/token；未启动第二个具有 Docker 写能力的 Agent，未替换或重启
  生产 Panel。
- 授权测试应用为 `bomb-party`，完整容器 ID 在整个测试期间保持
  `983b39f5e3f0e640044cfcc6acf13c7c362ef5781423c28b32192b88bcddef1a`。
- `running`：卡片与详情均显示运行中；页面只有一个 `[role="dialog"]`，
  `body.has-modal=true`，端口 `0.0.0.0:3020` 可见。
- `exited`：旧 Agent 的真实 `/v1/apps` 响应确认
  `state="exited"`、`ports=null`；详情仍显示“已停止”“没有可用 HTTP
  端口”和“启动”，关闭后 `dialog=0`、`body.has-modal=false`，再次打开成功。
- `start`：从真实详情页点击“启动”，原容器 ID 不变并恢复
  `running/healthy`。
- `restart`：从真实详情页点击“重启”，最终恢复 `running/healthy`，
  `StartedAt` 更新，详情保持可见；未实际捕获短暂的 `restarting`，该分支由
  `restarting + ports:null` 组件测试覆盖。
- `paused`：对锁定的测试容器执行真实 `docker pause`，卡片和详情均显示
  “已暂停”，详情可打开且端口仍可见；`docker unpause` 后无需额外重启即恢复
  `running/healthy`。
- 整个生命周期的 KPanel 页面控制台错误列表为空，没有 `null.find`、
  `TypeError` 或未处理 Promise 错误。浏览器控制插件访问其自身
  `ab.chatgpt.com` 辅助遥测时发生超时，只造成控制命令变慢，不属于 KPanel
  页面错误，也未影响 DOM 操作和 154 状态读取。
- 清理后业务容器 `name → id → state → image` 与基线完全一致；生产 Panel
  ID 仍为
  `ebb14cb7195060a3559597b96710c2da2a47938d3da464610c5dbdc0bf389305`
  且 `running/healthy`，生产 Agent 仍为 `active`，测试应用为
  `running/healthy`。候选容器、数据目录、SSH 隧道和本地临时凭据均已删除。

## 线上产物

- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.26.1>
- 生产镜像：
  `docker.io/kjlion/kejilion-panel@sha256:07cd9bee0367effd279beaf923f986cd508e74cfe8efa2deb5f701338aef5752`
- `0.26.1` 与 `latest` 为同一 manifest digest，包含 linux/amd64、
  linux/arm64 及对应证明清单。
- 发布附件已独立下载并通过 `SHA256SUMS`：
  - `kejilion-agent-linux-amd64`：
    `fa5e2531fa674ecacc86b49fb7af9474b0a6bd457a32aba7f6338f8e00cb06d2`
  - `kejilion-agent-linux-arm64`：
    `b8ee1c7f29d53805dc2e51c9494c14cbd8d390fe6c2bb8925d63bd9d020edc5d`
  - `kejilion-panel-deploy-0.26.1.tar.gz`：
    `7d6c4f90e732896a3416b40c4fccef483f7757bc416b5c9263ccfb671299d655`
- 主仓库与应用市场的 `kpanel.conf` SHA-256 均为
  `B46717B1CA057752305FB2DB657E7E194766AACAD949982A2722B402BFC1B87D`；
  `v0.26.0..v0.26.1` 无应用契约变化，因此未修改、提交或推送
  `C:\GitHub\kejilion\apps`。

## 隔离、回滚与待确认边界

- 发布只来自独立工作树
  `C:\GitHub\kejilion-panel-fix-app-details-026`；旧会话的
  `feature/cluster-monitoring/f518213` 不是发布提交祖先，其 42 个已提交文件和
  当时未提交的 v2 文件均未进入 `v0.26.1`。
- 代码回滚点：标签 `v0.26.0`。
- 镜像回滚点：
  `docker.io/kjlion/kejilion-panel@sha256:703a56b9ab8e5e6fd0a1e7ee21f16b060d647de150efe1d4ac18e3b836a9a9ed`。
- 本版本没有数据库格式迁移，也没有更新应用市场安装契约。
- 为避免额外启动具有 Docker 写能力的同版本 Agent，真实浏览器验收重点覆盖
  更危险的旧契约 `ports:null`。新 Agent 的 `ports:[]` 由精确发布源码的 Go
  回归测试、Release 门禁和公开镜像验收覆盖；未声称在第二 Agent 浏览器会话中
  观察到该数组形态。

## 工作流沉淀

- `.codex-workflows/kpanel-real-machine-app-lifecycle.workflow.yaml`：
  固化隔离候选、真实浏览器、运行/停止/重启/暂停状态矩阵、旧/新 Agent
  端口契约和强制恢复清理步骤。
- `.codex-workflows/release-kpanel.workflow.yaml`：
  固化独立工作树、候选 CI、远端主分支锁、L3、应用市场条件同步、Release、
  公开镜像和回滚验收流程。
