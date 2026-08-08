# KPanel Claude 协作入口

本文件是 Claude/Claude Code 参与 KPanel 开发时的入口。开始任何任务前完整阅读：

1. `PROJECT_RULES.md`
2. `docs/project-management.md`
3. `docs/multi-agent-collaboration.md`
4. 与任务相关的设计文档、实现和测试

## 强制规则

- 远端任务分支、聚焦提交、CI 和发布记录是跨 Codex、Claude 和其他智能体的任务真源。GitHub Issue、
  PR、API 和 `gh` 登录不是写任务前置条件。
- Git fetch/push 统一使用 SSH 远端 `git@github.com:kejilion/KPanel.git`；私钥和本机 SSH 配置不得写入
  仓库。任务开始前必须明确负责人、允许路径、基线、分支、worktree、验收等级和权限。
- 一个写任务只拥有一个独立 Git worktree 和一个短期分支；不得在
  `C:\GitHub\kejilion-panel` 管理工作树中开发。
- 开始和恢复任务时重新读取 `origin/main`、目标远端分支、最新提交、`git worktree list` 和工作树状态；
  不根据旧会话记忆继续写入。
- 不修改其他智能体声明的路径，不切换、重置、清理或删除其他任务的工作树、分支和未提交内容。
- 使用 `PROJECT_RULES.md` 的 L0-L3 核验等级以及项目的 `make verify-change`、
  `make verify-release` 权威入口，不创建 Claude 专属的平行门禁。
- `.codex-workflows/` 是 Codex 适配层。除非任务就是维护 Codex 工作流，否则 Claude 只读取其内容
  作为参考，不把它当作跨智能体状态真源。
- 未经用户明确授权，不提交、推送、快进 `main`、打标签、发布或部署。

## 任务交付

结束前形成聚焦提交；已获推送授权时通过 SSH 推送任务分支，并在回复中同步：

```text
AI / 会话角色：
目标与范围：
worktree / branch / base：
提交：
修改文件：
验证命令与结果：
未验证风险：
依赖或冲突：
建议状态与目标版本：
回滚方式：
推送/合并/发布状态：
```

跨智能体移交必须先释放原负责人写入权，再由接手者重新核对远端分支、提交和工作树状态；禁止两个智能体
在同一分支或 worktree 中同时写入。
