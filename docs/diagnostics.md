# KPanel 体检与第三方测试协议

## 产品形态

体检是独立于应用市场的一次性后台任务，不创建长期服务或安装状态。页面按脚本原有业务分组展示：

- IP 与解锁；
- 网络线路；
- 硬件性能；
- 综合评测。

每个项目显示名称、说明、第三方来源、预计耗时和资源影响。管理员普通确认后即可启动，
页面每两秒读取一次持久化状态与日志尾部，任务结束后结果保留在历史记录。

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
- 单任务日志最多写入 8 MiB；
- API 只返回清理控制字符后的最近 400 行；
- 最多保留 50 条任务历史；
- Web 启动动作通过 Session、Origin、CSRF 与审计检查；
- 脚本必须由 root 持有、不可被 group/other 写入，并包含许可接受和体检协议标记。

第三方脚本可能安装依赖、消耗带宽/CPU/磁盘。YABS 与 CPU 测试沿用脚本语义：主机没有 Swap
时会创建 1 GiB `/swapfile`。页面在执行前明确提示这些影响。

## 验收状态

- **已验证**：目录解析、HTTPS 校验、未知 selector 拒绝、协议许可校验、固定 systemd
  worker、单任务冲突、日志截断、API 路由、前端类型检查/测试/构建。
- **已实现未实机验证**：目标服务器上的 transient unit、第三方命令真实下载和完整跑分。
- **不在本版范围**：自定义测试命令、任意 URL、并行跑分、交互式目标 IP 输入。
