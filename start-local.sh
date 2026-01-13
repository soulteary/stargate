#!/bin/bash

# StarGate 本地启动脚本
# 用于快速启动和测试 StarGate 认证服务

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== StarGate 本地启动脚本 ===${NC}\n"

# 检查是否在正确的目录
if [ ! -d "src" ]; then
    echo -e "${RED}错误: 请在 codes 目录下运行此脚本${NC}"
    exit 1
fi

# 检查 Go 是否安装
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go，请先安装 Go 1.18+${NC}"
    exit 1
fi

# 设置默认值
AUTH_HOST=${AUTH_HOST:-"localhost"}
PASSWORDS=${PASSWORDS:-"plaintext:test123|admin123"}
DEBUG=${DEBUG:-"true"}
LANGUAGE=${LANGUAGE:-"zh"}
PORT=${PORT:-"8080"}

echo -e "${YELLOW}配置信息:${NC}"
echo "  AUTH_HOST: $AUTH_HOST"
echo "  PASSWORDS: $PASSWORDS"
echo "  DEBUG: $DEBUG"
echo "  LANGUAGE: $LANGUAGE"
echo "  端口: $PORT"
echo ""

# 提示用户
echo -e "${YELLOW}提示:${NC}"
echo "  1. 访问登录页面: http://localhost:$PORT/_login?callback=localhost"
echo "  2. 测试密码: test123 或 admin123"
echo "  3. 按 Ctrl+C 停止服务"
echo ""

# 设置环境变量
export AUTH_HOST
export PASSWORDS
export DEBUG
export LANGUAGE
export PORT

# 进入源代码目录
cd src

# 启动服务器
echo -e "${GREEN}正在启动服务器...${NC}\n"
# 使用包路径运行，这样会自动包含包内的所有文件
# 注入开发版本号
VERSION="dev-$(git rev-parse --short HEAD 2>/dev/null || echo 'local')"
go run -ldflags "-X github.com/soulteary/stargate/src/cmd/stargate.Version=${VERSION}" ./cmd/stargate
