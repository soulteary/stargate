# OIDC Support Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add OAuth2/OpenID Connect authentication to Stargate, allowing integration with enterprise SSO providers like Keycloak and Azure AD.

**Architecture:** When OIDC is enabled via environment variables, password authentication is disabled. Users click a login button, are redirected to the OIDC provider, and upon successful authentication, a session is created with user_id and email claims.

**Tech Stack:** Go 1.25, Fiber v2, coreos/go-oidc/v3, standard library crypto/rand

---

## Task 1: Add OIDC Configuration Variables

**Files:**
- Modify: `src/internal/config/config.go`
- Modify: `src/internal/config/validation.go`
- Test: `src/internal/config/config_test.go`

**Step 1: Write configuration structure test**

```go
// In src/internal/config/config_test.go

func TestOIDCConfigurationDefaults(t *testing.T) {
    // Reset environment
    os.Unsetenv("OIDC_ENABLED")
    os.Unsetenv("OIDC_ISSUER_URL")
    os.Unsetenv("OIDC_CLIENT_ID")
    os.Unsetenv("OIDC_CLIENT_SECRET")
    os.Unsetenv("OIDC_REDIRECT_URI")
    os.Unsetenv("OIDC_PROVIDER_NAME")

    Initialize()

    assert.Equal(t, "false", OIDC.Enabled.String())
    assert.Equal(t, "", OIDC.IssuerURL.Value)
    assert.Equal(t, "", OIDC.ClientID.Value)
    assert.Equal(t, "", OIDC.ClientSecret.Value)
    assert.Equal(t, "", OIDC.RedirectURI.Value)
    assert.Equal(t, "OIDC", OIDC.ProviderName.Value)
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test -v ./internal/config/... -run TestOIDCConfigurationDefaults`
Expected: FAIL with "undefined: OIDC"

**Step 3: Add OIDC configuration variables to config.go**

Add after the `Language` variable (line 78):

```go
// In src/internal/config/config.go

OIDCEnabled = EnvVariable{
    Name:           "OIDC_ENABLED",
    Required:       false,
    DefaultValue:   "false",
    PossibleValues: []string{"true", "false"},
    Validator:      ValidateCaseInsensitivePossibleValues,
}

OIDCIssuerURL = EnvVariable{
    Name:           "OIDC_ISSUER_URL",
    Required:       false,
    DefaultValue:   "",
    PossibleValues: []string{"*"},
    Validator:      ValidateAny,
}

OIDCClientID = EnvVariable{
    Name:           "OIDC_CLIENT_ID",
    Required:       false,
    DefaultValue:   "",
    PossibleValues: []string{"*"},
    Validator:      ValidateAny,
}

OIDCClientSecret = EnvVariable{
    Name:           "OIDC_CLIENT_SECRET",
    Required:       false,
    DefaultValue:   "",
    PossibleValues: []string{"*"},
    Validator:      ValidateAny,
}

OIDCRedirectURI = EnvVariable{
    Name:           "OIDC_REDIRECT_URI",
    Required:       false,
    DefaultValue:   "",
    PossibleValues: []string{"*"},
    Validator:      ValidateAny,
}

OIDCProviderName = EnvVariable{
    Name:           "OIDC_PROVIDER_NAME",
    Required:       false,
    DefaultValue:   "OIDC",
    PossibleValues: []string{"*"},
    Validator:      ValidateNotEmptyString,
}
```

**Step 4: Update Initialize function to validate OIDC config**

Modify the `Initialize()` function in `src/internal/config/config.go`:

