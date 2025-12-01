# AGENTS 指南

## 基本要求
- 所有回复、文档、代码注释必须使用简体中文。
- 项目处于 v3 重构期（网站/备份/计划任务等模块），保持现有架构和风格，避免随意改动。

## 项目概览与分层
- 技术栈：后端 Go 1.25 + go-chi + GORM + Wire；前端 Vue 3 + Vite + UnoCSS + Naive UI + pnpm（Node 24）。
- 分层：route -> service -> biz <- data，服务层只做编排/DTO 转换，业务逻辑放在 biz，数据访问在 data。
- 目录：`cmd/ace`/`cmd/cli` 入口；`internal/app` 配置/启动；`internal/route|service|biz|data|http|apps|bootstrap|migration|job|queuejob` 按职责拆分；`pkg/` 通用库与内嵌资源；`web/` 前端；`mocks/` 为 Mockery 生成的仓库接口；构建后的前端复制到 `pkg/embed/frontend`；多语言在 `pkg/embed/locales` 与 `web/src/locales`。
- 配置示例：`config.example.yml`；CI 脚本见 `.github/workflows/`。

## 开发约束
- 禁止在本地直接运行主程序，只允许在远程 Linux 服务器运行。
- 开发前准备：`cp config.example.yml config.yml`；前端开发可复制 `.env.development`（或按需 `.env.production`）为 `.env`，必要时复制 `settings/proxy-config.example.ts` 为 `settings/proxy-config.ts`。

## 构建与测试
- 后端：
  ```bash
  go test ./...
  go build ./cmd/ace
  go build ./cmd/cli
  # 如需注入版本信息可使用 go build -ldflags 方案，保持 -trimpath/-buildvcs=false 一致
  ```
- 前端：
  ```bash
  cd web
  pnpm install
  pnpm type-check
  pnpm lint
  pnpm dev
  pnpm build  # 产物输出 dist 并自动复制到 ../pkg/embed/frontend
  ```
- 后端单元测试仅覆盖 `pkg/` 公共包，`internal/` 无需测试；前端不写单元测试，依赖 TS 类型检查与 ESLint。

## 开发流程
1. 在 `internal/route/` 添加路由，注入所需 service。
2. 在 `internal/service/` 实现编排：参数校验、DTO 处理，返回 `Success`/`Error`/`ErrorSystem`。
3. 在 `internal/biz/` 定义接口与领域模型，保持精简，接口由 data 实现。
4. 在 `internal/data/` 用 GORM/缓存等实现仓库逻辑，遵循依赖倒置。
5. 更新对应 `wire.go` 并运行 `go generate ./...` 完成依赖注入。

## 编码规范
- Go：使用 `gofmt` 与 `golangci-lint`；导出符号需中文注释；错误返回统一使用 `error` 并添加上下文；避免循环依赖，包名短小；日志使用标准库 `slog`，可用 `samber/lo` 辅助；文件按领域拆分（如 `container_*`）。
- 前端：TypeScript + Vue SFC（组合式 API）；样式使用 UnoCSS/Naive UI 主题；状态集中在 `store/`（Pinia）；请求使用 Alova；命名采用帕斯卡组件名；Prettier 2 空格 + ESLint 规则。

## 数据与安全
- 默认数据库 SQLite（`github.com/ncruces/go-sqlite3`），通过 GORM 迁移与访问。
- 需要关注认证/授权（JWT）、SQL 注入防护、XSS/CSRF 防护、速率限制（`github.com/sethvargo/go-limiter`）。

## 提交与 PR
- 提交信息遵循惯例式格式（如 `chore(deps): ...`、`feat: ...`、`fix: ...`），一次提交聚焦单一主题。
- PR 应包含：变更摘要、关联 Issue/需求、测试命令与结果、前端可视化改动的截图；确保 CI（lint/test/build）在干净环境可复现。
