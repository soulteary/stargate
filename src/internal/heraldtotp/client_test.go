package heraldtotp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient_EmptyBaseURL(t *testing.T) {
	_, err := NewClient(DefaultOptions().WithBaseURL(""))
	if err == nil {
		t.Fatal("expected error when base URL is empty")
	}
}

func TestClient_Status_Verify_Revoke(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/status":
			_ = json.NewEncoder(w).Encode(StatusResponse{Subject: "user1", TotpEnabled: true})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/verify":
			_ = json.NewEncoder(w).Encode(VerifyResponse{OK: true})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/revoke":
			_ = json.NewEncoder(w).Encode(RevokeResponse{OK: true, Subject: "user1"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithAPIKey("test-key"))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()

	// Status
	statusResp, err := client.Status(ctx, "user1")
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if statusResp.Subject != "user1" || !statusResp.TotpEnabled {
		t.Errorf("Status: got subject=%q totp_enabled=%v", statusResp.Subject, statusResp.TotpEnabled)
	}

	// Verify
	verifyResp, err := client.Verify(ctx, &VerifyRequest{Subject: "user1", Code: "123456"})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !verifyResp.OK {
		t.Error("Verify: expected ok=true")
	}

	// Revoke
	revokeResp, err := client.Revoke(ctx, "user1")
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !revokeResp.OK || revokeResp.Subject != "user1" {
		t.Errorf("Revoke: got ok=%v subject=%q", revokeResp.OK, revokeResp.Subject)
	}
}

func TestClient_EnrollStart_EnrollConfirm(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/enroll/start":
			_ = json.NewEncoder(w).Encode(EnrollStartResponse{
				EnrollID:   "e1",
				OtpauthURI: "otpauth://totp/Test:user1?secret=JBSWY3DPEHPK3PXP",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/enroll/confirm":
			_ = json.NewEncoder(w).Encode(EnrollConfirmResponse{
				Subject:     "user1",
				TotpEnabled: true,
				BackupCodes: []string{"abc", "def"},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL).WithTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx := context.Background()

	startResp, err := client.EnrollStart(ctx, &EnrollStartRequest{Subject: "user1", Label: "user1@example.com"})
	if err != nil {
		t.Fatalf("EnrollStart: %v", err)
	}
	if startResp.EnrollID != "e1" || startResp.OtpauthURI == "" {
		t.Errorf("EnrollStart: got enroll_id=%q otpauth_uri=%q", startResp.EnrollID, startResp.OtpauthURI)
	}

	confirmResp, err := client.EnrollConfirm(ctx, &EnrollConfirmRequest{EnrollID: "e1", Code: "123456"})
	if err != nil {
		t.Fatalf("EnrollConfirm: %v", err)
	}
	if confirmResp.Subject != "user1" || !confirmResp.TotpEnabled || len(confirmResp.BackupCodes) != 2 {
		t.Errorf("EnrollConfirm: got subject=%q totp_enabled=%v backup_codes=%v",
			confirmResp.Subject, confirmResp.TotpEnabled, confirmResp.BackupCodes)
	}
}

func TestClient_Status_NonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.Status(context.Background(), "user1")
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestClient_Revoke_NonOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		if _, err := w.Write([]byte(`{"ok":false,"reason":"rate_limited"}`)); err != nil {
			return
		}
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	_, err = client.Revoke(context.Background(), "user1")
	if err == nil {
		t.Fatal("expected error for 429 response")
	}
}

// TestClient_WithHMACSecret verifies that WithHMACSecret sets HMAC auth and addAuthHeaders sends X-Timestamp, X-Service, X-Signature.
func TestClient_WithHMACSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Timestamp") == "" || r.Header.Get("X-Service") == "" || r.Header.Get("X-Signature") == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(StatusResponse{Subject: "user1", TotpEnabled: true})
	}))
	defer server.Close()

	client, err := NewClient(DefaultOptions().
		WithBaseURL(server.URL).
		WithHMACSecret("test-hmac-secret").
		WithTimeout(5 * time.Second))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	statusResp, err := client.Status(context.Background(), "user1")
	if err != nil {
		t.Fatalf("Status with HMAC: %v", err)
	}
	if statusResp.Subject != "user1" || !statusResp.TotpEnabled {
		t.Errorf("Status: got subject=%q totp_enabled=%v", statusResp.Subject, statusResp.TotpEnabled)
	}
}