```go
func Initialize() error {
    // First, initialize language setting
    Language.Validate()
    lang := strings.ToLower(Language.Value)
    switch lang {
    case "zh":
        i18n.SetLanguage(i18n.LangZH)
    case "fr":
        i18n.SetLanguage(i18n.LangFR)
    case "it":
        i18n.SetLanguage(i18n.LangIT)
    case "ja":
        i18n.SetLanguage(i18n.LangJA)
    case "de":
        i18n.SetLanguage(i18n.LangDE)
    case "ko":
        i18n.SetLanguage(i18n.LangKO)
    default:
        i18n.SetLanguage(i18n.LangEN)
    }

    // Validate OIDC configuration first (before PASSWORDS since it's mutually exclusive)
    OIDCEnabled.Validate()

    // Determine if OIDC is enabled
    oidcEnabled := strings.ToLower(OIDCEnabled.Value) == "true"

    // Build list of variables to validate based on authentication mode
    var envVariables []*EnvVariable

    if oidcEnabled {
        // OIDC mode: validate OIDC required fields
        envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &OIDCIssuerURL, &OIDCClientID, &OIDCClientSecret, &OIDCRedirectURI, &OIDCProviderName, &UserHeaderName, &CookieDomain}
    } else {
        // Password mode: validate PASSWORDS field
        envVariables = []*EnvVariable{&Debug, &AuthHost, &LoginPageTitle, &LoginPageFooterText, &Passwords, &UserHeaderName, &CookieDomain}
    }

    for _, variable := range envVariables {
        err := variable.Validate()
        if err != nil {
            return err
        }

        if variable.Value != "" {
            logrus.Info("Config: ", variable.Name, " = ", variable.Value)
        }
    }

    if Language.Value != "" {
        logrus.Info("Config: ", Language.Name, " = ", Language.Value)
    }

    return nil
}
```

**Step 5: Add validation for required OIDC fields**

Create `src/internal/config/oidc_validation.go`:

```go
package config

import (
    "fmt"
    "github.com/soulteary/stargate/src/internal/i18n"
)

// ValidateOIDCConfiguration validates that all required OIDC fields are set when enabled
func ValidateOIDCConfiguration() error {
    if !IsOIDCEnabled() {
        return nil
    }

    // Check required fields
    if OIDCIssuerURL.Value == "" {
        return fmt.Errorf(i18n.T("error.config_required"), "OIDC_ISSUER_URL")
    }
    if OIDCClientID.Value == "" {
        return fmt.Errorf(i18n.T("error.config_required"), "OIDC_CLIENT_ID")
    }
    if OIDCClientSecret.Value == "" {
        return fmt.Errorf(i18n.T("error.config_required"), "OIDC_CLIENT_SECRET")
    }

    return nil
}

// IsOIDCEnabled returns true if OIDC authentication is enabled
func IsOIDCEnabled() bool {
    return OIDCEnabled.String() == "true"
}

// GetOIDCProviderName returns the configured OIDC provider name
func GetOIDCProviderName() string {
    if OIDCProviderName.Value != "" {
        return OIDCProviderName.Value
    }
    return "OIDC"
}
```

**Step 6: Run tests to verify they pass**

Run: `cd src && go test -v ./internal/config/... -run TestOIDCConfigurationDefaults`
Expected: PASS

**Step 7: Commit**

```bash
git add src/internal/config/
git commit -m "feat: add OIDC configuration variables"
```

---

## Task 2: Create OIDC Package

**Files:**
- Create: `src/internal/oidc/oidc.go`
- Create: `src/internal/oidc/verifier.go`
- Create: `src/internal/oidc/claims.go`
- Create: `src/internal/oidc/oidc_test.go`

**Step 1: Write the failing test for provider initialization**

```go
// In src/internal/oidc/oidc_test.go

package oidc

import (
    "testing"
    "github.com/marvinjwendt/testza"
)

func TestNewProviderInvalidURL(t *testing.T) {
    _, err := NewProvider("invalid-url", "client-id", "client-secret")
    testza.AssertError(t, err)
}

func TestNewProviderMissingFields(t *testing.T) {
    _, err := NewProvider("", "client-id", "client-secret")
    testza.AssertError(t, err)
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test -v ./internal/oidc/... -run TestNewProvider`
Expected: FAIL with "package oidc does not exist"

**Step 3: Create the OIDC package structure**

Create `src/internal/oidc/oidc.go`:

