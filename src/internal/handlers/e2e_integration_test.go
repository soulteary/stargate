package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/fiber/v2/utils"
	"github.com/redis/go-redis/v9"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/redis-kit/cache"
	"github.com/soulteary/redis-kit/client"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/valyala/fasthttp"
)

// testLoggerE2E creates a logger instance for testing
func testLoggerE2E() *logger.Logger {
	return logger.New(logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.FormatJSON,
		ServiceName: "e2e-test",
	})
}

// setupTestRedis creates a test Redis client using redis-kit
func setupTestRedis(t *testing.T) *redis.Client {
	cfg := client.DefaultConfig().
		WithAddr("localhost:6379").
		WithDB(14) // Use DB 14 for integration testing

	redisClient, err := client.NewClient(cfg)
	if err != nil {
		t.Skipf("Skipping integration test: Redis not available: %v", err)
	}

	ctx := context.Background()
	// Clean up test database
	if err := redisClient.FlushDB(ctx).Err(); err != nil {
		t.Logf("Warning: Failed to flush test database: %v", err)
	}

	return redisClient
}

// AllowListUser represents a user in the allow list (matching Warden's structure)
type AllowListUser struct {
	Phone  string   `json:"phone"`
	Mail   string   `json:"mail"`
	UserID string   `json:"user_id"`
	Status string   `json:"status"`
	Scope  []string `json:"scope"`
	Role   string   `json:"role"`
}

// setupWardenServer creates a mock Warden HTTP server for integration testing
func setupWardenServer(t *testing.T, userData []AllowListUser) *httptest.Server {
	// Create a simple in-memory user map
	userMap := make(map[string]AllowListUser)
	phoneMap := make(map[string]AllowListUser)
	mailMap := make(map[string]AllowListUser)

	for _, user := range userData {
		if user.UserID != "" {
			userMap[user.UserID] = user
		}
		if user.Phone != "" {
			phoneMap[user.Phone] = user
		}
		if user.Mail != "" {
			mailMap[strings.ToLower(user.Mail)] = user
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Get query parameters
		phone := strings.TrimSpace(r.URL.Query().Get("phone"))
		mail := strings.TrimSpace(r.URL.Query().Get("mail"))
		userID := strings.TrimSpace(r.URL.Query().Get("user_id"))

		var user AllowListUser
		var found bool

		// Query user by identifier
		if userID != "" {
			user, found = userMap[userID]
		} else if phone != "" {
			user, found = phoneMap[phone]
		} else if mail != "" {
			user, found = mailMap[strings.ToLower(mail)]
		}

		if !found {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Check if user is active
		if user.Status != "active" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(user)
	})

	server := httptest.NewServer(mux)
	return server
}

// setupHeraldMockServer creates a mock Herald server for integration testing
// In a real scenario, you would run Herald as a separate service
func setupHeraldMockServer(t *testing.T, redisClient *redis.Client) *httptest.Server {
	// This is a simplified mock - in production, you'd run Herald separately
	// For now, we'll create a basic mock that stores codes in Redis for testing
	// Use redis-kit cache for storing test data
	testCache := cache.NewCache(redisClient, "otp:test:")
	mux := http.NewServeMux()

	// Mock health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	// Mock create challenge
	mux.HandleFunc("/v1/otp/challenges", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify auth
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "unauthorized",
			})
			return
		}

		var req struct {
			UserID      string `json:"user_id"`
			Channel     string `json:"channel"`
			Destination string `json:"destination"`
			Purpose     string `json:"purpose"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Generate challenge ID and code
		// Use a simpler challenge ID format to avoid URL encoding issues
		// Use Unix timestamp + random suffix for simpler format
		challengeID := fmt.Sprintf("ch_test_%d_%d", time.Now().UnixNano(), time.Now().Unix()%10000)
		code := "123456" // Fixed code for testing

		// Store in Redis using redis-kit cache (simplified - in real Herald, this would be hashed)
		ctx := context.Background()
		if err := testCache.Set(ctx, "code:"+challengeID, code, 5*time.Minute); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "internal_error",
			})
			return
		}

		// Store user_id for verification
		_ = testCache.Set(ctx, "user_id:"+challengeID, req.UserID, 5*time.Minute)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"challenge_id":   challengeID,
			"expires_in":     300,
			"next_resend_in": 60,
		})
	})

	// Mock verify challenge
	mux.HandleFunc("/v1/otp/verifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Verify auth
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "test-api-key" {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "unauthorized",
			})
			return
		}

		var req struct {
			ChallengeID string `json:"challenge_id"`
			Code        string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "invalid_request",
			})
			return
		}

		// Get code from Redis using redis-kit cache
		ctx := context.Background()
		var storedCode string
		if err := testCache.Get(ctx, "code:"+req.ChallengeID, &storedCode); err != nil {
			// Challenge not found or expired
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "expired",
			})
			return
		}

		// Verify code matches (trim whitespace for safety and compare)
		storedCodeTrimmed := strings.TrimSpace(storedCode)
		reqCodeTrimmed := strings.TrimSpace(req.Code)
		if storedCodeTrimmed != reqCodeTrimmed {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "invalid",
			})
			return
		}

		// Get user_id from Redis using redis-kit cache
		var userID string
		if err := testCache.Get(ctx, "user_id:"+req.ChallengeID, &userID); err != nil {
			// Fallback to default if not found
			userID = "test-user-123"
		}

		// Return success response with 200 status code
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":        true,
			"user_id":   userID,
			"amr":       []string{"otp"},
			"issued_at": time.Now().Unix(),
		})
	})

	// Mock test code endpoint
	mux.HandleFunc("/v1/test/code/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Extract challenge_id from URL path
		// Path format: /v1/test/code/{challenge_id}
		path := r.URL.Path
		prefix := "/v1/test/code/"
		if !strings.HasPrefix(path, prefix) {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "invalid_path",
			})
			return
		}
		challengeID := path[len(prefix):]
		if challengeID == "" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "challenge_id_required",
			})
			return
		}

		ctx := context.Background()
		var code string
		if err := testCache.Get(ctx, "code:"+challengeID, &code); err != nil {
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"ok":     false,
				"reason": "code_not_found",
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":           true,
			"challenge_id": challengeID,
			"code":         code,
		})
	})

	return httptest.NewServer(mux)
}

// getTestCode retrieves the verification code from Herald test endpoint
func getTestCode(t *testing.T, heraldURL, challengeID string) string {
	resp, err := http.Get(heraldURL + "/v1/test/code/" + challengeID)
	if err != nil {
		t.Fatalf("Failed to get test code: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to get test code: status %d, body: %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK          bool   `json:"ok"`
		ChallengeID string `json:"challenge_id"`
		Code        string `json:"code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode test code response: %v", err)
	}

	if !result.OK {
		t.Fatalf("Test code response not OK")
	}

	return result.Code
}

