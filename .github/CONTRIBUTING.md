# 贡献指南

感谢你考虑为 AcePanel 做出贡献！这份文档将帮助你了解如何参与到项目中来。

## 目录

- [我能做什么贡献？](#我能做什么贡献)
- [开发环境设置](#开发环境设置)
- [开发流程](#开发流程)
- [代码规范](#代码规范)
- [提交信息规范](#提交信息规范)
- [Pull Request 流程](#pull-request-流程)

## 我能做什么贡献？

你可以通过以下方式为 AcePanel 做出贡献：

- 🐛 报告 Bug
- 💡 提出新功能建议
- 📝 改进文档
- 🔧 修复 Bug
- ✨ 实现新功能
- 🧪 编写测试
- 🌍 翻译面板及文档

## 开发环境设置

### 前置要求

- Go 1.25 或更高版本
- Node.js 22+ 和 pnpm
- Git
- 本地开发环境没有特殊要求，但必须在 Linux 上运行和测试
- 在开发及测试前端项目时，需修改面板配置文件打开 debug 模式或者关闭安全入口

### 克隆仓库

```bash
git clone https://github.com/acepanel/panel.git
cd panel
```

### 后端设置

1. 安装 Go 依赖：

```bash
go mod download
```

2. 复制配置文件：

```bash
cp config.example.yml config.yml # 按需修改配置
```

3. 构建项目：

```bash
go build -o ace ./cmd/ace # 主程序
go build -o cli ./cmd/cli # CLI 工具
```

### 前端设置

1. 进入前端目录：

```bash
cd web
```

2. 安装依赖：

```bash
pnpm install
```

3. 配置开发环境：

```bash
cp .env.production .env # 按需修改
cp settings/proxy-config.example.ts settings/proxy-config.ts # 配置 Linux 测试服务器信息
pnpm run gen-auto-import # 生成自动导入文件，开发中无需导入 vue alova 等常用包
pnpm run gettext:compile # 预编译翻译文件，否则开发中没有翻译
```

4. 启动开发服务器：

```bash
pnpm dev
```

## 开发流程

### 项目架构

AcePanel 采用类 DDD 分层架构，依赖关系为：route → service → biz ← data

主要目录结构：

- `cmd/` - 程序入口（ace 主程序、cli 工具）
- `internal/route/` - HTTP 路由定义
- `internal/service/` - 服务层（处理 HTTP 请求/响应）
- `internal/biz/` - 业务逻辑层（定义业务接口和领域模型）
- `internal/data/` - 数据访问层（实现 biz 接口）
- `pkg/` - 工具函数和通用包
- `web/` - Vue 3 前端项目

### 开发新功能的标准流程

1. **在 `internal/route/` 中添加路由**
    - 按需注入需要的服务

2. **在 `internal/service/` 中实现服务方法**
    - 处理请求验证和响应格式化
    - 使用 `Success()` 返回成功响应
    - 使用 `Error()` 返回错误响应
    - 使用 `ErrorSystem()` 返回系统严重错误

3. **在 `internal/biz/` 中定义业务接口**
    - 定义 Repository 接口
    - 定义领域模型结构体
    - 保持接口简洁明确

4. **在 `internal/data/` 中实现 biz 接口**
    - 创建 repo 结构体
    - 实现构造函数
    - 实现所有接口方法

5. **使用 Wire 进行依赖注入**
    - 在对应的 `包名.go` 文件中添加 Provider
    - 运行 `go generate ./...` 生成依赖注入代码

## 代码规范

**所有代码注释必须使用简体中文**

面板基于 Gettext 搭建了自动化国际化流程，所有对用户可见的文本均需支持国际化，原文使用英文。

开发中 Go 代码中注入 `*gotext.Locale`，前端导入 `useGettext` 进行翻译。

### Go 代码规范

- 遵循 Go 官方代码风格
- 使用 `gofmt` 格式化代码和 `golangci-lint` 检查代码质量
- 函数和方法注释必须以函数名开头，复杂逻辑应添加注释说明

### 前端代码规范

- 使用 Vue 3 Composition API
- 遵循项目已有的组件结构和编码风格
- 使用 TypeScript 进行类型检查
- 运行 `pnpm lint` 检查代码质量

## 提交信息规范

我们使用语义化的提交信息，格式为：

```
<类型>(<范围>): <简短描述>

<详细描述>（可选）

<关联的 Issue>（可选）
```

### 类型

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档更新
- `style`: 代码格式调整（不影响功能）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建/工具链相关

### 示例

```
feat(website): 添加网站备份功能

实现了网站配置和数据的自动备份功能，支持：
- 按计划自动备份
- 手动立即备份
- 备份文件压缩存储

Closes #123
```

```
fix(apache): 修复代理配置解析错误

修复了在解析包含特殊字符的代理配置时的崩溃问题
```

## Pull Request 流程

### 1. Fork 项目

点击 GitHub 页面右上角的 "Fork" 按钮。

### 2. 创建开发分支

```bash
git checkout -b your-develop-name
```

### 3. 进行开发

- 遵循代码规范
- 中文编写必要的代码注释
- 添加必要的测试，目前主要针对 `pkg` 目录下的公共包
- 若进行大范围的重构/修改，请提前与维护者沟通

### 4. 提交更改

```bash
git add .
git commit -m "feat(scope): 描述你的更改"
```

### 5. 推送到你的 Fork

```bash
git push origin your-develop-name
```

### 6. 创建 Pull Request

1. 访问你 Fork 的仓库
2. 点击 "New Pull Request"
3. 选择你的分支
4. 填写 PR 检查单并点击 "Create Pull Request"

### 7. 等待审查

当 Pull Request 开发完毕后，请为其添加 `🚀 Review Ready` 标签，维护者将及时进行评审并提供反馈。请及时响应评论并根据需要进行修改。

## 许可证

通过向本项目贡献代码，你同意你的贡献将在与项目相同的许可证下发布。

---

再次感谢你的贡献！🎉
