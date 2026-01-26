#!/bin/bash

# StarGate 本地启动脚本
# 用于快速启动和测试 StarGate 认证服务
# 使用方式: ./start-local.sh [选项]
# 选项:
#   -port, --port PORT     设置服务端口（默认: 8080）
#   -h, --help             显示帮助信息

set -e

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 解析命令行参数
CUSTOM_PORT=""
while [[ $# -gt 0 ]]; do
    case $1 in
        -port|--port)
            CUSTOM_PORT="$2"
            shift 2
            ;;
        -h|--help)
            echo "StarGate 本地启动脚本"
            echo ""
            echo "使用方式: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  -port, --port PORT     设置服务端口（默认: 8080）"
            echo "  -h, --help             显示帮助信息"
            echo ""
            echo "环境变量:"
            echo "  可以通过环境变量设置配置，命令行参数会覆盖环境变量"
            echo "  PORT                   服务端口"
            echo "  AUTH_HOST              认证服务主机名"
            echo "  PASSWORDS              密码配置"
            echo "  DEBUG                  调试模式（true/false）"
            echo "  LANGUAGE               界面语言（zh/en）"
            echo "  WARDEN_ENABLED         启用 Warden 集成（true/false）"
            echo "  WARDEN_URL             Warden 服务地址"
            echo "  WARDEN_API_KEY         Warden API 密钥"
            echo "  WARDEN_CACHE_TTL       Warden 缓存 TTL（秒）"
            exit 0
            ;;
        *)
            echo -e "${RED}错误: 未知参数: $1${NC}"
            echo "使用 $0 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

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

# 设置默认值（命令行参数优先级最高，然后是环境变量，最后是默认值）
AUTH_HOST=${AUTH_HOST:-"localhost"}
PASSWORDS=${PASSWORDS:-"plaintext:test123|admin123"}
DEBUG=${DEBUG:-"true"}
LANGUAGE=${LANGUAGE:-"zh"}
# 端口设置：命令行参数 > 环境变量 > 默认值
if [ -n "$CUSTOM_PORT" ]; then
    PORT="$CUSTOM_PORT"
elif [ -z "$PORT" ]; then
    PORT="8080"
fi

# Warden 配置（可选）
WARDEN_ENABLED=${WARDEN_ENABLED:-"false"}
WARDEN_URL=${WARDEN_URL:-""}
WARDEN_API_KEY=${WARDEN_API_KEY:-""}
WARDEN_CACHE_TTL=${WARDEN_CACHE_TTL:-"300"}

echo -e "${BLUE}配置信息:${NC}"
echo "  AUTH_HOST: $AUTH_HOST"
echo "  PASSWORDS: $PASSWORDS"
echo "  DEBUG: $DEBUG"
echo "  LANGUAGE: $LANGUAGE"
echo "  端口: $PORT"
if [ -n "$CUSTOM_PORT" ]; then
    echo -e "  ${GREEN}✓ 端口通过命令行参数设置: $CUSTOM_PORT${NC}"
fi
echo ""
echo -e "${BLUE}Warden 配置:${NC}"
echo "  WARDEN_ENABLED: $WARDEN_ENABLED"
if [ "$WARDEN_ENABLED" = "true" ]; then
    echo "  WARDEN_URL: ${WARDEN_URL:-"未设置"}"
    echo "  WARDEN_API_KEY: ${WARDEN_API_KEY:+已设置}"
    echo "  WARDEN_CACHE_TTL: $WARDEN_CACHE_TTL (秒)"
fi
echo ""

# 提示用户
echo -e "${YELLOW}提示:${NC}"
echo "  1. 访问登录页面: http://localhost:$PORT/_login?callback=localhost"
echo "  2. 测试密码: test123 或 admin123"
if [ "$WARDEN_ENABLED" = "true" ]; then
    echo "  3. Warden 模式已启用，可以使用用户列表认证"
    echo "  4. 确保 WARDEN_URL 和 WARDEN_API_KEY 已正确配置"
fi
echo "  按 Ctrl+C 停止服务"
echo ""

# 设置环境变量
export AUTH_HOST
export PASSWORDS
export DEBUG
export LANGUAGE
export PORT
export WARDEN_ENABLED
export WARDEN_URL
export WARDEN_API_KEY
export WARDEN_CACHE_TTL

# 进入源代码目录
cd src

# 启动服务器
echo -e "${GREEN}正在启动服务器...${NC}\n"
# 使用包路径运行，这样会自动包含包内的所有文件
# 注入开发版本号
VERSION="dev-$(git rev-parse --short HEAD 2>/dev/null || echo 'local')"
COMMIT="$(git rev-parse HEAD 2>/dev/null || echo 'unknown')"
BUILD_DATE="$(date +%FT%T%z)"
go run -ldflags "-X 'github.com/soulteary/version-kit.Version=${VERSION}' -X 'github.com/soulteary/version-kit.Commit=${COMMIT}' -X 'github.com/soulteary/version-kit.BuildDate=${BUILD_DATE}'" ./cmd/stargate
