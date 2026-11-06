package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/template/html/v3"
	sessionkit "github.com/soulteary/session-kit/v3"
	sessionfiber "github.com/soulteary/session-kit/v3/fiberadapter"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/publicsuffix"
)

type browserExchangeTransport struct{ app *fiber.App }

func (t browserExchangeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Host != "auth.example.com" && request.URL.Host != "app.example.net" && request.URL.Host != "other.example.org" {
		return nil, fmt.Errorf("unexpected test origin %s", request.URL.Host)
	}
	cloned := request.Clone(request.Context())
	cloned.Header.Set("X-Forwarded-Host", request.URL.Host)
	cloned.Header.Set("X-Forwarded-Proto", request.URL.Scheme)
	response, err := t.app.Test(cloned)
	if err == nil {
		response.Request = request
	}
	return response, err
}

type exchangeBrowserFixture struct {
	t     *testing.T
	app   *fiber.App
	proto string
}

func newExchangeBrowserFixture(t *testing.T, secure bool) *exchangeBrowserFixture {
	t.Helper()
	variables := []*config.EnvVariable{&config.AuthHost, &config.CookieDomain, &config.CookieSecure, &config.CallbackAllowedHosts, &config.SessionExchangeSecret, &config.SessionExchangeRedirectPath, &config.Passwords, &config.WardenEnabled, &config.HeraldEnabled, &config.TrustedProxies}
	previous := make([]string, len(variables))
	for i, v := range variables {
		previous[i] = v.Value
	}
	t.Cleanup(func() {
		for i, v := range variables {
			v.Value = previous[i]
		}
	})
	config.AuthHost.Value = "auth.example.com"
	config.CookieDomain.Value = ""
	config.CookieSecure.Value = fmt.Sprint(secure)
	config.CallbackAllowedHosts.Value = "app.example.net,other.example.org"
	config.SessionExchangeSecret.Value = "browser-exchange-test-secret-at-least-32-bytes"
	config.SessionExchangeRedirectPath.Value = "/done"
	config.Passwords.Value = "plaintext:browser-password"
	config.WardenEnabled.Value = "false"
	config.HeraldEnabled.Value = "false"
	config.TrustedProxies.Value = "0.0.0.0,127.0.0.1,::1"
	storage := sessionkit.NewMemoryStorage("browser:revocable:", 0)
	t.Cleanup(func() { require.NoError(t, storage.Close()) })
	manager := sessionkit.NewManager(storage, sessionkit.DefaultConfig().WithCookieName(auth.SessionCookieName).WithSecure(secure))
	cfg, err := sessionfiber.RevocableSessionConfig(manager)
	require.NoError(t, err)
	store := session.NewStore(cfg)
	app := fiber.New(fiber.Config{Views: html.New("../web/templates", ".html"), TrustProxy: true, TrustProxyConfig: fiber.TrustProxyConfig{Proxies: []string{"0.0.0.0", "127.0.0.1", "::1"}}})
	app.Get("/_login", LoginRoute(store))
	app.Post("/_login", RequireSameOrigin(), LoginAPI(store))
	app.Get("/_session_exchange", SessionShareRoute(store))
	app.Get("/protected", CheckRoute(store))
	app.Get("/done", func(c fiber.Ctx) error {
		sess, err := store.Get(c)
		if err != nil {
			return err
		}
		defer sess.Release()
		if !auth.IsAuthenticated(sess) {
			return c.SendStatus(401)
		}
		return c.SendString(sess.ID())
	})
	proto := "http"
	if secure {
		proto = "https"
	}
	return &exchangeBrowserFixture{t: t, app: app, proto: proto}
}
func (f *exchangeBrowserFixture) client() *http.Client {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	require.NoError(f.t, err)
	return &http.Client{Transport: browserExchangeTransport{f.app}, Jar: jar}
}
func (f *exchangeBrowserFixture) address(host, path string) string {
	return f.proto + "://" + host + path
}
func (f *exchangeBrowserFixture) body(response *http.Response) string {
	f.t.Helper()
	body, err := io.ReadAll(response.Body)
	require.NoError(f.t, err)
	require.NoError(f.t, response.Body.Close())
	return string(body)
}
func (f *exchangeBrowserFixture) login(client *http.Client, target string) string {
	f.t.Helper()
	response, err := client.PostForm(f.address("auth.example.com", "/_login"), url.Values{"password": {"browser-password"}, "callback": {target}})
	require.NoError(f.t, err)
	require.Equal(f.t, 200, response.StatusCode)
	return f.body(response)
}
func (f *exchangeBrowserFixture) sessionCookie(client *http.Client, host string) string {
	f.t.Helper()
	address, err := url.Parse(f.address(host, "/"))
	require.NoError(f.t, err)
	for _, cookie := range client.Jar.Cookies(address) {
		if cookie.Name == auth.SessionCookieName {
			return cookie.Value
		}
	}
	return ""
}
func (f *exchangeBrowserFixture) noRedirect(client *http.Client, address string) *http.Response {
	f.t.Helper()
	original := client.CheckRedirect
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	defer func() { client.CheckRedirect = original }()
	response, err := client.Get(address)
	require.NoError(f.t, err)
	return response
}
func (f *exchangeBrowserFixture) pendingTicket(client *http.Client, target string) string {
	f.t.Helper()
	response := f.noRedirect(client, f.address(target, "/_session_exchange?start=1"))
	require.Equal(f.t, 302, response.StatusCode)
	authURL := response.Header.Get("Location")
	f.body(response)
	response = f.noRedirect(client, authURL)
	require.Equal(f.t, 302, response.StatusCode)
	ticketURL := response.Header.Get("Location")
	require.Contains(f.t, ticketURL, "ticket=")
	f.body(response)
	return ticketURL
}

