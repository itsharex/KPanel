# Project Workflows

本目录保存 KPanel 可跨 Codex 会话复用的执行适配工作流。跨 Codex、Claude 和其他智能体的共享
规则与状态真源位于 `PROJECT_RULES.md`、`docs/project-management.md`、
`docs/multi-agent-collaboration.md`、GitHub AI Task Issue 和 Git/CI；本目录不得形成第二套规范。

- `session-collaboration.workflow.yaml`：复用或创建任务、等待、复核并统一交付。
- `release-kpanel.workflow.yaml`：版本准备、CI、Release、Docker Hub、应用市场和线上验收。
- `quality-audit-kpanel.workflow.yaml`：快速迭代后的规范、功能、性能、安全和发布门禁健康审计。
- 最近一次完整证据见 [`docs/quality-audit-2026-08-02.md`](../docs/quality-audit-2026-08-02.md)，
  后续审计应复用相同指标和环境比较，不重复发明基线。
- `kpanel-real-machine-app-lifecycle.workflow.yaml`：用隔离候选实例和真实 Chrome 验证应用
  在运行、停止、重启、暂停状态下均可打开详情，并恢复真机现场。

工作流由 `codex-workflows` 技能管理。使用前运行 `workflow.py list`，执行前使用
`workflow.py run <name> --param key=value` 渲染参数，修改后必须运行
`workflow.py validate <name>`。
