package handlers

import (
	"testing"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/valyala/fasthttp"
)

func createTestContextForUtils(method, path string, headers map[string]string) (*fiber.Ctx, *fiber.App) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})

	ctx.Request().SetRequestURI(path)
	ctx.Request().Header.SetMethod(method)

	for k, v := range headers {
		ctx.Request().Header.Set(k, v)
	}

	return ctx, app
}

func TestGetForwardedHost_WithHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host": "forwarded.example.com",
		"Host":             "original.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedHost(ctx)
	testza.AssertEqual(t, "forwarded.example.com", result)
}

func TestGetForwardedHost_WithoutHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Host": "original.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedHost(ctx)
	testza.AssertEqual(t, "original.example.com", result)
}

func TestGetForwardedHost_EmptyHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host": "",
		"Host":             "original.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedHost(ctx)
	testza.AssertEqual(t, "original.example.com", result)
}

func TestGetForwardedURI_WithHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/original", map[string]string{
		"X-Forwarded-Uri": "/forwarded",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedURI(ctx)
	testza.AssertEqual(t, "/forwarded", result)
}

func TestGetForwardedURI_WithoutHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/original", nil)
	defer app.ReleaseCtx(ctx)

	result := GetForwardedURI(ctx)
	// ctx.Path() returns "/" by default in test context, not the request URI
	testza.AssertEqual(t, "/", result)
}

func TestGetForwardedURI_EmptyHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/original", map[string]string{
		"X-Forwarded-Uri": "",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedURI(ctx)
	// ctx.Path() returns "/" by default in test context
	testza.AssertEqual(t, "/", result)
}

func TestGetForwardedProto_WithHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Proto": "https",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedProto(ctx)
	testza.AssertEqual(t, "https", result)
}

func TestGetForwardedProto_WithoutHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", nil)
	defer app.ReleaseCtx(ctx)

	result := GetForwardedProto(ctx)
	// Default protocol should be http
	testza.AssertEqual(t, "http", result)
}

func TestGetForwardedProto_EmptyHeader(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Proto": "",
	})
	defer app.ReleaseCtx(ctx)

	result := GetForwardedProto(ctx)
	// ctx.Protocol() may return empty string in test context
	// The function should return empty string if protocol is not set
	testza.AssertTrue(t, result == "" || result == "http", "should return empty or http")
}

func TestBuildCallbackURL(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
	})
	defer app.ReleaseCtx(ctx)

	result := BuildCallbackURL(ctx)
	expected := "https://auth.example.com/_login?callback=app.example.com"
	testza.AssertEqual(t, expected, result)
}

func TestBuildCallbackURL_WithoutHeaders(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Host": "app.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := BuildCallbackURL(ctx)
	expected := "http://auth.example.com/_login?callback=app.example.com"
	testza.AssertEqual(t, expected, result)
}

func TestIsHTMLRequest_EmptyAccept(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", nil)
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertTrue(t, result, "empty Accept header should default to HTML")
}

func TestIsHTMLRequest_TextHTML(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "text/html",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertTrue(t, result)
}

func TestIsHTMLRequest_TextHTMLWithQuality(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "text/html;q=0.9",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertTrue(t, result)
}

func TestIsHTMLRequest_MultipleTypesWithHTML(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "text/html,application/json",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertTrue(t, result)
}

func TestIsHTMLRequest_WildcardFirst(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "*/*",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertTrue(t, result, "wildcard */* should be treated as HTML")
}