func TestBrowserBoundExchangeSupportsDirectAndTargetInitiatedLogin(t *testing.T) {
	for _, secure := range []bool{false, true} {
		t.Run(fmt.Sprint(secure), func(t *testing.T) {
			fixture := newExchangeBrowserFixture(t, secure)
			direct := fixture.client()
			id := fixture.login(direct, "app.example.net")
			require.NotEmpty(t, id)
			require.Equal(t, id, fixture.sessionCookie(direct, "auth.example.com"))
			require.Equal(t, id, fixture.sessionCookie(direct, "app.example.net"))
			initiated := fixture.client()
			response, err := initiated.Get(fixture.address("other.example.org", "/_session_exchange?start=1&next=https://evil.example/"))
			require.NoError(t, err)
			require.Equal(t, 200, response.StatusCode)
			page := fixture.body(response)
			match := regexp.MustCompile(`name="exchange_state" value="([^"]+)"`).FindStringSubmatch(page)
			require.Len(t, match, 2)
			response, err = initiated.PostForm(fixture.address("auth.example.com", "/_login"), url.Values{"password": {"browser-password"}, "callback": {"other.example.org"}, exchangeStateParameter: {match[1]}})
			require.NoError(t, err)
			require.Equal(t, 200, response.StatusCode)
			otherID := fixture.body(response)
			require.NotEqual(t, id, otherID)
			require.Equal(t, otherID, fixture.sessionCookie(initiated, "other.example.org"))
			require.Equal(t, "/done", response.Request.URL.Path)
		})
	}
}

func TestExchangeCannotImportAnotherBrowsersAuthenticatedSession(t *testing.T) {
	fixture := newExchangeBrowserFixture(t, true)
	attacker, victim := fixture.client(), fixture.client()
	attackerID := fixture.login(attacker, "app.example.net")
	victimID := fixture.login(victim, "app.example.net")
	require.NotEqual(t, attackerID, victimID)
	ticketURL := fixture.pendingTicket(attacker, "app.example.net")
	response, err := victim.Get(ticketURL)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, victimID, fixture.body(response))
	require.Equal(t, victimID, fixture.sessionCookie(victim, "app.example.net"))
	response, err = attacker.Get(ticketURL)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, attackerID, fixture.body(response))
	response = fixture.noRedirect(victim, fixture.address("app.example.net", "/_session_exchange?start=1"))
	fixture.body(response)
	ticketURL = fixture.pendingTicket(attacker, "app.example.net")
	response, err = victim.Get(ticketURL)
	require.NoError(t, err)
	require.Equal(t, 401, response.StatusCode)
	fixture.body(response)
	require.Equal(t, victimID, fixture.sessionCookie(victim, "app.example.net"))
}