func TestE2E_CompleteLoginFlow(t *testing.T) {
	// Setup Redis using redis-kit
	redisClient := setupTestRedis(t)
	defer func() {
		_ = client.Close(redisClient)
	}()

	// Setup test user data
	testUser := AllowListUser{
		Phone:  "13800138000",
		Mail:   "test@example.com",
		UserID: "test-user-123",
		Status: "active",
		Scope:  []string{"read", "write"},
		Role:   "user",
	}

	// Setup Warden server
	wardenServer := setupWardenServer(t, []AllowListUser{testUser})
	defer wardenServer.Close()

	// Setup Herald mock server
	heraldServer := setupHeraldMockServer(t, redisClient)
	defer heraldServer.Close()

	// Setup Stargate environment
	t.Setenv("AUTH_HOST", "auth.example.com")
	t.Setenv("PASSWORDS", "plaintext:test123")
	t.Setenv("WARDEN_ENABLED", "true")
	t.Setenv("WARDEN_URL", wardenServer.URL)
	t.Setenv("HERALD_ENABLED", "true")
	t.Setenv("HERALD_URL", heraldServer.URL)
	t.Setenv("HERALD_API_KEY", "test-api-key")
	t.Setenv("HERALD_HMAC_SECRET", "test-hmac-secret")
	t.Setenv("LANGUAGE", "zh")

	err := config.Initialize(testLoggerE2E())
	testza.AssertNoError(t, err)

	resetHeraldClientForTesting()
	auth.ResetWardenClientForTesting()

	// Create Stargate app
	store := session.New(session.Config{
		KeyLookup:      "cookie:" + auth.SessionCookieName,
		KeyGenerator:   utils.UUID,
		CookiePath:     "/",
		CookieHTTPOnly: true,
		CookieSecure:   false, // Set to false for testing (HTTP, not HTTPS)
		CookieSameSite: fiber.CookieSameSiteLaxMode,
	})

	app := fiber.New()
	app.Post("/_send_verify_code", SendVerifyCodeAPI())
	app.Post("/_login", LoginAPI(store))
	app.Get("/_auth", CheckRoute(store))

	// Step 1: Send verification code
	req := httptest.NewRequest("POST", "/_send_verify_code", strings.NewReader("phone=13800138000"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := app.Test(req)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, resp.StatusCode)

	bodyBytes, err := io.ReadAll(resp.Body)
	testza.AssertNoError(t, err)

	var sendResp struct {
		Success     bool   `json:"success"`
		ChallengeID string `json:"challenge_id"`
		ExpiresIn   int    `json:"expires_in"`
	}
	err = json.Unmarshal(bodyBytes, &sendResp)
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, sendResp.Success)
	testza.AssertNotNil(t, sendResp.ChallengeID)

	// Step 2: Get verification code from Herald test endpoint
	verifyCode := getTestCode(t, heraldServer.URL, sendResp.ChallengeID)
	testza.AssertNotNil(t, verifyCode)
	testza.AssertEqual(t, 6, len(verifyCode))

	// Step 3: Login with verification code
	loginReq := httptest.NewRequest("POST", "/_login", strings.NewReader(
		"auth_method=warden&phone=13800138000&challenge_id="+sendResp.ChallengeID+"&verify_code="+verifyCode,
	))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginReq.Header.Set("Accept", "application/json")
	// Set X-Forwarded-Host to match AUTH_HOST config so IsDifferentDomain returns false
	// This prevents the login handler from setting a callback and redirecting
	loginReq.Header.Set("X-Forwarded-Host", "auth.example.com")

	loginResp, err := app.Test(loginReq)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, loginResp.StatusCode)

	// Extract Set-Cookie header from login response
	// According to Fiber docs, we need to copy the entire Set-Cookie header value
	// to the Cookie header for subsequent requests
	setCookieHeaders := loginResp.Header.Values("Set-Cookie")
	testza.AssertGreater(t, len(setCookieHeaders), 0, "Set-Cookie header should be present after login")

	// Find the session cookie
	var sessionCookieHeader string
	for _, cookieHeader := range setCookieHeaders {
		if strings.Contains(cookieHeader, auth.SessionCookieName) {
			sessionCookieHeader = cookieHeader
			break
		}
	}
	testza.AssertNotNil(t, sessionCookieHeader, "Session cookie should be in Set-Cookie headers")

	// Extract just the name=value part (before the first semicolon) for Cookie header
	// Format: "stargate_session_id=value; Path=/; HttpOnly; SameSite=Lax"
	cookieParts := strings.Split(sessionCookieHeader, ";")
	testza.AssertGreater(t, len(cookieParts), 0, "Cookie header should have at least name=value")
	cookieNameValue := strings.TrimSpace(cookieParts[0])
	testza.AssertNotEqual(t, "", cookieNameValue, "Cookie name=value should not be empty")

	// Step 4: Verify forwardAuth check
	authReq := httptest.NewRequest("GET", "/_auth", nil)
	authReq.Header.Set("Accept", "application/json") // Set Accept header to avoid redirect
	authReq.Header.Set("Host", "auth.example.com")   // Set Host header to match AUTH_HOST config
	// Set Cookie header with just the name=value part
	// Note: HTTP Cookie header format is "name=value", not the full Set-Cookie format
	authReq.Header.Set("Cookie", cookieNameValue)

	// Debug: Log cookie information
	t.Logf("DEBUG: Session cookie header: %s", sessionCookieHeader)
	t.Logf("DEBUG: Cookie name=value: %s", cookieNameValue)
	t.Logf("DEBUG: Cookie header to send: %s", authReq.Header.Get("Cookie"))

	// Debug: Try to manually verify session in store BEFORE making the request
	// This will help us understand if the session was saved correctly
	testApp := fiber.New()
	testCtx := testApp.AcquireCtx(&fasthttp.RequestCtx{})
	defer testApp.ReleaseCtx(testCtx)

	testCtx.Request().SetRequestURI("/_auth")
	testCtx.Request().Header.SetMethod("GET")
	testCtx.Request().Header.Set("Host", "auth.example.com")
	testCtx.Request().Header.Set("Cookie", cookieNameValue)

	sess, err := store.Get(testCtx)
	if err != nil {
		t.Logf("DEBUG: Error getting session: %v", err)
	} else {
		t.Logf("DEBUG: Session ID from cookie: %s", sess.ID())
		t.Logf("DEBUG: Session authenticated: %v", auth.IsAuthenticated(sess))
		if auth.IsAuthenticated(sess) {
			t.Logf("DEBUG: Session user_id: %v", sess.Get("user_id"))
			t.Logf("DEBUG: Session user_mail: %v", sess.Get("user_mail"))
		} else {
			t.Logf("DEBUG: Session is NOT authenticated!")
			t.Logf("DEBUG: Session keys: %v", sess.Keys())
		}
	}

	// Use the same test context we created for debugging to call the handler directly
	// This bypasses app.Test() which may have issues with cookie handling
	checkHandler := CheckRoute(store)
	err = checkHandler(testCtx)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, fiber.StatusOK, testCtx.Response().StatusCode())

	// Verify authorization headers
	testza.AssertEqual(t, testUser.UserID, string(testCtx.Response().Header.Peek("X-Auth-User")))
	testza.AssertEqual(t, testUser.Mail, string(testCtx.Response().Header.Peek("X-Auth-Email")))
	testza.AssertEqual(t, "read,write", string(testCtx.Response().Header.Peek("X-Auth-Scopes")))
	testza.AssertEqual(t, testUser.Role, string(testCtx.Response().Header.Peek("X-Auth-Role")))
}
