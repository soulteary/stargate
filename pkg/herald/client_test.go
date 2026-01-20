package herald

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MarvinJWendt/testza"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()
	testza.AssertEqual(t, 10*time.Second, opts.Timeout)
	testza.AssertEqual(t, "stargate", opts.Service)
	testza.AssertEqual(t, "", opts.BaseURL)
}

func TestOptionsValidate(t *testing.T) {
	err := (&Options{}).Validate()
	testza.AssertNotNil(t, err)

	opts := &Options{BaseURL: "http://example.com"}
	testza.AssertNoError(t, opts.Validate())
}

func TestOptionsFluentSetters(t *testing.T) {
	opts := DefaultOptions().
		WithBaseURL("http://example.com").
		WithAPIKey("api-key").
		WithHMACSecret("hmac-secret").
		WithService("custom-service").
		WithTimeout(3 * time.Second)

	testza.AssertEqual(t, "http://example.com", opts.BaseURL)
	testza.AssertEqual(t, "api-key", opts.APIKey)
	testza.AssertEqual(t, "hmac-secret", opts.HMACSecret)
	testza.AssertEqual(t, "custom-service", opts.Service)
	testza.AssertEqual(t, 3*time.Second, opts.Timeout)
}

func TestNewClient_MissingBaseURL(t *testing.T) {
	client, err := NewClient(&Options{})
	testza.AssertNil(t, client)
	testza.AssertNotNil(t, err)
}

func TestNewClient_Success(t *testing.T) {
	opts := DefaultOptions().
		WithBaseURL("http://example.com").
		WithAPIKey("api-key").
		WithHMACSecret("hmac-secret").
		WithService("custom-service").
		WithTimeout(5 * time.Second)

	client, err := NewClient(opts)
	testza.AssertNoError(t, err)
	testza.AssertNotNil(t, client)
	testza.AssertEqual(t, "http://example.com", client.baseURL)
	testza.AssertEqual(t, "api-key", client.apiKey)
	testza.AssertEqual(t, "hmac-secret", client.hmacSecret)
	testza.AssertEqual(t, "custom-service", client.service)
	testza.AssertEqual(t, 5*time.Second, client.httpClient.Timeout)
}

func TestAddAuthHeaders_APIKeyOnly(t *testing.T) {
	client := &Client{
		apiKey:  "api-key",
		service: "stargate",
	}

	req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
	testza.AssertNoError(t, err)

	client.addAuthHeaders(req, []byte(`{"ok":true}`))

	testza.AssertEqual(t, "api-key", req.Header.Get("X-API-Key"))
	testza.AssertEqual(t, "", req.Header.Get("X-Timestamp"))
	testza.AssertEqual(t, "", req.Header.Get("X-Signature"))
	testza.AssertEqual(t, "", req.Header.Get("X-Service"))
}

func TestAddAuthHeaders_HMACOnly(t *testing.T) {
	body := []byte(`{"ok":true}`)
	client := &Client{
		hmacSecret: "hmac-secret",
		service:    "custom-service",
	}

	req, err := http.NewRequest(http.MethodPost, "http://example.com", nil)
	testza.AssertNoError(t, err)

	client.addAuthHeaders(req, body)

	timestamp := req.Header.Get("X-Timestamp")
	service := req.Header.Get("X-Service")
	signature := req.Header.Get("X-Signature")

	testza.AssertNotNil(t, timestamp)
	testza.AssertEqual(t, "custom-service", service)
	expectedSig := client.computeHMAC(timestamp, service, body)
	testza.AssertEqual(t, expectedSig, signature)
}

func TestComputeHMAC(t *testing.T) {
	client := &Client{hmacSecret: "hmac-secret"}
	timestamp := "1700000000"
	service := "stargate"
	body := []byte("payload")

	signature := client.computeHMAC(timestamp, service, body)

	mac := hmac.New(sha256.New, []byte("hmac-secret"))
	message := timestamp + ":" + service + ":" + string(body)
	mac.Write([]byte(message))
	expected := hex.EncodeToString(mac.Sum(nil))

	testza.AssertEqual(t, expected, signature)
}