```go
package oidc

import (
    "context"
    "errors"
    "fmt"

    "github.com/coreos/go-oidc/v3"
    "github.com/sirupsen/logrus"
    "golang.org/x/oauth2"
)

// Provider represents an OIDC provider configuration
type Provider struct {
    issuerURL      string
    clientID       string
    clientSecret   string
    redirectURI    string
    verifier       *oidc.IDTokenVerifier
    oauth2Config   *oauth2.Config
    provider       *oidc.Provider
}

// NewProvider creates a new OIDC provider instance
// It fetches and validates the provider's discovery document
func NewProvider(issuerURL, clientID, clientSecret, redirectURI string) (*Provider, error) {
    if issuerURL == "" {
        return nil, errors.New("issuer URL is required")
    }
    if clientID == "" {
        return nil, errors.New("client ID is required")
    }
    if clientSecret == "" {
        return nil, errors.New("client secret is required")
    }

    logrus.Infof("Initializing OIDC provider with issuer: %s", issuerURL)

    // Create provider with discovery
    ctx := context.Background()
    provider, err := oidc.NewProvider(ctx, issuerURL)
    if err != nil {
        return nil, fmt.Errorf("failed to create OIDC provider: %w", err)
    }

    // Create ID token verifier
    verifier := provider.Verifier(&oidc.Config{
        ClientID: clientID,
    })

    // Create OAuth2 config
    var oauth2Config *oauth2.Config
    if redirectURI != "" {
        oauth2Config = &oauth2.Config{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            RedirectURL:  redirectURI,
            Endpoint:     provider.Endpoint(),
            Scopes:       []string{oidc.ScopeOpenID, "email"},
        }
    } else {
        oauth2Config = &oauth2.Config{
            ClientID:     clientID,
            ClientSecret: clientSecret,
            Endpoint:     provider.Endpoint(),
            Scopes:       []string{oidc.ScopeOpenID, "email"},
        }
    }

    return &Provider{
        issuerURL:    issuerURL,
        clientID:     clientID,
        clientSecret: clientSecret,
        redirectURI:  redirectURI,
        verifier:     verifier,
        oauth2Config: oauth2Config,
        provider:     provider,
    }, nil
}

// AuthURL generates the OAuth2 authorization URL
func (p *Provider) AuthURL(state string) string {
    return p.oauth2Config.AuthCodeURL(state)
}

// Exchange exchanges the authorization code for an OAuth2 token
func (p *Provider) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
    return p.oauth2Config.Exchange(ctx, code)
}

// VerifyIDToken verifies the ID token and returns the claims
func (p *Provider) VerifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
    return p.verifier.Verify(ctx, rawIDToken)
}

// GetIssuer returns the issuer URL
func (p *Provider) GetIssuer() string {
    return p.issuerURL
}
```

Create `src/internal/oidc/claims.go`:

```go
package oidc

import (
    "errors"

    "github.com/coreos/go-oidc/v3"
)

// UserInfo represents the user information extracted from OIDC claims
type UserInfo struct {
    UserID string
    Email  string
}

// ExtractUserInfo extracts user information from ID token claims
func ExtractUserInfo(token *oidc.IDToken) (*UserInfo, error) {
    var claims struct {
        UserID string `json:"sub"`
        Email  string `json:"email"`
    }

    if err := token.Claims(&claims); err != nil {
        return nil, errors.New("failed to extract claims from token")
    }

    if claims.UserID == "" {
        return nil, errors.New("missing sub claim in token")
    }

    return &UserInfo{
        UserID: claims.UserID,
        Email:  claims.Email,
    }, nil
}
```

Create `src/internal/oidc/verifier.go`:

