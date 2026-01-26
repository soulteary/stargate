package main

const (
	// DefaultPort is the default server port
	DefaultPort = ":80"

	// RouteRoot is the root route
	RouteRoot = "/"
	// RouteLogin is the login page route
	RouteLogin = "/_login"
	// RouteLogout is the logout route
	RouteLogout = "/_logout"
	// RouteSessionExchange is the session exchange route
	RouteSessionExchange = "/_session_exchange"
	// RouteAuth is the authentication check route
	RouteAuth = "/_auth"
	// RouteHealth is the health check route
	RouteHealth = "/health"

	// StaticAssetsPath is the static assets path
	StaticAssetsPath = "./internal/web/templates/assets"
	// FaviconPath is the favicon file path
	FaviconPath = "./internal/web/templates/assets/favicon.ico"
)