func TestIsHTMLRequest_WildcardNotFirst(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/json,*/*",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertFalse(t, result, "wildcard not first should not be treated as HTML")
}

func TestIsHTMLRequest_ApplicationJson(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/json",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertFalse(t, result)
}

func TestIsHTMLRequest_ApplicationXml(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/xml",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertFalse(t, result)
}

func TestIsHTMLRequest_WithSpaces(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": " text/html , application/json ",
	})
	defer app.ReleaseCtx(ctx)

	result := IsHTMLRequest(ctx)
	testza.AssertTrue(t, result)
}

func TestSendErrorResponse_JSON(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/json",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 401, "Unauthorized")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 401, ctx.Response().StatusCode())
	testza.AssertEqual(t, "application/json", string(ctx.Response().Header.Peek("Content-Type")))
	testza.AssertContains(t, string(ctx.Response().Body()), "Unauthorized")
	testza.AssertContains(t, string(ctx.Response().Body()), "401")
}

func TestSendErrorResponse_XML(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/xml",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 500, "Internal Server Error")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 500, ctx.Response().StatusCode())
	testza.AssertEqual(t, "application/xml", string(ctx.Response().Header.Peek("Content-Type")))
	body := string(ctx.Response().Body())
	testza.AssertContains(t, body, "Internal Server Error")
	testza.AssertContains(t, body, "500")
	testza.AssertContains(t, body, "<error")
	testza.AssertContains(t, body, "</error>")
}

func TestSendErrorResponse_PlainText(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", nil)
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 404, "Not Found")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 404, ctx.Response().StatusCode())
	testza.AssertEqual(t, "text/plain", string(ctx.Response().Header.Peek("Content-Type")))
	testza.AssertEqual(t, "Not Found", string(ctx.Response().Body()))
}

func TestSendErrorResponse_PlainText_EmptyAccept(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 403, "Forbidden")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, 403, ctx.Response().StatusCode())
	testza.AssertEqual(t, "text/plain", string(ctx.Response().Header.Peek("Content-Type")))
	testza.AssertEqual(t, "Forbidden", string(ctx.Response().Body()))
}

func TestSendErrorResponse_JSON_WithQuality(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/json;q=0.9,application/xml;q=0.8",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 400, "Bad Request")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "application/json", string(ctx.Response().Header.Peek("Content-Type")))
}

func TestSendErrorResponse_XML_WithQuality(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/xml;q=0.9,text/plain;q=0.8",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 400, "Bad Request")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "application/xml", string(ctx.Response().Header.Peek("Content-Type")))
}

func TestSendErrorResponse_MultipleTypes_JSONFirst(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/json,application/xml,text/plain",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 401, "Unauthorized")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "application/json", string(ctx.Response().Header.Peek("Content-Type")))
}

func TestSendErrorResponse_MultipleTypes_XMLFirst(t *testing.T) {
	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"Accept": "application/xml,application/json,text/plain",
	})
	defer app.ReleaseCtx(ctx)

	err := SendErrorResponse(ctx, 401, "Unauthorized")
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "application/xml", string(ctx.Response().Header.Peek("Content-Type")))
}

func TestNormalizeHost_WithPort(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host": "app.example.com:8080",
		"Host":             "auth.example.com",
	})
	defer app.ReleaseCtx(ctx)

	// Test IsDifferentDomain which uses normalizeHost internally
	// With port, it should normalize and compare correctly
	result := IsDifferentDomain(ctx)
	testza.AssertTrue(t, result, "should detect different domains even with port")
}

func TestNormalizeHost_WithoutPort(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host": "app.example.com",
		"Host":             "auth.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := IsDifferentDomain(ctx)
	testza.AssertTrue(t, result, "should detect different domains")
}

func TestNormalizeHost_SameDomainWithPort(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com:80")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host": "auth.example.com:8080",
		"Host":             "auth.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := IsDifferentDomain(ctx)
	testza.AssertFalse(t, result, "should detect same domain even with different ports")
}

func TestValidateCallbackHost_DisallowsScheme(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	result := ValidateCallbackHost("https://evil.example.com")
	testza.AssertEqual(t, "", result)
}

func TestValidateCallbackHost_AllowsCookieDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	result := ValidateCallbackHost("app.example.com")
	testza.AssertEqual(t, "app.example.com", result)
}

func TestValidateCallbackHost_DisallowsOutsideCookieDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	result := ValidateCallbackHost("evil.example.net")
	testza.AssertEqual(t, "", result)
}

func TestSetCallbackCookie_EmptyCallback(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", nil)
	defer app.ReleaseCtx(ctx)

	// Should not panic with empty callback
	testza.AssertNotPanics(t, func() {
		SetCallbackCookie(ctx, "")
	})
}

func TestSetCallbackCookie_WithCookieDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host":  "app.example.com",
		"X-Forwarded-Proto": "https",
	})
	defer app.ReleaseCtx(ctx)

	SetCallbackCookie(ctx, "app.example.com")

	// Check that cookie was set
	cookies := ctx.Response().Header.Peek("Set-Cookie")
	cookieStr := string(cookies)
	testza.AssertContains(t, cookieStr, CallbackCookieName)
	testza.AssertContains(t, cookieStr, ".example.com")
}

func TestClearCallbackCookie_WithCookieDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", nil)
	defer app.ReleaseCtx(ctx)

	ClearCallbackCookie(ctx)

	// Check that cookie was cleared (expired)
	cookies := ctx.Response().Header.Peek("Set-Cookie")
	cookieStr := string(cookies)
	testza.AssertContains(t, cookieStr, CallbackCookieName)
	testza.AssertContains(t, cookieStr, ".example.com")
}

func TestIsDifferentDomain_SameDomain(t *testing.T) {
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	err := config.Initialize()
	testza.AssertNoError(t, err)

	ctx, app := createTestContextForUtils("GET", "/test", map[string]string{
		"X-Forwarded-Host": "auth.example.com",
		"Host":             "auth.example.com",
	})
	defer app.ReleaseCtx(ctx)

	result := IsDifferentDomain(ctx)
	testza.AssertFalse(t, result, "should detect same domain")
}
