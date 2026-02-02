package heraldtotp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client is the herald-totp HTTP client for Status and Verify.
type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
	hmacSecret string
	service    string
}

// Options for creating a client.
type Options struct {
	BaseURL    string
	APIKey     string
	HMACSecret string
	Service    string
	Timeout    time.Duration
}

// DefaultOptions returns default options.
func DefaultOptions() *Options {
	return &Options{
		Timeout: 10 * time.Second,
		Service: "stargate",
	}
}

// WithBaseURL sets the base URL.
func (o *Options) WithBaseURL(u string) *Options {
	o.BaseURL = u
	return o
}

// WithAPIKey sets the API key.
func (o *Options) WithAPIKey(k string) *Options {
	o.APIKey = k
	return o
}

// WithHMACSecret sets the HMAC secret.
func (o *Options) WithHMACSecret(s string) *Options {
	o.HMACSecret = s
	return o
}

// WithTimeout sets the timeout.
func (o *Options) WithTimeout(d time.Duration) *Options {
	o.Timeout = d
	return o
}

// NewClient creates a new herald-totp client.
func NewClient(opts *Options) (*Client, error) {
	if opts == nil {
		opts = DefaultOptions()
	}
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("base URL is required")
	}
	return &Client{
		httpClient: &http.Client{Timeout: opts.Timeout},
		baseURL:    opts.BaseURL,
		apiKey:     opts.APIKey,
		hmacSecret: opts.HMACSecret,
		service:    opts.Service,
	}, nil
}

// StatusResponse is the response from GET /v1/status.
type StatusResponse struct {
	Subject     string `json:"subject"`
	TotpEnabled bool   `json:"totp_enabled"`
}

// Status returns whether the subject has TOTP enabled.
func (c *Client) Status(ctx context.Context, subject string) (*StatusResponse, error) {
	u := c.baseURL + "/v1/status?subject=" + url.QueryEscape(subject)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	c.addAuthHeaders(req, nil)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(resp.Body)
	var out StatusResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status returned %d: %s", resp.StatusCode, string(body))
	}
	return &out, nil
}

// VerifyRequest is the request for POST /v1/verify.
type VerifyRequest struct {
	Subject     string `json:"subject"`
	Code        string `json:"code"`
	ChallengeID string `json:"challenge_id,omitempty"`
}

// VerifyResponse is the response from POST /v1/verify.
type VerifyResponse struct {
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

// EnrollStartRequest is the request for POST /v1/enroll/start.
type EnrollStartRequest struct {
	Subject string `json:"subject"`
	Label   string `json:"label"`
}

// EnrollStartResponse is the response from POST /v1/enroll/start.
type EnrollStartResponse struct {
	EnrollID     string `json:"enroll_id"`
	SecretBase32 string `json:"secret_base32,omitempty"`
	OtpauthURI   string `json:"otpauth_uri"`
}

// EnrollStart starts TOTP enrollment and returns enroll_id and otpauth_uri for QR code.
func (c *Client) EnrollStart(ctx context.Context, req *EnrollStartRequest) (*EnrollStartResponse, error) {
	u := c.baseURL + "/v1/enroll/start"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.addAuthHeaders(httpReq, body)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	var out EnrollStartResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("enroll/start returned %d: %s", resp.StatusCode, string(respBody))
	}
	return &out, nil
}

// EnrollConfirmRequest is the request for POST /v1/enroll/confirm.
type EnrollConfirmRequest struct {
	EnrollID string `json:"enroll_id"`
	Code     string `json:"code"`
}

// EnrollConfirmResponse is the response from POST /v1/enroll/confirm.
type EnrollConfirmResponse struct {
	Subject     string   `json:"subject"`
	TotpEnabled bool     `json:"totp_enabled"`
	BackupCodes []string `json:"backup_codes,omitempty"`
}

// EnrollConfirm confirms TOTP enrollment with a one-time code.
func (c *Client) EnrollConfirm(ctx context.Context, req *EnrollConfirmRequest) (*EnrollConfirmResponse, error) {
	u := c.baseURL + "/v1/enroll/confirm"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.addAuthHeaders(httpReq, body)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	var out EnrollConfirmResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("enroll/confirm returned %d: %s", resp.StatusCode, string(respBody))
	}
	return &out, nil
}

// Verify verifies a TOTP code for the subject.
func (c *Client) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error) {
	u := c.baseURL + "/v1/verify"
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.addAuthHeaders(httpReq, body)
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	respBody, _ := io.ReadAll(resp.Body)
	var out VerifyResponse
	_ = json.Unmarshal(respBody, &out)
	if resp.StatusCode != http.StatusOK {
		return &out, fmt.Errorf("verify returned %d: %s", resp.StatusCode, string(respBody))
	}
	return &out, nil
}

func (c *Client) addAuthHeaders(req *http.Request, body []byte) {
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}
	if c.hmacSecret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		signature := c.computeHMAC(timestamp, c.service, body)
		req.Header.Set("X-Timestamp", timestamp)
		req.Header.Set("X-Service", c.service)
		req.Header.Set("X-Signature", signature)
	}
}

func (c *Client) computeHMAC(timestamp, service string, body []byte) string {
	var b []byte
	if body != nil {
		b = body
	}
	message := fmt.Sprintf("%s:%s:%s", timestamp, service, string(b))
	mac := hmac.New(sha256.New, []byte(c.hmacSecret))
	mac.Write([]byte(message))
	return hex.EncodeToString(mac.Sum(nil))
}