func TestCreateChallenge_Success(t *testing.T) {
	expectedReq := &CreateChallengeRequest{
		UserID:      "user-1",
		Channel:     "sms",
		Destination: "13800138000",
		Purpose:     "login",
		Locale:      "zh-CN",
		ClientIP:    "127.0.0.1",
		UA:          "stargate-test",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testza.AssertEqual(t, http.MethodPost, r.Method)
		testza.AssertEqual(t, "/v1/otp/challenges", r.URL.Path)
		testza.AssertEqual(t, "application/json", r.Header.Get("Content-Type"))
		testza.AssertEqual(t, "api-key", r.Header.Get("X-API-Key"))

		bodyBytes, err := io.ReadAll(r.Body)
		testza.AssertNoError(t, err)

		timestamp := r.Header.Get("X-Timestamp")
		service := r.Header.Get("X-Service")
		signature := r.Header.Get("X-Signature")
		testza.AssertEqual(t, "stargate", service)

		expectedSig := (&Client{hmacSecret: "hmac-secret"}).computeHMAC(timestamp, service, bodyBytes)
		testza.AssertEqual(t, expectedSig, signature)

		var got CreateChallengeRequest
		err = json.Unmarshal(bodyBytes, &got)
		testza.AssertNoError(t, err)
		testza.AssertEqual(t, expectedReq, &got)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(CreateChallengeResponse{
			ChallengeID:  "challenge-1",
			ExpiresIn:    120,
			NextResendIn: 30,
		})
	}))
	defer server.Close()

	opts := DefaultOptions().
		WithBaseURL(server.URL).
		WithAPIKey("api-key").
		WithHMACSecret("hmac-secret").
		WithService("stargate")
	client, err := NewClient(opts)
	testza.AssertNoError(t, err)

	resp, err := client.CreateChallenge(context.Background(), expectedReq)
	testza.AssertNoError(t, err)
	testza.AssertEqual(t, "challenge-1", resp.ChallengeID)
	testza.AssertEqual(t, 120, resp.ExpiresIn)
	testza.AssertEqual(t, 30, resp.NextResendIn)
}

func TestCreateChallenge_StatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	testza.AssertNoError(t, err)

	_, err = client.CreateChallenge(context.Background(), &CreateChallengeRequest{
		UserID:      "user-1",
		Channel:     "sms",
		Destination: "13800138000",
	})
	testza.AssertNotNil(t, err)
	testza.AssertTrue(t, strings.Contains(err.Error(), "status 400"))
	testza.AssertTrue(t, strings.Contains(err.Error(), "bad request"))
}

func TestCreateChallenge_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	testza.AssertNoError(t, err)

	_, err = client.CreateChallenge(context.Background(), &CreateChallengeRequest{
		UserID:      "user-1",
		Channel:     "sms",
		Destination: "13800138000",
	})
	testza.AssertNotNil(t, err)
}

func TestVerifyChallenge_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		testza.AssertEqual(t, http.MethodPost, r.Method)
		testza.AssertEqual(t, "/v1/otp/verifications", r.URL.Path)
		testza.AssertEqual(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(VerifyChallengeResponse{
			OK:       true,
			UserID:   "user-1",
			AMR:      []string{"sms"},
			IssuedAt: 1700000000,
		})
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	testza.AssertNoError(t, err)

	resp, err := client.VerifyChallenge(context.Background(), &VerifyChallengeRequest{
		ChallengeID: "challenge-1",
		Code:        "123456",
	})
	testza.AssertNoError(t, err)
	testza.AssertTrue(t, resp.OK)
	testza.AssertEqual(t, "user-1", resp.UserID)
}

func TestVerifyChallenge_FailureStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(VerifyChallengeResponse{
			OK:     false,
			Reason: "invalid",
		})
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	testza.AssertNoError(t, err)

	resp, err := client.VerifyChallenge(context.Background(), &VerifyChallengeRequest{
		ChallengeID: "challenge-1",
		Code:        "000000",
	})
	testza.AssertNotNil(t, err)
	testza.AssertNotNil(t, resp)
	testza.AssertFalse(t, resp.OK)
	testza.AssertEqual(t, "invalid", resp.Reason)
	testza.AssertTrue(t, strings.Contains(err.Error(), "invalid"))
}

func TestVerifyChallenge_DecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	testza.AssertNoError(t, err)

	_, err = client.VerifyChallenge(context.Background(), &VerifyChallengeRequest{
		ChallengeID: "challenge-1",
		Code:        "123456",
	})
	testza.AssertNotNil(t, err)
}
