# Project Workflows

本目录保存 KPanel 可跨会话复用的 Codex 工作流：

- `session-collaboration.workflow.yaml`：复用或创建任务、等待、复核并统一交付。
- `release-kpanel.workflow.yaml`：版本准备、CI、Release、Docker Hub、应用市场和线上验收。
- `kpanel-real-machine-app-lifecycle.workflow.yaml`：用隔离候选实例和真实 Chrome 验证应用
  在运行、停止、重启、暂停状态下均可打开详情，并恢复真机现场。

工作流由 `codex-workflows` 技能管理。使用前运行 `workflow.py list`，执行前使用
`workflow.py run <name> --param key=value` 渲染参数，修改后必须运行
`workflow.py validate <name>`。