func TestExchangeNonceCookiesAreTargetSpecificAndHostOnly(t *testing.T) {
	fixture := newExchangeBrowserFixture(t, true)
	client := fixture.client()
	response := fixture.noRedirect(client, fixture.address("app.example.net", "/_session_exchange?start=1&nonce=attacker-controlled"))
	require.Equal(t, 302, response.StatusCode)
	cookies := response.Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "__Host-stargate_exchange_nonce", cookies[0].Name)
	require.Empty(t, cookies[0].Domain)
	require.Equal(t, "/", cookies[0].Path)
	require.True(t, cookies[0].Secure)
	require.True(t, cookies[0].HttpOnly)
	require.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
	other, _ := url.Parse(fixture.address("other.example.org", "/"))
	require.Empty(t, client.Jar.Cookies(other))
	destination, err := url.Parse(response.Header.Get("Location"))
	require.NoError(t, err)
	fixture.body(response)
	query := destination.Query()
	query.Set("callback", "other.example.org")
	destination.RawQuery = query.Encode()
	response, err = client.Get(destination.String())
	require.NoError(t, err)
	require.Equal(t, 400, response.StatusCode)
	fixture.body(response)
}

func TestBrowserExchangeRejectsLegacyTicket(t *testing.T) {
	fixture := newExchangeBrowserFixture(t, false)
	client := fixture.client()
	id := fixture.login(client, "app.example.net")
	target, _ := url.Parse(fixture.address("app.example.net", "/"))
	client.Jar.SetCookies(target, []*http.Cookie{{Name: exchangeCookieName(false), Value: testExchangeNonce, Path: "/"}})
	legacy, err := sealExchangeValue(sessionExchangeClaims{SessionID: "attacker", Audience: "app.example.net", ExpiresAt: 1 << 40}, nil)
	require.NoError(t, err)
	response, err := client.Get(fixture.address("app.example.net", "/_session_exchange?ticket="+legacy))
	require.NoError(t, err)
	require.Equal(t, 401, response.StatusCode)
	fixture.body(response)
	require.Equal(t, id, fixture.sessionCookie(client, "app.example.net"))
}

func TestBrowserExchangeKeepsOnlyConfiguredReturnPath(t *testing.T) {
	fixture := newExchangeBrowserFixture(t, true)
	client := fixture.client()
	response, err := client.PostForm(fixture.address("auth.example.com", "/_login"), url.Values{"password": {"browser-password"}, "callback": {"app.example.net"}, "next": {"https://evil.example/"}})
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.NotEmpty(t, fixture.body(response))
	require.Equal(t, "app.example.net", response.Request.URL.Host)
	require.Equal(t, "/done", response.Request.URL.Path)
}

func TestBrowserExchangeCompletesForwardAuthCallback(t *testing.T) {
	fixture := newExchangeBrowserFixture(t, true)
	client := fixture.client()
	request, err := http.NewRequest(http.MethodGet, fixture.address("app.example.net", "/protected"), nil)
	require.NoError(t, err)
	request.Header.Set("Accept", "text/html")
	response, err := client.Do(request)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, "auth.example.com", response.Request.URL.Host)
	require.Contains(t, fixture.body(response), `name="callback" value="app.example.net"`)
	response, err = client.PostForm(fixture.address("auth.example.com", "/_login"), url.Values{"password": {"browser-password"}, "callback": {"app.example.net"}})
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	require.Equal(t, "/done", response.Request.URL.Path)
	require.Equal(t, fixture.sessionCookie(client, "app.example.net"), fixture.body(response))
	request, err = http.NewRequest(http.MethodGet, fixture.address("app.example.net", "/protected"), nil)
	require.NoError(t, err)
	request.Header.Set("Accept", "text/html")
	response, err = client.Do(request)
	require.NoError(t, err)
	require.Equal(t, 200, response.StatusCode)
	fixture.body(response)
}
