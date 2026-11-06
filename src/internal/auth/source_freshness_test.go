package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/warden/pkg/warden"
	"github.com/stretchr/testify/require"
)

func freshnessHeaders(now time.Time) http.Header {
	h := make(http.Header)
	h.Set("Warden-Snapshot-Metadata", "v1")
	h.Set("Warden-Snapshot-Version", strings.Repeat("a", 64))
	h.Set("Warden-Source-Succeeded-At", now.Add(-30*time.Second).Format(time.RFC3339Nano))
	h.Set("Warden-Snapshot-Valid-Until", now.Add(60*time.Second).Format(time.RFC3339Nano))
	return h
}

func TestSourceDeadlineRequiresCompleteCurrentEvidence(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	transport := sourceFreshnessTransport{maxAge: 90 * time.Second, skew: time.Second, now: func() time.Time { return now }}
	tests := []struct {
		name   string
		change func(http.Header)
	}{
		{"missing negotiation", func(h http.Header) { h.Del("Warden-Snapshot-Metadata") }},
		{"wrong negotiation", func(h http.Header) { h.Set("Warden-Snapshot-Metadata", "v2") }},
		{"duplicate", func(h http.Header) { h.Add("Warden-Snapshot-Metadata", "v1") }},
		{"oversized", func(h http.Header) { h.Set("Warden-Snapshot-Metadata", strings.Repeat("x", 129)) }},
		{"missing version", func(h http.Header) { h.Del("Warden-Snapshot-Version") }},
		{"malformed version", func(h http.Header) { h.Set("Warden-Snapshot-Version", "wrong") }},
		{"missing source", func(h http.Header) { h.Del("Warden-Source-Succeeded-At") }},
		{"invalid source", func(h http.Header) { h.Set("Warden-Source-Succeeded-At", "wrong") }},
		{"zero source", func(h http.Header) { h.Set("Warden-Source-Succeeded-At", time.Time{}.Format(time.RFC3339Nano)) }},
		{"missing deadline", func(h http.Header) { h.Del("Warden-Snapshot-Valid-Until") }},
		{"invalid deadline", func(h http.Header) { h.Set("Warden-Snapshot-Valid-Until", "wrong") }},
		{"deadline before source", func(h http.Header) {
			h.Set("Warden-Snapshot-Valid-Until", now.Add(-time.Hour).Format(time.RFC3339Nano))
		}},
		{"future source", func(h http.Header) {
			h.Set("Warden-Source-Succeeded-At", now.Add(2*time.Second).Format(time.RFC3339Nano))
		}},
		{"expired server deadline", func(h http.Header) { h.Set("Warden-Snapshot-Valid-Until", now.Format(time.RFC3339Nano)) }},
		{"expired local age", func(h http.Header) { h.Set("Warden-Source-Succeeded-At", now.Add(-time.Hour).Format(time.RFC3339Nano)) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := freshnessHeaders(now)
			tt.change(h)
			_, err := transport.sourceDeadline(h)
			require.ErrorIs(t, err, errSourceEvidence)
		})
	}
	h := freshnessHeaders(now)
	deadline, err := transport.sourceDeadline(h)
	require.NoError(t, err)
	require.Equal(t, now.Add(59*time.Second), deadline)
	h.Set("Warden-Snapshot-Valid-Until", now.Add(10*time.Second).Format(time.RFC3339Nano))
	deadline, err = transport.sourceDeadline(h)
	require.NoError(t, err)
	require.Equal(t, now.Add(9*time.Second), deadline)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	io.Reader
	closed bool
}

func (b *trackedBody) Close() error { b.closed = true; return nil }

func TestFreshnessTransportRejectsRedirectOriginsAndInvalidEvidence(t *testing.T) {
	now := time.Now().Round(0)
	body := &trackedBody{Reader: strings.NewReader("{}")}
	calls := 0
	transport := sourceFreshnessTransport{origin: "https://warden.example", maxAge: time.Minute, now: func() time.Time { return now }, base: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, "v1", r.Header.Get("Warden-Snapshot-Metadata"))
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: body}, nil
	})}
	req, _ := http.NewRequest(http.MethodGet, "https://other.example/user", nil)
	_, err := transport.RoundTrip(req)
	require.Error(t, err)
	require.Zero(t, calls)
	transport.origin = ":invalid"
	_, err = transport.RoundTrip(req)
	require.Error(t, err)
	transport.origin = "https://warden.example"
	req, _ = http.NewRequest(http.MethodGet, "https://warden.example/user", nil)
	_, err = transport.RoundTrip(req)
	require.ErrorIs(t, err, errSourceEvidence)
	require.True(t, body.closed)
	require.Empty(t, req.Header.Get("Warden-Snapshot-Metadata"))
	transport.base = roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("down") })
	_, err = transport.RoundTrip(req)
	require.EqualError(t, err, "down")
	transport.base = roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 503, Body: http.NoBody}, nil
	})
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, 503, resp.StatusCode)
	for _, code := range []int{200, 404} {
		transport.base = roundTripFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: code, Header: freshnessHeaders(now), Body: http.NoBody}, nil
		})
		evidence := &sourceEvidence{}
		req = req.WithContext(context.WithValue(context.Background(), sourceEvidenceKey{}, evidence))
		_, err = transport.RoundTrip(req)
		require.NoError(t, err)
		require.Equal(t, now.Add(30*time.Second), evidence.deadline())
	}
}