```go
package oidc

import (
    "context"
    "crypto/rand"
    "encoding/base64"

    "github.com/gofiber/fiber/v2/middleware/session"
)

// StateManager manages OAuth2 state parameters for CSRF protection
type StateManager struct{}

// NewStateManager creates a new state manager
func NewStateManager() *StateManager {
    return &StateManager{}
}

// GenerateState generates a cryptographically random state parameter
func (sm *StateManager) GenerateState() (string, error) {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateState validates the state parameter from the callback
func (sm *StateManager) ValidateState(sess *session.Session, state string) bool {
    storedState := sess.Get("oauth_state")
    if storedState == nil {
        return false
    }

    storedStateStr, ok := storedState.(string)
    if !ok {
        return false
    }

    // Clear the state after validation (single-use)
    sess.Delete("oauth_state")

    return storedStateStr == state
}

// SetState stores the state parameter in the session
func (sm *StateManager) SetState(sess *session.Session, state string) error {
    sess.Set("oauth_state", state)
    return sess.Save()
}

// GetUserInfoFromToken verifies the token and extracts user info
func (p *Provider) GetUserInfoFromToken(ctx context.Context, rawIDToken string) (*UserInfo, error) {
    idToken, err := p.VerifyIDToken(ctx, rawIDToken)
    if err != nil {
        return nil, err
    }

    return ExtractUserInfo(idToken)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd src && go test -v ./internal/oidc/...`
Expected: PASS (some tests may fail due to invalid URL, which is expected)

**Step 5: Add go-oidc dependency**

Run: `cd src && go get github.com/coreos/go-oidc/v3`

**Step 6: Run tests again**

Run: `cd src && go test -v ./internal/oidc/...`
Expected: PASS

**Step 7: Commit**

```bash
git add src/internal/oidc/
git commit -m "feat: add OIDC package with provider and token verification"
```

---

## Task 3: Add OIDC Session Management to Auth Package

**Files:**
- Modify: `src/internal/auth/auth.go`
- Modify: `src/internal/auth/auth_test.go`

**Step 1: Write the failing test**

```go
// In src/internal/auth/auth_test.go

func TestSetUserInfoInSession(t *testing.T) {
    // This test requires a session store, which we'll mock
    // For now, we'll test the function exists
    // Full integration tests will be added later
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test -v ./internal/auth/... -run TestSetUserInfoInSession`
Expected: FAIL with "undefined: SetUserInfoInSession"

**Step 3: Add user info functions to auth.go**

Add to `src/internal/auth/auth.go`:

```go
// GetUserID returns the user ID from the session
func GetUserID(session *session.Session) string {
    if userID := session.Get("user_id"); userID != nil {
        if str, ok := userID.(string); ok {
            return str
        }
    }
    return ""
}

// GetEmail returns the email from the session
func GetEmail(session *session.Session) string {
    if email := session.Get("email"); email != nil {
        if str, ok := email.(string); ok {
            return str
        }
    }
    return ""
}

// AuthenticateOIDC marks a session as authenticated with OIDC user info
func AuthenticateOIDC(session *session.Session, userID, email string) error {
    session.Set("authenticated", true)
    session.Set("user_id", userID)
    session.Set("email", email)
    session.Set("provider", "oidc")
    return session.Save()
}

// GetForwardedUserValue returns the value to use for X-Forwarded-User header
// Priority: user_id > email > "authenticated"
func GetForwardedUserValue(session *session.Session) string {
    if userID := GetUserID(session); userID != "" {
        return userID
    }
    if email := GetEmail(session); email != "" {
        return email
    }
    return "authenticated"
}
```

**Step 4: Run tests to verify they pass**

Run: `cd src && go test -v ./internal/auth/...`
Expected: PASS

**Step 5: Commit**

```bash
git add src/internal/auth/
git commit -m "feat: add OIDC user info to auth package"
```

---

## Task 4: Create OIDC Handlers

**Files:**
- Create: `src/internal/handlers/oidc.go`
- Modify: `src/internal/handlers/handlers_test.go`

**Step 1: Write the failing test for OIDC login handler**

```go
// In src/internal/handlers/handlers_test.go

func TestOIDCLoginHandler(t *testing.T) {
    // Test that handler generates state and redirects
    // This will be a basic structural test
    // Full integration tests require mock OIDC server
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test -v ./internal/handlers/... -run TestOIDCLoginHandler`
Expected: FAIL with "undefined: OIDCLoginHandler"

**Step 3: Create OIDC handlers**

Create `src/internal/handlers/oidc.go`:

```go
package handlers

import (
    "context"
    "fmt"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/session"
    "github.com/soulteary/stargate/src/internal/auth"
    "github.com/soulteary/stargate/src/internal/config"
    "github.com/soulteary/stargate/src/internal/i18n"
    "github.com/soulteary/stargate/src/internal/oidc"
)

var (
    oidcProvider     *oidc.Provider
    oidcStateManager *oidc.StateManager
)

// InitOIDC initializes the OIDC provider and state manager
// Call this during server initialization if OIDC is enabled
func InitOIDC() error {
    if !config.IsOIDCEnabled() {
        return nil
    }

    redirectURI := config.OIDCRedirect.Value
    if redirectURI == "" {
        // Auto-generate redirect URI from AUTH_HOST
        redirectURI = fmt.Sprintf("https://%s/_oidc/callback", config.AuthHost.Value)
    }

    var err error
    oidcProvider, err = oidc.NewProvider(
        config.OIDCIssuerURL.Value,
        config.OIDCClientID.Value,
        config.OIDCClientSecret.Value,
        redirectURI,
    )
    if err != nil {
        return fmt.Errorf("failed to initialize OIDC provider: %w", err)
    }

    oidcStateManager = oidc.NewStateManager()
    return nil
}

// OIDCLoginHandler initiates the OIDC login flow
func OIDCLoginHandler(store *session.Store) fiber.Handler {
    return func(ctx *fiber.Ctx) error {
        if oidcProvider == nil {
            return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T("error.oidc_not_configured"))
        }

        // Get session
        sess, err := store.Get(ctx)
        if err != nil {
            return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
        }

        // Generate state parameter
        state, err := oidcStateManager.GenerateState()
        if err != nil {
            return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.state_generation_failed"))
        }

        // Store state in session
        if err := oidcStateManager.SetState(sess, state); err != nil {
            return SendErrorResponse(ctx, fiber.StatusInternalServerError, i18n.T("error.session_store_failed"))
        }

        // Redirect to OIDC provider
        authURL := oidcProvider.AuthURL(state)
        return ctx.Redirect(authURL)
    }
}

// OIDCCallbackHandler handles the OIDC callback
func OIDCCallbackHandler(store *session.Store) fiber.Handler {
    return func(ctx *fiber.Ctx) error {
        if oidcProvider == nil {
            return SendErrorResponse(ctx, fiber.StatusServiceUnavailable, i18n.T("error.oidc_not_configured"))
        }

        // Get session
        sess, err := store.Get(ctx)
        if err != nil {
            return renderErrorPage(ctx, i18n.T("error.session_store_failed"))
        }

        // Get parameters
        code := ctx.Query("code")
        state := ctx.Query("state")

        if code == "" {
            return renderErrorPage(ctx, i18n.T("error.oidc_missing_code"))
        }

        // Validate state
        if !oidcStateManager.ValidateState(sess, state) {
            return renderErrorPage(ctx, i18n.T("error.oidc_invalid_state"))
        }

        // Exchange code for token
        token, err := oidcProvider.Exchange(context.Background(), code)
        if err != nil {
            return renderErrorPage(ctx, i18n.T("error.oidc_token_exchange_failed"))
        }

        // Get ID token
        rawIDToken, ok := token.Extra("id_token").(string)
        if !ok {
            return renderErrorPage(ctx, i18n.T("error.oidc_missing_id_token"))
        }

        // Verify token and extract user info
        userInfo, err := oidcProvider.GetUserInfoFromToken(context.Background(), rawIDToken)
        if err != nil {
            return renderErrorPage(ctx, i18n.T("error.oidc_token_verification_failed"))
        }

        // Create session
        if err := auth.AuthenticateOIDC(sess, userInfo.UserID, userInfo.Email); err != nil {
            return renderErrorPage(ctx, i18n.T("error.authenticate_failed"))
        }

        // Redirect to session exchange
        callbackURL := ctx.Query("callback")
        if callbackURL == "" {
            callbackURL = fmt.Sprintf("/_session_exchange?id=%s", sess.ID())
        } else {
            callbackURL = fmt.Sprintf("%s/_session_exchange?id=%s", callbackURL, sess.ID())
        }

        return ctx.Redirect(callbackURL)
    }
}

// renderErrorPage renders an error page with retry button
func renderErrorPage(ctx *fiber.Ctx, message string) error {
    if IsHTMLRequest(ctx) {
        // Return HTML error page
        return ctx.Status(fiber.StatusBadRequest).SendString(fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>%s</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; height: 100vh; margin: 0; background: #f5f5f5; }
        .container { text-align: center; background: white; padding: 2rem; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #e74c3c; }
        a { display: inline-block; margin-top: 1rem; padding: 0.5rem 1rem; background: #3498db; color: white; text-decoration: none; border-radius: 4px; }
        a:hover { background: #2980b9; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Authentication Error</h1>
        <p>%s</p>
        <a href="/_login">Retry</a>
    </div>
</body>
</html>
        `, i18n.T("error.oidc_error"), message))
    }

    return SendErrorResponse(ctx, fiber.StatusBadRequest, message)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd src && go test -v ./internal/handlers/... -run TestOIDCLoginHandler`
Expected: PASS

**Step 5: Commit**

```bash
git add src/internal/handlers/oidc.go
git commit -m "feat: add OIDC login and callback handlers"
```

---

## Task 5: Update CheckRoute to Use User Info

**Files:**
- Modify: `src/internal/handlers/check.go`

**Step 1: Write the failing test**

```go
// In src/internal/handlers/handlers_test.go

