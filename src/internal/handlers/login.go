package handlers

import (
	"fmt"
	"html"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"

	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// LoginAPI handles POST requests to /_login for password authentication.
// It validates the password from the form data, creates a session if valid,
// and redirects to the callback URL (if provided) or returns a success response.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginAPI(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		password := ctx.FormValue("password")

		if !auth.CheckPassword(password) {
			return SendErrorResponse(ctx, fiber.StatusUnauthorized, i18n.T("error.invalid_password"))
		}

		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		err = auth.Authenticate(sess)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.authenticate_failed"))
		}

		// 获取 callback 参数（优先从 cookie 获取，其次从表单数据，最后从查询参数获取）
		callbackFromCookie := GetCallbackFromCookie(ctx)
		callback := callbackFromCookie
		if callback == "" {
			callback = ctx.FormValue("callback")
		}
		if callback == "" {
			callback = ctx.Query("callback")
		}

		// 如果从 cookie 获取到了 callback，登录成功后清除 cookie
		if callbackFromCookie != "" {
			ClearCallbackCookie(ctx)
		}

		// 如果没有 callback，尝试使用来源域名作为 callback
		if callback == "" {
			originHost := GetForwardedHost(ctx)
			// 只有当来源域名与认证服务域名不一致时，才使用来源域名作为 callback
			if IsDifferentDomain(ctx) {
				callback = originHost
			}
		}

		// 如果有 callback，重定向到会话交换端点
		if callback != "" {
			// 获取 session ID（应该已经存在）
			sessionID := sess.ID()
			if sessionID == "" {
				// 如果 ID 为空，尝试从响应 cookie 中获取
				// 在 Fiber session 中，Save() 会将 session ID 设置到响应 cookie 中
				cookieBytes := ctx.Response().Header.Peek("Set-Cookie")
				if len(cookieBytes) > 0 {
					cookieStr := string(cookieBytes)
					// 查找 session cookie (格式: stargate_session=<session_id>; ...)
					cookieName := auth.SessionCookieName + "="
					if idx := strings.Index(cookieStr, cookieName); idx >= 0 {
						start := idx + len(cookieName)
						end := start
						for end < len(cookieStr) && cookieStr[end] != ';' && cookieStr[end] != ' ' {
							end++
						}
						sessionID = cookieStr[start:end]
					}
				}
				// 如果仍然为空，尝试重新获取 session
				if sessionID == "" {
					sess, err = store.Get(ctx)
					if err != nil {
						return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
					}
					sessionID = sess.ID()
				}
				if sessionID == "" {
					// 如果 session ID 仍然为空，返回错误
					return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.missing_session_id"))
				}
			}
			proto := GetForwardedProto(ctx)
			if proto == "" {
				proto = ctx.Protocol()
			}
			redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
			return ctx.Redirect(redirectURL)
		}

		// 如果仍然没有 callback（说明来源域名就是认证服务本身），根据请求类型返回响应
		if IsHTMLRequest(ctx) {
			// HTML 请求返回成功消息，并添加 meta refresh 重定向到来源域名
			ctx.Set("Content-Type", "text/html; charset=utf-8")
			successMsg := i18n.T("success.login")

			// 获取来源域名和协议
			originHost := GetForwardedHost(ctx)
			proto := GetForwardedProto(ctx)
			redirectURL := fmt.Sprintf("%s://%s", proto, originHost)

			// 转义 URL 以确保 HTML 安全
			escapedURL := html.EscapeString(redirectURL)

			// 构建包含 meta refresh 的 HTML
			htmlContent := fmt.Sprintf(`<html><head><meta charset="UTF-8"><meta http-equiv="refresh" content="0;url=%s"><title>%s</title></head><body><h1>%s</h1><p>%s</p><p><a href="%s">点击这里如果页面没有自动跳转</a></p></body></html>`,
				escapedURL, successMsg, successMsg, successMsg, escapedURL)
			return ctx.Status(fiber.StatusOK).SendString(htmlContent)
		}

		// API 请求返回 JSON 响应
		ctx.Set("Content-Type", "application/json")
		response := fiber.Map{
			"success": true,
			"message": i18n.T("success.login"),
		}
		// 如果 session ID 存在，则添加到响应中
		if sessionID := sess.ID(); sessionID != "" {
			response["session_id"] = sessionID
		}
		return ctx.Status(fiber.StatusOK).JSON(response)
	}
}

// LoginRoute handles GET requests to /_login for displaying the login page.
// If the user is already authenticated, it redirects to the session exchange endpoint.
// Otherwise, it renders the login page template.
//
// Parameters:
//   - store: Session store for managing user sessions
//
// Returns a Fiber handler function.
func LoginRoute(store *session.Store) func(c *fiber.Ctx) error {
	return func(ctx *fiber.Ctx) error {
		// 获取 callback 参数（优先从 URL 查询参数获取，其次从 cookie 获取）
		// URL 参数优先，因为它是当前请求的明确意图
		callback := ctx.Query("callback")
		if callback == "" {
			callback = GetCallbackFromCookie(ctx)
		} else {
			// 如果 URL 中有 callback 参数，更新 cookie（如果域名不一致）
			SetCallbackCookie(ctx, callback)
		}

		sess, err := store.Get(ctx)
		if err != nil {
			return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
		}

		if auth.IsAuthenticated(sess) {
			// 使用 X-Forwarded-* 头部构建正确的重定向 URL
			sessionID := sess.ID()
			if sessionID == "" {
				return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.missing_session_id"))
			}
			proto := GetForwardedProto(ctx)
			if proto == "" {
				proto = ctx.Protocol()
			}
			// 如果有 callback，重定向到 callback 的 _session_exchange 端点
			// 如果没有 callback，重定向到当前主机的根路径
			if callback != "" {
				redirectURL := fmt.Sprintf("%s://%s/_session_exchange?id=%s", proto, callback, sessionID)
				return ctx.Redirect(redirectURL)
			}
			// 没有 callback 时，重定向到当前主机的根路径
			host := GetForwardedHost(ctx)
			redirectURL := fmt.Sprintf("%s://%s/", proto, host)
			return ctx.Redirect(redirectURL)
		}

		return ctx.Render("login", fiber.Map{
			"Callback":   callback,
			"SessionID":  sess.ID(),
			"Title":      config.LoginPageTitle.Value,
			"FooterText": config.LoginPageFooterText.Value,
		})
	}
}
