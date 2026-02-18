# 贡献指南

> 🌐 **Language / 语言**: [English](../enUS/CONTRIBUTING.md) | [中文](CONTRIBUTING.md) | [Français](../frFR/CONTRIBUTING.md) | [Italiano](../itIT/CONTRIBUTING.md) | [日本語](../jaJP/CONTRIBUTING.md) | [Deutsch](../deDE/CONTRIBUTING.md) | [한국어](../koKR/CONTRIBUTING.md)

感谢你对 Stargate 项目的关注！我们欢迎所有形式的贡献。

## 📋 目录

- [如何贡献](#如何贡献)
- [开发环境设置](#开发环境设置)
- [代码规范](#代码规范)
- [提交规范](#提交规范)
- [Pull Request 流程](#pull-request-流程)
- [问题报告与功能请求](#问题报告与功能请求)
- [Traefik 集成开发](#traefik-集成开发)

## 🚀 如何贡献

你可以通过以下方式贡献：

- **报告 Bug**: 在 GitHub Issues 中报告问题
- **提出功能建议**: 在 GitHub Issues 中提出新功能想法
- **提交代码**: 通过 Pull Request 提交代码改进
- **改进文档**: 帮助改进项目文档
- **回答问题**: 在 Issues 中帮助其他用户
- **测试集成**: 测试 Traefik 集成和 Warden/Herald 集成

参与本项目时，请尊重所有贡献者，接受建设性的批评，并专注于对项目最有利的事情。

## 🛠️ 开发环境设置

### 前置要求

- Go 1.26 或更高版本
- Redis（可选，用于会话存储测试）
- Git
- Traefik（可选，用于集成测试）

### 快速开始

```bash
# 1. Fork 并克隆项目
git clone https://github.com/your-username/stargate.git
cd stargate

# 2. 添加上游仓库
git remote add upstream https://github.com/soulteary/stargate.git

# 3. 安装依赖
go mod download

# 4. 运行测试
go test ./...

# 5. 启动本地服务
chmod +x start-local.sh
./start-local.sh

# 或手动启动
export AUTH_HOST=localhost
export PASSWORDS=plaintext:test123
go run src/cmd/stargate/main.go
```

### 使用 Traefik 进行测试

测试 Traefik 集成：

1. **启动 Stargate**:
   ```bash
   export AUTH_HOST=auth.example.com
   export PASSWORDS=plaintext:test123
   go run src/cmd/stargate/main.go
   ```

2. **配置 Traefik**（示例 `traefik.yml`）:
   ```yaml
   entryPoints:
     web:
       address: ":80"
   
   forwardAuth:
     address: "http://localhost:8080/_auth"
     authResponseHeaders:
       - X-Forwarded-User
   ```

3. **测试 forwardAuth**: 通过 Traefik 访问受保护的服务

### 使用 Warden 和 Herald 进行测试

测试服务集成：

1. **启动 Warden**（如果测试 Warden 集成）
2. **启动 Herald**（如果测试 Herald 集成）
3. **配置 Stargate**:
   ```bash
   export WARDEN_ENABLED=true
   export WARDEN_URL=http://warden:8080
   export WARDEN_API_KEY=your-api-key
   
   export HERALD_ENABLED=true
   export HERALD_URL=http://herald:8082
   export HERALD_HMAC_SECRET=your-secret
   ```

## 📝 代码规范

请遵循以下代码规范：

1. **遵循 Go 官方代码规范**: [Effective Go](https://go.dev/doc/effective_go)
2. **格式化代码**: 运行 `go fmt ./...`
3. **代码检查**: 使用 `golangci-lint` 或 `go vet ./...`
4. **编写测试**: 新功能必须包含测试
5. **添加注释**: 公共函数和类型必须有文档注释
6. **常量命名**: 所有常量必须使用 `ALL_CAPS` (UPPER_SNAKE_CASE) 命名风格

### 测试要求

- 所有新功能必须包含单元测试
- Forward auth 功能必须包含集成测试
- 尽可能测试 Traefik 集成
- 测试覆盖率应保持或提高
- 提交 PR 前运行 `go test ./...`

## 📦 提交规范

### Commit Message 格式

我们使用 [Conventional Commits](https://www.conventionalcommits.org/) 规范：

```
<type>(<scope>): <subject>

<body>

<footer>
```

### Type 类型

- `feat`: 新功能
- `fix`: 修复 Bug
- `docs`: 文档更新
- `style`: 代码格式调整（不影响代码运行）
- `refactor`: 代码重构
- `perf`: 性能优化
- `test`: 测试相关
- `chore`: 构建过程或辅助工具的变动

### 示例

```
feat(auth): 添加跨域会话共享

实现了安全的跨域会话交换机制。

Closes #123
```

```
fix(forwardauth): 修复认证请求头处理问题

修复了 API 请求时认证请求头未正确设置的问题。

Fixes #456
```

## 🔄 Pull Request 流程

### 创建 Pull Request

```bash
# 1. 创建功能分支
git checkout -b feature/your-feature-name

# 2. 进行更改并提交
git add .
git commit -m "feat: 添加新功能"

# 3. 同步上游代码
git fetch upstream
git rebase upstream/main

# 4. 推送分支并创建 PR
git push origin feature/your-feature-name
```

### Pull Request 检查清单

在提交 Pull Request 之前，请确保：

- [ ] 代码遵循项目代码规范
- [ ] 所有测试通过（`go test ./...`）
- [ ] 代码已格式化（`go fmt ./...`）
- [ ] 添加了必要的测试
- [ ] 更新了相关文档
- [ ] Commit message 遵循 [提交规范](#提交规范)
- [ ] 代码已通过 lint 检查
- [ ] 已测试 Traefik 集成（如适用）
- [ ] 已测试 Warden/Herald 集成（如适用）

所有 Pull Request 都需要经过代码审查，请及时响应审查意见。

## 🐛 问题报告与功能请求

在创建 Issue 之前，请先搜索现有的 Issues，确认问题或功能未被报告。

### Bug 报告模板

```markdown
**描述**
清晰简洁地描述 Bug。

**复现步骤**
1. 执行 '...'
2. 看到错误

**预期行为**
清晰简洁地描述你期望发生什么。

**实际行为**
清晰简洁地描述实际发生了什么。

**环境信息**
- OS: [e.g. macOS 12.0]
- Go 版本: [e.g. 1.26]
- Traefik 版本: [e.g. v2.10]（如适用）
- Stargate 版本: [e.g. v1.0.0]
```

### 功能请求模板

```markdown
**功能描述**
清晰简洁地描述你想要的功能。

**问题描述**
这个功能解决了什么问题？为什么需要它？

**建议的解决方案**
清晰简洁地描述你希望如何实现这个功能。
```

## 🔗 Traefik 集成开发

如果你正在开发 Traefik 集成功能：

### Forward Auth 中间件

Stargate 实现 Traefik Forward Auth 中间件：

- **端点**: `GET /_auth`
- **响应头**: 成功时设置 `X-Forwarded-User`
- **错误处理**: 返回 401 或重定向到登录页面

### 测试 Forward Auth

1. **启动 Stargate** 并使用测试配置
2. **配置 Traefik** 使用 Stargate forwardAuth
3. **测试受保护的路由** 通过 Traefik
4. **验证认证** 请求头是否正确设置

### 集成示例

查看 `docs/enUS/DEPLOYMENT.md` 了解 Traefik 配置示例。

## 🎯 开始贡献

如果你想贡献但不知道从哪里开始，可以关注：

- 标记为 `good first issue` 的 Issues
- 标记为 `help wanted` 的 Issues
- 代码中的 `TODO` 注释
- 文档改进（修复错别字、改进清晰度、添加示例）
- 测试覆盖率改进
- Traefik 集成测试
- Warden/Herald 客户端改进

如有问题，请查看现有的 Issues 和 Pull Requests，或在相关 Issue 中提问。

---

再次感谢你对 Stargate 项目的贡献！🎉
