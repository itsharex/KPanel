# KPanel 多智能体会话协作

## 使用方式

以后只需在置顶的 `KPanel · 协调中心` 中描述目标。协调中心先通过 SSH fetch 盘点远端任务分支、
worktree 和活跃任务，再决定交给 Codex、Claude 或其他智能体；领域匹配时复用任务，否则创建新的
独立 worktree/分支，并在结果返回后统一验收和汇报。

完整的角色、跨智能体真源、worktree 隔离、状态流转、发布冻结、交接、冲突恢复和权限规则以
[`project-management.md`](project-management.md) 为准；具体运行方式见
[`multi-agent-collaboration.md`](multi-agent-collaboration.md)。本文件只保留日常使用入口。

建议用一句话同时说明目标和验收标准，例如：

```text
检查应用市场安装终端卡住的问题，修复后跑相关 Go 测试，不要部署。
```

## 会话分工

| 标题 | 职责 |
| --- | --- |
| `KPanel · 协调中心` | 接收需求、路由、跟踪、验收、汇总 |
| `KPanel · <AI> · 分析 · <领域>` | 只读排查、设计和风险分析 |
| `KPanel · <AI> · 开发 · <领域>` | 在明确文件范围内实现修改 |
| `KPanel · <AI> · 验证 · <领域>` | 独立运行测试和复核结果 |

会话 ID 不写入项目。SSH 远端分支、聚焦提交、CI 和发布记录构成跨工具索引；标题、项目路径和
会话状态只用于各 AI 工具内部恢复。GitHub Issue、PR、API 和 `gh` 登录不作为协作前置条件。

## 并发与工作树

当前 Codex 保存的项目是父目录 `C:\GitHub`，而不是独立的
`C:\GitHub\kejilion-panel` Git 项目。因此：

- 新建会话和复用会话均可正常工作；
- 共享目录中的只读分析可以并发；
- 两个会话不能同时修改同一工作树；
- 写任务不得再直接使用共享的 `C:\GitHub\kejilion-panel`；协调中心必须先创建专用 worktree；
- 应尽快在 Codex 中把 `C:\GitHub\kejilion-panel` 添加为独立项目，之后新开发任务默认使用
  Codex worktree。

## 决策顺序

1. 读取共享项目规则，执行 `git fetch origin --prune` 并核对当前 Git 状态。
2. 查找同领域、同路径范围的远端分支、worktree 和跨智能体任务。
3. 无冲突时继续旧任务；没有合适任务时创建专用 worktree、任务分支和目标智能体任务。
4. 等待目标工具回传；获授权后通过 SSH 推送任务分支，协调中心从精确提交独立复核。
5. 向用户统一说明完成项、修改、验证、风险、交接和知识沉淀。

可复用的完整执行步骤见：

- `.codex-workflows/session-collaboration.workflow.yaml`
- `.codex-workflows/release-kpanel.workflow.yaml`

## 历史初始化验证

2026-07-27 已完成一次 Codex 内部真实链路测试：

- 通过保存的 `C:\GitHub` 项目创建只读 KPanel 验证任务；
- 验证任务正确读取协作协议 `v1.0` 和当前分支；
- 向同一任务发送第二次请求，返回 `REUSE-OK` 和项目版本 `0.18.1`；
- 两次请求使用同一个任务，且均未修改文件、提交、推送或部署。

该记录只证明 Codex 会话复用可用，不代表跨 Codex/Claude 协作已验收。跨工具协作以 SSH 远端提交、
CI 和独立验证为准。