func TestCheckRouteWithOIDCSession(t *testing.T) {
    // Test that X-Forwarded-User contains user_id when OIDC is used
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test -v ./internal/handlers/... -run TestCheckRouteWithOIDCSession`
Expected: FAIL with "X-Forwarded-User does not contain user_id"

**Step 3: Update CheckRoute to use user info**

Modify `src/internal/handlers/check.go`:

```go
// In the CheckRoute function, replace the hardcoded "authenticated" values

// Find these lines (around line 54-55 and line 66-67):
// Old code:
// ctx.Set(userHeaderName, "authenticated")

// Replace with:
userValue := auth.GetForwardedUserValue(sess)
ctx.Set(userHeaderName, userValue)
```

**Step 4: Run tests to verify they pass**

Run: `cd src && go test -v ./internal/handlers/... -run TestCheckRouteWithOIDCSession`
Expected: PASS

**Step 5: Commit**

```bash
git add src/internal/handlers/check.go
git commit -m "feat: use user_id/email from session in X-Forwarded-User header"
```

---

## Task 6: Update Login Route for OIDC Mode

**Files:**
- Modify: `src/internal/handlers/login.go`

**Step 1: Write the failing test**

```go
// In src/internal/handlers/handlers_test.go

func TestLoginRouteOIDCMode(t *testing.T) {
    // Test that login page shows button in OIDC mode
}
```

**Step 2: Run test to verify it fails**

Run: `cd src && go test -v ./internal/handlers/... -run TestLoginRouteOIDCMode`
Expected: FAIL with "login page does not show OIDC button"

**Step 3: Update LoginRoute to check OIDC mode**

Modify the GET handler in `src/internal/handlers/login.go`:

```go
// In LoginRoute function, add OIDC check at the beginning of GET handler

func LoginRoute(store *session.Store) fiber.Handler {
    return func(ctx *fiber.Ctx) error {
        // ... existing session checks ...

        if ctx.Method() == fiber.MethodGet {
            // Check if already authenticated
            if auth.IsAuthenticated(sess) {
                // ... existing redirect logic ...
            }

            // Check if OIDC is enabled
            if config.IsOIDCEnabled() {
                return renderOIDCLoginPage(ctx)
            }

            // ... existing password login page render ...
        }

        if ctx.Method() == fiber.MethodPost {
            // Check if OIDC is enabled (deny password login)
            if config.IsOIDCEnabled() {
                return SendErrorResponse(ctx, fiber.StatusMethodNotAllowed, i18n.T("error.password_login_disabled"))
            }

            // ... existing password login logic ...
        }

        // ... existing error handling ...
    }
}

// Add new function to render OIDC login page
func renderOIDCLoginPage(ctx *fiber.Ctx) error {
    providerName := config.GetOIDCProviderName()

    // Render OIDC login button page
    html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="%s">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            margin: 0;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
        }
        .container {
            text-align: center;
            background: white;
            padding: 3rem;
            border-radius: 12px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.2);
        }
        h1 { color: #333; margin-bottom: 2rem; }
        .login-btn {
            display: inline-block;
            padding: 0.75rem 2rem;
            background: #667eea;
            color: white;
            text-decoration: none;
            border-radius: 6px;
            font-size: 1rem;
            font-weight: 500;
            transition: background 0.2s;
        }
        .login-btn:hover { background: #5568d3; }
        .footer { margin-top: 2rem; color: #666; font-size: 0.875rem; }
    </style>
</head>
<body>
    <div class="container">
        <h1>%s</h1>
        <a href="/_oidc/login" class="login-btn">%s</a>
        <div class="footer">%s</div>
    </div>
</body>
</html>`,
        i18n.GetLanguage(),
        config.LoginPageTitle.Value,
        config.LoginPageTitle.Value,
        fmt.Sprintf(i18n.T("login.oidc_button"), providerName),
        config.LoginPageFooterText.Value,
    )

    ctx.Set("Content-Type", "text/html; charset=utf-8")
    return ctx.SendString(html)
}
```

**Step 4: Run tests to verify they pass**

Run: `cd src && go test -v ./internal/handlers/... -run TestLoginRouteOIDCMode`
Expected: PASS

**Step 5: Commit**

```bash
git add src/internal/handlers/login.go
git commit -m "feat: add OIDC login page"
```

---

## Task 7: Add OIDC Routes to Server

**Files:**
- Modify: `src/cmd/stargate/server.go`

**Step 1: Write the failing test**

```bash
# Start server with OIDC enabled and verify routes exist
curl -I http://localhost:8080/_oidc/login
# Expected: 302 redirect (or 404 if not yet implemented)
```

**Step 2: Run test to verify it fails**

Expected: 404 Not Found

**Step 3: Add OIDC routes to server.go**

In `src/cmd/stargate/server.go`, add routes:

```go
// After existing route definitions, add:

// OIDC routes (only if OIDC is enabled)
if config.IsOIDCEnabled() {
    // Initialize OIDC provider
    if err := handlers.InitOIDC(); err != nil {
        logrus.Fatalf("Failed to initialize OIDC: %v", err)
    }

    // OIDC authentication flow
    app.Get("/_oidc/login", handlers.OIDCLoginHandler(sessionStore))
    app.Get("/_oidc/callback", handlers.OIDCCallbackHandler(sessionStore))
}
```

**Step 4: Run tests to verify they pass**

Run: Start server and test routes
Expected: Routes work correctly

**Step 5: Commit**

```bash
git add src/cmd/stargate/server.go
git commit -m "feat: register OIDC routes"
```

---

## Task 8: Add i18n Strings for OIDC

**Files:**
- Modify: `src/internal/i18n/i18n.go`

**Step 1: Add i18n strings**

Add to each language map in `src/internal/i18n/i18n.go`:

```go
// For English (LangEN map, add after existing keys):
"error.oidc_not_configured":           "OIDC is not configured",
"error.state_generation_failed":       "Failed to generate security token",
"error.oidc_missing_code":             "Authorization code is missing",
"error.oidc_invalid_state":            "Invalid security token",
"error.oidc_token_exchange_failed":    "Failed to exchange authorization code",
"error.oidc_missing_id_token":         "ID token is missing from response",
"error.oidc_token_verification_failed": "Failed to verify authentication token",
"error.password_login_disabled":       "Password login is disabled when OIDC is enabled",
"error.oidc_error":                    "Authentication Error",
"login.oidc_button":                   "Login with %s",

// For Chinese (LangZH map):
"error.oidc_not_configured":           "OIDC 未配置",
"error.state_generation_failed":       "生成安全令牌失败",
"error.oidc_missing_code":             "缺少授权码",
"error.oidc_invalid_state":            "无效的安全令牌",
"error.oidc_token_exchange_failed":    "交换授权码失败",
"error.oidc_missing_id_token":         "响应中缺少 ID 令牌",
"error.oidc_token_verification_failed": "验证认证令牌失败",
"error.password_login_disabled":       "启用 OIDC 时禁用密码登录",
"error.oidc_error":                    "认证错误",
"login.oidc_button":                   "使用 %s 登录",

// Add similar translations for fr, it, ja, de, ko
```

**Step 2: Commit**

```bash
git add src/internal/i18n/i18n.go
git commit -m "feat: add i18n strings for OIDC"
```

---

## Task 9: Update go.mod and Run Full Tests

**Files:**
- Modify: `go.mod`
- Test: All tests

**Step 1: Update go.mod with new dependency**

```bash
cd src
go get github.com/coreos/go-oidc/v3@v3.10.0
go mod tidy
```

**Step 2: Run all tests**

```bash
cd src
go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
```

**Step 3: Fix any test failures**

If tests fail, fix and re-run.

**Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add go-oidc dependency"
```

---

## Task 10: Update Documentation

**Files:**
- Create: `docs/enUS/OIDC.md`
- Create: `docs/zhCN/OIDC.md`
- Modify: `README.md` (optional)

**Step 1: Create OIDC documentation**

Create `docs/enUS/OIDC.md`:

```markdown
# OIDC Authentication

Stargate supports OpenID Connect (OIDC) authentication for integration with enterprise SSO providers.

## Configuration

Enable OIDC by setting the following environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `OIDC_ENABLED` | Yes | `false` | Set to `true` to enable OIDC |
| `OIDC_ISSUER_URL` | Yes* | - | Your OIDC provider's issuer URL |
| `OIDC_CLIENT_ID` | Yes* | - | Client ID from your OIDC provider |
| `OIDC_CLIENT_SECRET` | Yes* | - | Client secret from your OIDC provider |
| `OIDC_REDIRECT_URI` | No | Auto-generated | Callback URL for your application |
| `OIDC_PROVIDER_NAME` | No | `OIDC` | Display name for login button |

*Required when `OIDC_ENABLED=true`

## Example docker-compose.yml

```yaml
services:
  stargate:
    image: stargate:latest
    environment:
      - AUTH_HOST=auth.example.com
      - OIDC_ENABLED=true
      - OIDC_ISSUER_URL=https://keycloak.example.com/realms/myrealm
      - OIDC_CLIENT_ID=stargate-client
      - OIDC_CLIENT_SECRET=your-secret
      - OIDC_PROVIDER_NAME=Company Account
```

## Supported Providers

Any OIDC-compliant provider should work, including:
- Keycloak
- Azure AD / Entra ID
- Auth0
- Okta
- Google Workspace
- Self-hosted OIDC servers

## Notes

- When OIDC is enabled, password authentication is automatically disabled
- Only `openid` and `email` scopes are requested
- User ID (sub claim) and email are stored in the session
- The `X-Forwarded-User` header contains the user ID or email
```

Create `docs/zhCN/OIDC.md` with Chinese translation.

**Step 2: Commit**

```bash
git add docs/
git commit -m "docs: add OIDC authentication documentation"
```

---

## Testing Checklist

After implementation, verify:

- [ ] Configuration validation works (missing required fields causes startup failure)
- [ ] Login page shows OIDC button when enabled
- [ ] Login page shows password form when OIDC disabled
- [ ] Clicking login button redirects to OIDC provider
- [ ] Callback after authentication creates session
- [ ] `/_auth` endpoint returns user info in header
- [ ] Password login is disabled when OIDC enabled
- [ ] Error pages display correctly
- [ ] i18n works for all supported languages
- [ ] Existing password authentication still works when OIDC disabled

---

## Execution Summary

Total tasks: 10
Estimated time: 2-3 hours

Use `superpowers:subagent-driven-development` or `superpowers:executing-plans` to implement this plan step by step.
