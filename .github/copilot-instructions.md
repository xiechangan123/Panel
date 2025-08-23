## 项目描述

本项目是基于 Go 语言的 Fiber 框架和 wire 依赖注入开发的 AcePanel Linux 服务器运维管理面板，目前正在进行 v3 版本重构。

v3 版本需要完成以下重构任务：
1. 使用 Fiber v3 替换目前的 go-chi 路由
2. 全新的项目模块，支持运行 Java/Go/Python 等项目
3. 网站模块重构，支持多 Web 服务器（Apache/OLS/Kangle）
4. 备份模块重构，需要支持 s3 和 ftp/sftp 备份途径
5. 计划任务模块重构，支持管理备份任务和自定义脚本任务等

## 项目结构

├── cmd/
│   ├── ace/ 面板主程序
│   └── cli/ 面板命令行工具
├── internal/
│   ├── app/ 应用入口
│   ├── apps/ 面板各子应用的实现
│   ├── biz/ 业务逻辑的接口和数据库模型定义，类似 DDD 的 domain 层，data 类似 DDD 的 repo，而业务接口在这里定义，使用依赖倒置的原则
│   ├── bootstrap/ 各个模块的启动引导
│   ├── data/ 业务数据访问，包含 cache、db 等封装，实现了 biz 的业务接口。我们可能会把 data 与 dao 混淆在一起，data 偏重业务的含义，它所要做的是将领域对象重新拿出来，我们去掉了 DDD 的 infra 层
│   ├── http/
│   │   ├── middleware/ 自定义路由中间件
│   │   ├── request/ 请求结构体
│   │   └── rule/ 自定义验证规则
│   ├── job/ 面板后台任务
│   ├── migration/ 数据库迁移定义
│   ├── queuejob/ 面板任务队列
│   ├── route/ 路由定义
│   └── service/ 实现了路由定义的服务层，类似 DDD 的 application 层，处理 DTO 到 biz 领域实体的转换(DTO -> DO)，同时协同各类 biz 交互，但是不应处理复杂逻辑
├── mocks/ 模拟数据，目前没有使用
├── pkg/ 工具函数及包
├── storage/ 数据存储
└── web/ 前端项目

### 架构

- 后端：Go 语言，使用 chi 路由（正在迁移至 Fiber v3）+ wire 依赖注入 + GORM
- 前端：Vue 3 + TypeScript + Vite + Naive UI + pnpm
- 数据库：SQLite3
- 配置：基于 YAML 的 config.yml 文件

## 引导程序和依赖项

- 使用 pnpm 安装 Go 1.24+ 和 Node.js：
- `go version` -- 验证 Go 1.24+
- `npm install -g pnpm` -- 安装 pnpm 包管理器
- `cd /home/runner/work/panel/panel && cp config.example.yml config.yml` -- 复制所需的配置文件
- 下载依赖项：
- `go mod download` -- 大约需要 30 秒。请勿取消。请将超时设置为 60 分钟以上。
- `cd web && pnpm install` -- 大约需要 30 秒。请勿取消。请将超时设置为 60 分钟以上。

## 构建流程

- 构建后端应用：
- `go build -o ace ./cmd/ace` -- 耗时约 14 秒。请勿取消。请将超时时间设置为 30 分钟以上。
- `go build -o cli ./cmd/cli` -- 耗时约 1 秒。请勿取消。请将超时时间设置为 30 分钟以上。
- 构建前端应用：
- `cd web && cp .env.production .env && cp settings/proxy-config.example.ts settings/proxy-config.ts`
- `cd web && pnpm run gettext:compile` -- 编译翻译，耗时约 1 秒
- `cd web && pnpm build` -- 耗时约 30 秒。请勿取消。请将超时时间设置为 60 分钟以上。
- 构建工件位置：
- 后端二进制文件：在仓库根目录中构建为 `ace` 和 `cli`
- 前端资源：构建到 `web/dist/` 并复制到 `pkg/embed/frontend/`

## 开发指南

- 使用 github.com/gofiber/fiber/v3 和 gorm.io/gorm 进行开发
- Fiber v3 handler 使用 `c fiber.Ctx` 而不是 `c *fiber.Ctx`
- 使用泛型 Bind 助手函数绑定请求，使用 Success 助手函数响应成功，使用 Error/ErrorSystem 助手函数响应错误，泛型 Paginate 助手函数构建各种分页响应
- 遵循项目的 DDD 分层架构：biz → data → service → route
- 使用标准库 slog 进行日志记录
- 编写完整、安全、高效的代码，不留待办事项
- 使用 testify/suite 模式编写测试

## 开发新需求时的流程

1. 在 route/http 中添加新的路由和注入需要的服务
2. 在 service 中添加新的服务方法，先读取已存在的其他服务方法，以参考它们的实现方式
3. 在 biz 中添加新的业务逻辑需要的接口等，先读取已存在的其他接口，以参考它们的实现方式
4. 在 data 中实现 biz 的接口，先读取已存在的其他实现，以参考它们的实现方式
