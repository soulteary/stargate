package auth

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type sourceEvidenceKey struct{}
type sourceEvidence struct {
	mu    sync.Mutex
	until time.Time
	now   func() time.Time
}

func (e *sourceEvidence) set(t time.Time, now func() time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.until = t
	e.now = now
}
func (e *sourceEvidence) deadline() time.Time { e.mu.Lock(); defer e.mu.Unlock(); return e.until }
func (e *sourceEvidence) expired() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.until.IsZero() || e.now == nil || !e.now().Before(e.until)
}

type sourceFreshnessTransport struct {
	base         http.RoundTripper
	origin       string
	maxAge, skew time.Duration
	now          func() time.Time
}

var sourceVersionPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)
var errSourceEvidence = errors.New("warden source freshness evidence unavailable")

func singleSourceHeader(h http.Header, name string) (string, bool) {
	values := h.Values(name)
	if len(values) != 1 || len(values[0]) > 128 {
		return "", false
	}
	return values[0], true
}

func (t *sourceFreshnessTransport) sourceDeadline(h http.Header) (time.Time, error) {
	metadata, ok := singleSourceHeader(h, "Warden-Snapshot-Metadata")
	if !ok || metadata != "v1" {
		return time.Time{}, errSourceEvidence
	}
	version, ok := singleSourceHeader(h, "Warden-Snapshot-Version")
	if !ok || !sourceVersionPattern.MatchString(version) {
		return time.Time{}, errSourceEvidence
	}
	rawSource, sourceOK := singleSourceHeader(h, "Warden-Source-Succeeded-At")
	rawUntil, untilOK := singleSourceHeader(h, "Warden-Snapshot-Valid-Until")
	source, sourceErr := time.Parse(time.RFC3339Nano, rawSource)
	until, untilErr := time.Parse(time.RFC3339Nano, rawUntil)
	if !sourceOK || !untilOK || sourceErr != nil || untilErr != nil || source.IsZero() || !until.After(source) {
		return time.Time{}, errSourceEvidence
	}
	now := t.now()
	if source.After(now.Add(t.skew)) {
		return time.Time{}, errSourceEvidence
	}
	deadline := source.Add(t.maxAge)
	if until.Before(deadline) {
		deadline = until
	}
	deadline = deadline.Add(-t.skew)
	if !now.Before(deadline) {
		return time.Time{}, errSourceEvidence
	}
	return deadline, nil
}

func (t *sourceFreshnessTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	origin, err := url.Parse(t.origin)
	if err != nil || !strings.EqualFold(req.URL.Scheme, origin.Scheme) || !strings.EqualFold(req.URL.Host, origin.Host) {
		return nil, errors.New("warden redirects are disabled")
	}
	cloned := req.Clone(req.Context())
	cloned.Header.Set("Warden-Snapshot-Metadata", "v1")
	response, err := t.base.RoundTrip(cloned)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusNotFound {
		return response, nil
	}
	deadline, err := t.sourceDeadline(response.Header)
	if err != nil {
		_ = response.Body.Close()
		return nil, err
	}
	if evidence, ok := req.Context().Value(sourceEvidenceKey{}).(*sourceEvidence); ok {
		evidence.set(deadline, t.now)
	}
	return response, nil
}
