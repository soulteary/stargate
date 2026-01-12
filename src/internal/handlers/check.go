package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// handleNotAuthenticated handles unauthenticated requests.
// For HTML requests, it redirects to the login page.
// For API requests (JSON/XML), it returns a 401 error response.
func handleNotAuthenticated(ctx *fiber.Ctx) error {
	if IsHTMLRequest(ctx) {
		// HTML 请求：重定向到登录页
		callbackURL := BuildCallbackURL(ctx)
		return ctx.Redirect(callbackURL)
	}

	// 非 HTML 请求：返回 401 错误
	return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.auth_required"))
}

// CheckRoute is the main authentication check handler for Traefik Forward Auth.
// It validates requests in two ways:
//  1. Stargate-Password header authentication (for API requests)
//  2. Session cookie authentication (for web requests)
//
// On successful authentication, it sets the X-Forwarded-User header (or configured header name)
// and returns 200 OK. On failure, it either redirects to login (HTML) or returns 401 (API).
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func CheckRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			// 会话存储错误，返回 500 错误
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		// 处理 Stargate-Password Header 认证
		stargatePassword := ctx.Get("Stargate-Password")
		if stargatePassword != "" {
			if !auth.CheckPassword(stargatePassword) {
				return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
			}

			// 认证成功，设置用户信息头部
			// 由于 Stargate 使用密码认证，没有具体的用户名，使用默认值
			userHeaderName := config.UserHeaderName.String()
			ctx.Set(userHeaderName, "authenticated")
			return ctx.SendStatus(fiber.StatusOK)
		}

		// 检查会话认证
		if !auth.IsAuthenticated(sess) {
			return handleNotAuthenticated(ctx)
		}

		// 认证成功，设置用户信息头部
		// 由于 Stargate 使用密码认证，没有具体的用户名，使用默认值
		userHeaderName := config.UserHeaderName.String()
		ctx.Set(userHeaderName, "authenticated")

		return ctx.SendStatus(fiber.StatusOK)
	}
}
