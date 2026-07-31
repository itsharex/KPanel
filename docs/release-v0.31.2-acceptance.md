# KPanel v0.31.2 发布验收

## 范围

- 右键未勾选的文件或文件夹时，自动单选并勾选该项后打开菜单。
- 右键已在多选集合中的项目时，保留现有多选结果。
- 继续使用固定在路径工具栏内的批量操作栏，不推动文件列表位置。

## 验收结果

- [x] Web 类型检查、156 项单元测试和生产构建。
- [x] 右键未选项目与右键已选项目两种选择行为的回归测试。
- [x] 生态规则检查。
- [x] Linux CI、Go 测试与 vet、部署安装安全测试。
- [x] 漏洞扫描、依赖审计和 kejilion.sh 应用生命周期测试。
- [x] Linux amd64/arm64 镜像构建、运行时健康检查与镜像契约检查。
- [x] GitHub Release、版本镜像和 `latest` 镜像发布。

## 发布证据

- 发布提交：`5bfa892`
- 主线 CI：<https://github.com/kejilion/KPanel/actions/runs/30593110013>
- Release：<https://github.com/kejilion/KPanel/actions/runs/30593200733>
- GitHub Release：<https://github.com/kejilion/KPanel/releases/tag/v0.31.2>
- Docker Hub `0.31.2` / `latest`：
  `sha256:960ac7c7ced5dd22bd71d2db6ec7f9cfbb29af809850e05e9b7299b67bffb265`
- 平台：`linux/amd64`、`linux/arm64`

## 兼容与回滚

- 本次只修改浏览器端选择状态，不改变文件 API、权限、数据和任务协议。
- 回滚版本：`v0.31.1`
- 回滚镜像：
  `docker.io/kjlion/kejilion-panel@sha256:3048ee2c4b693261107a1c739239e7a23f4d7313c5136aee85b28c846190bfbd`
