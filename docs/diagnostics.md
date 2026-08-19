# KPanel 体检与第三方测试协议

## 产品形态

体检是独立于应用市场的一次性后台任务，不创建长期服务或安装状态。页面按脚本原有业务分组展示：

- IP 与解锁；
- 网络线路；
- 硬件性能；
- 综合评测。

每个项目显示名称、说明、第三方来源、预计耗时和资源影响。管理员普通确认后即可启动，
页面每两秒读取一次持久化状态；右侧使用 PTY 终端持续读取原始输出，任务结束后结果保留在历史记录。

## 业务真源

`kejilion.sh` 是目录和执行方式的唯一真源：

```text
KJ_TEST_NONINTERACTIVE=1 k test list
KJ_TEST_NONINTERACTIVE=1 k test run <fixed-selector>
```

`list` 输出 `KPANEL_TEST_CATEGORY` 和 `KPANEL_TEST_ITEM` 制表符记录。Agent 只接受：

- `^[a-z0-9][a-z0-9-]{0,47}$` 形式的固定分类与 selector；
- HTTPS、无凭证、无 fragment 的第三方来源；
- 1 至 120 分钟的预计耗时；
- `light`、`network` 或 `intensive` 三种资源影响。

KPanel 不维护第二份命令表，不接收 URL、Shell 文本或自定义参数。脚本更新目录后，页面刷新即可
读取新目录；未知或格式错误的记录会使整个目录不可用，不会降级执行。

## 执行与恢复

Agent 为每个任务创建 `kejilion-panel-diagnostic-<job-id>` transient systemd unit，
再由 root-only `diagnostic-run` 子命令执行可信脚本。任务状态和日志位于
`/var/lib/kejilion-panel/diagnostic-jobs`。

- 同一时间只运行一个体检；
- 最长运行 90 分钟；
- 每个任务创建独立 FIFO 输入通道与 `xterm-256color` PTY；终端保留 ANSI 颜色并支持按脚本提示输入；
- 固定第三方来源先下载到权限受限的临时文件，再由 PTY 执行，避免 `curl | bash` 占用脚本标准输入；
- Agent 重启后继续读取同一系统启动周期内的后台任务；系统重启或任务超过 100 分钟时，
  自动把遗留的运行状态标记为中断，避免永久阻塞新体检；
- 单任务日志最多写入 8 MiB；
- 普通任务 API 只返回清理控制字符后的最近 400 行；终端 API 按字节偏移返回原始日志；
- 最多保留 50 条任务历史；
- Web 启动和终端输入动作通过 Session、Origin、CSRF 与长度/NUL 校验；
- 脚本必须由 root 持有、不可被 group/other 写入，并包含许可接受和体检协议标记。

第三方脚本可能安装依赖、消耗带宽/CPU/磁盘。YABS 与 CPU 测试沿用脚本语义：主机没有 Swap
时会创建 1 GiB `/swapfile`。页面在执行前明确提示这些影响。

任务完成后，Agent 会从脚本原始输出中提取明确存在的少量指标，写入任务的 `summary` 字段，供体检首页汇总展示：

- NodeQuality/YABS 使用各自的文本与 JSON 字段适配；IPQuality、NetQuality 和 SuperSpeed 使用对应的报告结构或输出行适配；
- 只保留性能、路由、延迟、网速和 IP 质量五类白名单字段，不计算或猜测一个不存在的“综合分数”；
- 未识别的字段、线路明细、第三方提示和完整报告仍以终端原始输出为准，不因汇总失败而丢失；
- `summary` 缺失时，旧任务仍可在读取完成状态时按同一规则从持久化日志懒解析，保持升级兼容。

## 验收状态

- **已验证**：目录解析、HTTPS 校验、未知 selector 拒绝、协议许可校验、固定 systemd
  worker、单任务冲突、跨重启与超时恢复、日志截断、ANSI 原样传输、输入 CSRF、API 路由、
  前端类型检查/测试/构建。
- **已实现未实机验证**：目标服务器上的 transient unit、第三方命令真实下载和完整跑分。
- **不在本版范围**：自定义测试命令、任意 URL、并行跑分。
