## 项目概述

AcePanel 是基于 Go 语言开发的新一代 Linux 服务器运维管理面板。项目采用前后端分离架构：
- 后端：Go 1.25 + go-chi 路由 + GORM + Wire 依赖注入
- 前端：Vue 3 + Vite + Naive UI + pnpm

**重要提示：** 项目目前正在进行 v3 版本重构，主要包括重构网站/备份/计划任务模块等。

## 语言和编码规范

**所有代码注释、文档和回复必须使用简体中文。**

## 构建和测试

### 后端构建

构建主程序：
```bash
go build -o ace ./cmd/ace
```

构建 CLI 工具：
```bash
go build -o cli ./cmd/cli
```

构建时注入版本信息：
```bash
VERSION="1.0.0"
BUILD_TIME="$(date -u '+%F %T UTC')"
COMMIT_HASH="$(git rev-parse --short HEAD)"
GO_VERSION="$(go version | cut -d' ' -f3)"

LDFLAGS="-s -w --extldflags '-static'"
LDFLAGS="${LDFLAGS} -X 'github.com/acepanel/panel/internal/app.Version=${VERSION}'"
LDFLAGS="${LDFLAGS} -X 'github.com/acepanel/panel/internal/app.BuildTime=${BUILD_TIME}'"
LDFLAGS="${LDFLAGS} -X 'github.com/acepanel/panel/internal/app.CommitHash=${COMMIT_HASH}'"
LDFLAGS="${LDFLAGS} -X 'github.com/acepanel/panel/internal/app.GoVersion=${GO_VERSION}'"

go build -trimpath -buildvcs=false -ldflags "${LDFLAGS}" -o ace ./cmd/ace
```

### 运行测试

运行所有测试：
```bash
go test -v ./...
```

运行测试并生成覆盖率报告：
```bash
go test -v -coverprofile="coverage.out" ./...
```

运行单个测试：
```bash
go test -v -run TestFunctionName ./path/to/package
```

### 前端开发

进入前端目录：
```bash
cd web
```

安装依赖：
```bash
pnpm install
```

开发模式（带热重载）：
```bash
pnpm dev
```

类型检查：
```bash
pnpm type-check
```

代码检查：
```bash
pnpm lint
```

构建生产版本：
```bash
pnpm build
```

## 代码架构

项目采用类 DDD 分层架构，依赖关系为：route -> service -> biz <- data

### 核心目录结构

- **`cmd/`**: 程序入口
    - `ace/`: 面板主程序
    - `cli/`: 命令行工具

- **`internal/app/`**: 应用入口和配置

- **`internal/route/`**: HTTP 路由定义
    - 定义路由规则
    - 注入所需的 service 依赖

- **`internal/service/`**: 服务层（类似 DDD 的 application 层）
    - 处理 HTTP 请求/响应
    - DTO 到 DO 的转换
    - 协调多个 biz 接口完成业务流程
    - **不应处理复杂业务逻辑**

- **`internal/biz/`**: 业务逻辑层（类似 DDD 的 domain 层）
    - 定义业务接口（Repository 模式）
    - 定义领域模型和数据结构
    - 使用依赖倒置原则：biz 定义接口，data 实现接口

- **`internal/data/`**: 数据访问层（类似 DDD 的 repository 层）
    - 实现 biz 中定义的业务接口
    - 封装数据库、缓存等操作
    - 处理数据持久化逻辑

- **`internal/http/`**: HTTP 相关
    - `middleware/`: 自定义中间件
    - `request/`: 请求结构体定义
    - `rule/`: 自定义验证规则

- **`internal/apps/`**: 面板子应用实现

- **`internal/bootstrap/`**: 各模块启动引导

- **`internal/migration/`**: 数据库迁移

- **`internal/job/`**: 后台任务

- **`internal/queuejob/`**: 任务队列

- **`pkg/`**: 工具函数和通用包
    - 包含各种独立的工具模块
    - 可被项目任何部分引用

- **`web/`**: Vue 3 前端项目

## 开发新功能的标准流程

1. **在 `internal/route/` 中添加路由**
    - 参考已有路由文件（如 `http.go`）
    - 注入需要的 service 依赖
    - 定义路由规则和 handler 映射

2. **在 `internal/service/` 中实现服务方法**
    - **先阅读已有的类似服务**以了解代码风格
    - 处理请求验证和响应格式化
    - 使用 `Success()` 返回成功响应
    - 使用 `Error()` 返回错误响应
    - 使用 `ErrorSystem()` 返回系统严重错误
    - 调用 biz 层接口完成业务逻辑

3. **在 `internal/biz/` 中定义业务接口**
    - **先阅读已有的类似接口定义**
    - 定义 Repository 接口（如 `WebsiteRepo`）
    - 定义领域模型结构体（如 `Website`）
    - 保持接口简洁明确

4. **在 `internal/data/` 中实现 biz 接口**
    - **先阅读已有的类似实现**
    - 创建 repo 结构体（如 `websiteRepo`）
    - 实现构造函数（如 `NewWebsiteRepo`）
    - 实现所有接口方法
    - 处理数据库操作和缓存逻辑

5. **使用 Wire 进行依赖注入**
    - 在对应的 wire.go 文件中添加 provider
    - 运行 `go generate` 生成依赖注入代码

## 技术栈特定注意事项

### Go 语言规范

- 使用 Go 1.25 稳定版本
- 遵循 Go 标准库和习惯用法
- 日志使用标准库的 `slog` 包
- 使用 `github.com/samber/lo` 进行函数式编程辅助

### 当前框架

- 路由：`github.com/go-chi/chi/v5`
- ORM：`gorm.io/gorm`
- 依赖注入：`github.com/google/wire`
- 验证：`github.com/gookit/validate`

### 助手函数（service 层）

在 service 层使用以下助手函数：
- `Success(w, data)`: 返回成功响应
- `Error(w, statusCode, format, args...)`: 返回错误响应
- `ErrorSystem(w, format, args...)`: 返回系统严重错误（500）
- `Bind[T](r)`: 绑定请求参数到泛型类型 T
- `Paginate[T](...)`: 构建分页响应

### 数据库

- 使用 SQLite（`github.com/ncruces/go-sqlite3`）
- 使用 GORM 进行数据库迁移和操作

### 安全性

- 实现认证/授权（JWT）
- 防止 SQL 注入（使用 GORM 参数化查询）
- 防止 XSS 和 CSRF 攻击
- 实现速率限制（`github.com/sethvargo/go-limiter`）

## 代码风格

- 所有代码注释必须使用简体中文
- 遵循 Go 官方代码风格
- 使用 `gofmt` 格式化代码
- 复杂逻辑添加注释说明
- 导出的函数和类型必须有注释

## Wire 依赖注入

项目使用 Wire 进行依赖注入。当添加新的依赖时：

1. 在 `cmd/ace/wire.go` 或 `cmd/cli/wire.go` 中添加 provider
2. 运行生成命令：
```bash
go generate ./...
```

## 前端开发注意事项

- 使用 Vue 3 Composition API
- UI 框架：Naive UI
- 状态管理：Pinia
- HTTP 请求：Alova
- 图标：@iconify/vue
- 终端：xterm.js
- 遵循项目已有的组件结构和编码风格

## 配置文件

开发时需要准备配置文件：
```bash
cp config.example.yml config.yml
```

前端开发配置：
```bash
cd web
cp .env.production .env
cp settings/proxy-config.example.ts settings/proxy-config.ts
```