func TestStrictWardenLookupNegotiatesAndPreservesSourceDeadline(t *testing.T) {
	vars := []*config.EnvVariable{&config.WardenEnabled, &config.WardenURL, &config.WardenAPIKey, &config.WardenHMACKeyID, &config.WardenHMACSecret, &config.WardenSourceMaxAge, &config.WardenSourceClockSkew, &config.WardenTLSCACertFile, &config.WardenTLSClientCert, &config.WardenTLSClientKey, &config.WardenTLSServerName}
	previous := make([]string, len(vars))
	for i, v := range vars {
		previous[i] = v.Value
		v.Value = ""
	}
	t.Cleanup(func() {
		ResetWardenClientForTesting()
		for i, v := range vars {
			v.Value = previous[i]
		}
	})
	config.WardenEnabled.Value = "true"
	config.WardenSourceMaxAge.Value = "90s"
	config.WardenSourceClockSkew.Value = "1s"
	config.WardenAPIKey.Value = "test-key"
	config.WardenHMACKeyID.Value = "test-id"
	config.WardenHMACSecret.Value = "test-signing-secret"
	now := time.Now().Round(0)
	headers := freshnessHeaders(now)
	status := 200
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "v1", r.Header.Get("Warden-Snapshot-Metadata"))
		require.Equal(t, "test-key", r.Header.Get("X-API-Key"))
		if r.TLS == nil {
			require.Equal(t, "test-id", r.Header.Get("X-Key-Id"))
			path := r.URL.EscapedPath()
			if r.URL.RawQuery != "" {
				path += "?" + r.URL.RawQuery
			}
			emptyHash := sha256.Sum256(nil)
			canonical := strings.Join([]string{r.Method, path, r.Header.Get("X-Key-Id"), r.Header.Get("X-Timestamp"), r.Header.Get("X-Nonce"), hex.EncodeToString(emptyHash[:])}, "\n")
			mac := hmac.New(sha256.New, []byte("test-signing-secret"))
			_, _ = mac.Write([]byte(canonical))
			require.Equal(t, hex.EncodeToString(mac.Sum(nil)), r.Header.Get("X-Signature"))
		}
		for k, v := range headers {
			w.Header()[k] = v
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"user_id":"u1","mail":"active@example.com","status":"active"}`))
	}))
	defer server.Close()
	// Use HTTP for InitWardenClient, then a separate TLS transport test proves the wrapper preserves TLS settings.
	plain := httptest.NewServer(server.Config.Handler)
	defer plain.Close()
	config.WardenURL.Value = plain.URL
	ResetWardenClientForTesting()
	require.NoError(t, InitWardenClient(testLogger()))
	user, deadline, err := LookupUserInfo(context.Background(), "", "active@example.com")
	require.NoError(t, err)
	require.Equal(t, "u1", user.UserID)
	require.Equal(t, now.Add(59*time.Second), deadline)
	status = 404
	_, _, err = LookupUserInfo(context.Background(), "", "active@example.com")
	require.ErrorIs(t, err, ErrUserUnavailable)
	status = 503
	_, _, err = LookupUserInfo(context.Background(), "", "active@example.com")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUserUnavailable)
	status = 200
	headers.Del("Warden-Snapshot-Metadata")
	_, _, err = LookupUserInfo(context.Background(), "", "active@example.com")
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrUserUnavailable)
	headers = freshnessHeaders(now)
	tr := &sourceFreshnessTransport{origin: server.URL, maxAge: 90 * time.Second, now: time.Now, base: server.Client().Transport}
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/user", nil)
	req.Header.Set("X-API-Key", "test-key")
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
}

type expiringBody struct {
	io.Reader
	advance func()
}

func (b *expiringBody) Read(p []byte) (int, error) {
	n, err := b.Reader.Read(p)
	b.advance()
	return n, err
}
func (b *expiringBody) Close() error { return nil }

func TestStrictLookupRechecksEvidenceBeforeClassifyingResponse(t *testing.T) {
	previousClient, previousEnabled, previousMax := wardenClient, config.WardenEnabled.Value, config.WardenSourceMaxAge.Value
	t.Cleanup(func() {
		wardenClient = previousClient
		config.WardenEnabled.Value = previousEnabled
		config.WardenSourceMaxAge.Value = previousMax
	})
	config.WardenEnabled.Value = "true"
	config.WardenSourceMaxAge.Value = "90s"
	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"404 must not fall back to mail", 404, `{}`}, {"inactive identity", 200, `{"user_id":"u1","status":"disabled"}`}, {"active identity", 200, `{"user_id":"u1","status":"active"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			initial := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
			now := initial
			calls := 0
			wrapper := &sourceFreshnessTransport{origin: "https://warden.example", maxAge: 90 * time.Second, now: func() time.Time { return now }, base: roundTripFunc(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: tc.status, Header: freshnessHeaders(initial), Body: &expiringBody{Reader: strings.NewReader(tc.body), advance: func() { now = initial.Add(time.Minute) }}}, nil
			})}
			transport := &http.Transport{}
			transport.RegisterProtocol("https", wrapper)
			var err error
			wardenClient, err = warden.NewClient(warden.DefaultOptions().WithBaseURL("https://warden.example").WithTransport(transport))
			require.NoError(t, err)
			//nolint:staticcheck // Exercise the nil-context compatibility path.
			user, deadline, err := LookupUserInfo(nil, "13800138000", "active@example.com")
			require.Error(t, err)
			require.NotErrorIs(t, err, ErrUserUnavailable)
			require.Nil(t, user)
			require.True(t, deadline.IsZero())
			require.Equal(t, 1, calls)
		})
	}
}
