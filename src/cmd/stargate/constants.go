package main

const (
	// DefaultPort 默认服务器端口
	DefaultPort = ":80"

	// RouteRoot 根路由
	RouteRoot = "/"
	// RouteLogin 登录页面路由
	RouteLogin = "/_login"
	// RouteLogout 登出路由
	RouteLogout = "/_logout"
	// RouteSessionExchange 会话交换路由
	RouteSessionExchange = "/_session_exchange"
	// RouteAuth 认证检查路由
	RouteAuth = "/_auth"
	// RouteHealth 健康检查路由
	RouteHealth = "/health"

	// StaticAssetsPath 静态资源路径
	StaticAssetsPath = "./internal/web/templates/assets"
	// FaviconPath Favicon 文件路径
	FaviconPath = "./internal/web/templates/assets/favicon.ico"
)
