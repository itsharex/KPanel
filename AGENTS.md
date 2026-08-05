# KPanel 会话协作规则

本文件是 Codex 在整个 `kejilion-panel` 仓库中的执行入口。所有 Codex 会话开始工作前必须先阅读
`PROJECT_RULES.md`、`docs/project-management.md` 和 `docs/multi-agent-collaboration.md`；如涉及具体
业务，再读取对应设计文档和现有实现。Claude 使用 `CLAUDE.md` 进入同一套共享规范。

KPanel 的研发执行者可以是 Codex、Claude 或其他经授权智能体。共享规则和自动门禁必须与工具无关；
`.codex-workflows/` 只负责把共享流程适配为 Codex 可发现、可执行、可验证和可恢复的步骤。

## Codex 适配入口

1. 先运行 `git status --short`、`git branch --show-current` 和 `git rev-parse --short HEAD`，
   记录回滚点并保留用户已有改动。
2. 读取 `PROJECT_RULES.md`，再用 `rg` 定位与任务直接相关的实现、测试和设计文档；已有证据
   足够时禁止重复全仓扫描。
3. 若存在 `.codex-workflows/`，先列出工作流；发布、真机应用回归、站点图标验收和整体质量
   审计优先复用对应工作流，不在会话里临时重造流程。
4. 开发阶段按改动运行 `make verify-change`；跨域、高风险或发布前运行 `make verify-release`。
   不得把“本地构建成功”替代 race、安全扫描、最终镜像和生产限制验证。
5. 代码修改使用最小、可回滚补丁；新网络入口、宿主机写操作、交互终端、归档和身份验证
   必须先补失败边界测试，再实现成功路径。
6. 结论必须区分已验证事实、分析结论和未验证风险，并回传修改文件、命令、结果、回滚点；
   未经用户明确授权不得提交、推送、打 tag、发布或部署。

统一自动门禁由以下入口负责，任何智能体都不应复制参数不同的平行命令：

- `scripts/check-version-consistency.sh`：发行版本元数据一致性；
- `scripts/security-scan.sh`：固定摘要 Trivy 源码和最终镜像扫描；
- `scripts/verify-deploy.sh`：在隔离根文件系统验证安装安全，不读取或碰撞已部署 KPanel；
- `make verify-change`：按差异选择的日常验证；
- `make verify-release`：完整测试、race、`govulncheck`、`npm audit`、Trivy、Linux 构建和最终镜像验证；
- `.github/workflows/ci.yml` 与 `.github/workflows/release.yml`：远端独立复核和发布阻断。

## 协作入口

- `KPanel · 协调中心` 是用户的唯一沟通入口，负责拆解、分派、跟踪、验收和汇总。
- GitHub 的 AI Task Issue 是 Codex、Claude 和其他智能体共享的全局任务真源；Codex 任务列表只用于
  Codex 内部复用和等待。
- 用户已授权协调中心在 KPanel 范围内登记跨智能体任务，并按需复用或创建 Codex 任务；启动 Claude
  等外部智能体时仍使用对应工具的会话或编排入口。
- 子任务不要求用户逐个跟进；协调中心必须等待结果，并把最终结论统一反馈给用户。
- 不把会话 ID 写入仓库。每次通过项目路径、标题、状态和任务摘要动态查找，避免 ID 失效。

## 并行写入硬规则

- 一个写任务只能拥有一个独立 Git worktree 和一个短期分支；一个分支同时只能有一个写任务。
- `C:\GitHub\kejilion-panel` 默认工作树只用于同步、盘点和只读核对，不承载并行功能开发或发布候选。
- 功能开发、缺陷修复、项目规范和发布候选必须使用不同 worktree；禁止在其他任务的工作树中执行
  `git switch`、`git checkout`、`git reset`、`git clean` 或删除分支。
- 发布任务拥有唯一发布通道。候选进入 L3 后范围冻结；新功能继续在独立分支开发，但不得进入当前候选。
- 发现当前分支、`HEAD` 或工作树状态被其他任务改变时立即停止，保留现场并按
  `docs/project-management.md` 的冲突恢复流程迁移到新 worktree；不得覆盖或清理他人改动。

## 路由规则

1. 先明确目标、范围、限制、交付物和验收方式。
2. 先列出开放的 GitHub AI Task Issue，再列出 Codex 任务，优先匹配：
   - Issue 中的领域、允许路径、负责人和状态与当前任务一致；
   - 工作目录为 KPanel 主仓或其专用 worktree，且摘要明确指向 KPanel；
   - 标题使用 `KPanel · Codex · <角色> · <领域>`；
   - 领域一致且没有冲突中的未完成任务。
3. 有合适任务时发送后续消息继续使用；没有时才创建新任务。
4. 新任务标题统一为：
   - `KPanel · Codex · 开发 · <领域>`
   - `KPanel · Codex · 分析 · <领域>`
   - `KPanel · Codex · 验证 · <领域>`
5. 同一工作目录内禁止两个写任务并发。只读分析可以并发；所有代码或文档写入都应使用独立 Git
   worktree，或由协调中心串行执行。
6. 子任务必须回传：完成内容、修改文件、验证结果、风险、提交哈希（如有）。
7. 结果先更新到对应 AI Task Issue，再由协调中心复核差异和测试；未经用户明确授权，不发布、部署
   或推送远端。

## 新建任务的环境选择

- 如果 Codex 已将当前 KPanel 仓库注册为独立 Git 项目，新写任务默认使用
  worktree。
- 如果只有父级 `C:\GitHub` 项目可用，新任务必须先由协调中心创建专用 worktree，并在提示中明确
  该 worktree 的绝对路径。共享的 `C:\GitHub\kejilion-panel` 默认只做同步、只读分析或盘点。
- 创建后立即设置规范标题；后续通过标题和项目上下文复用，不依赖手工登记表。

## 子任务提示模板

```text
你是 KPanel 的 Codex <角色>会话。AI Task Issue 为 #<编号>，项目路径为<当前 KPanel worktree 绝对路径>。
先阅读 AGENTS.md、PROJECT_RULES.md、docs/project-management.md、
docs/multi-agent-collaboration.md 和与任务相关的现有文件。
任务：<目标>
范围：<允许修改或检查的内容；禁止路径；共享契约>
基线/分支：<base commit> / <branch>
限制：保留用户已有改动；不发布、不部署、不推送；不要扩大任务范围。
验收：<测试或检查>
完成后先更新 Issue，再回传：完成内容、修改文件、验证结果、风险、提交哈希和交接状态。
```

## 协调中心验收

- 使用 `wait_threads` 获取子任务最终状态，避免频繁轮询。
- 收到结果后检查工作树、目标文件和必要测试；不能只依据子任务自述。
- 失败时优先把具体错误继续发给同一任务；只有领域变化或上下文污染时才新建任务。
- 完成后按 `PROJECT_RULES.md` 汇报，并说明是否新建或复用了任务。
