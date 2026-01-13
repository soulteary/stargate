# 文档索引

欢迎查阅 Stargate Forward Auth Service 的文档。

## 🌐 多语言文档 / Multi-language Documentation

- [English](../enUS/README.md) | [中文](README.md) | [Français](../frFR/README.md) | [Italiano](../itIT/README.md) | [日本語](../jaJP/README.md) | [Deutsch](../deDE/README.md) | [한국어](../koKR/README.md)

## 📚 文档列表

### 核心文档

- **[README.md](../../README.zhCN.md)** - 项目概述和快速开始指南
- **[ARCHITECTURE.md](ARCHITECTURE.md)** - 技术架构和设计决策

### 详细文档

- **[API.md](API.md)** - 完整的 API 端点文档
  - 认证检查端点
  - 登录和登出端点
  - 会话交换端点
  - 健康检查端点
  - 错误响应格式
  - 认证流程示例

- **[CONFIG.md](CONFIG.md)** - 配置参考文档
  - 配置方式
  - 必需配置项
  - 可选配置项
  - 密码配置详解
  - 配置示例
  - 配置最佳实践

- **[DEPLOYMENT.md](DEPLOYMENT.md)** - 部署指南
  - Docker 部署
  - Docker Compose 部署
  - Traefik 集成
  - 生产环境部署
  - 监控和维护
  - 故障排查

## 🚀 快速导航

### 新手入门

1. 阅读 [README.zhCN.md](../../README.zhCN.md) 了解项目
2. 查看 [快速开始](../../README.zhCN.md#快速开始) 部分
3. 参考 [配置说明](../../README.zhCN.md#配置说明) 配置服务

### 开发人员

1. 阅读 [ARCHITECTURE.md](ARCHITECTURE.md) 了解架构
2. 查看 [API.md](API.md) 了解 API 接口
3. 参考 [开发指南](../../README.zhCN.md#开发指南) 进行开发

### 运维人员

1. 阅读 [DEPLOYMENT.md](DEPLOYMENT.md) 了解部署方式
2. 查看 [CONFIG.md](CONFIG.md) 了解配置选项
3. 参考 [故障排查](DEPLOYMENT.md#故障排查) 解决问题

## 📖 文档结构

```
codes/
├── README.md              # 项目主文档（英文）
├── README.zhCN.md         # 项目主文档（中文）
├── docs/
│   ├── enUS/
│   │   ├── README.md       # 文档索引（英文）
│   │   ├── ARCHITECTURE.md # 架构文档（英文）
│   │   ├── API.md          # API 文档（英文）
│   │   ├── CONFIG.md       # 配置参考（英文）
│   │   └── DEPLOYMENT.md   # 部署指南（英文）
│   └── zhCN/
│       ├── README.md       # 文档索引（中文，本文件）
│       ├── ARCHITECTURE.md # 架构文档（中文）
│       ├── API.md          # API 文档（中文）
│       ├── CONFIG.md       # 配置参考（中文）
│       └── DEPLOYMENT.md   # 部署指南（中文）
└── ...
```

## 🔍 按主题查找

### 配置相关

- 环境变量配置：[CONFIG.md](CONFIG.md)
- 密码配置：[CONFIG.md#密码配置](CONFIG.md#密码配置)
- 配置示例：[CONFIG.md#配置示例](CONFIG.md#配置示例)

### API 相关

- API 端点列表：[API.md](API.md)
- 认证流程：[API.md#认证流程示例](API.md#认证流程示例)
- 错误处理：[API.md#错误响应格式](API.md#错误响应格式)

### 部署相关

- Docker 部署：[DEPLOYMENT.md#docker-部署](DEPLOYMENT.md#docker-部署)
- Traefik 集成：[DEPLOYMENT.md#traefik-集成](DEPLOYMENT.md#traefik-集成)
- 生产环境：[DEPLOYMENT.md#生产环境部署](DEPLOYMENT.md#生产环境部署)

### 架构相关

- 技术栈：[ARCHITECTURE.md#技术栈](ARCHITECTURE.md#技术栈)
- 项目结构：[ARCHITECTURE.md#项目结构](ARCHITECTURE.md#项目结构)
- 核心组件：[ARCHITECTURE.md#核心组件](ARCHITECTURE.md#核心组件)

## 💡 使用建议

1. **首次使用**：从 [README.zhCN.md](../../README.zhCN.md) 开始，按照快速开始指南操作
2. **配置服务**：参考 [CONFIG.md](CONFIG.md) 了解所有配置选项
3. **集成 Traefik**：查看 [DEPLOYMENT.md](DEPLOYMENT.md) 中的 Traefik 集成部分
4. **开发扩展**：阅读 [ARCHITECTURE.md](ARCHITECTURE.md) 了解架构设计
5. **问题排查**：查看 [DEPLOYMENT.md#故障排查](DEPLOYMENT.md#故障排查)

## 📝 文档更新

文档会随着项目的发展持续更新。如果发现文档有误或需要补充，欢迎提交 Issue 或 Pull Request。

## 🤝 贡献

欢迎贡献文档改进：

1. 发现错误或需要改进的地方
2. 提交 Issue 描述问题
3. 或直接提交 Pull Request
