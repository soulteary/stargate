package auditlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	audit "github.com/soulteary/audit-kit"
	"github.com/soulteary/stargate/src/internal/config"
)

var (
	logger     *audit.Logger
	loggerInit sync.Once
)

// Init initializes the audit logger with the given storage and config
func Init(storage audit.Storage, cfg *audit.Config) {
	loggerInit.Do(func() {
		if cfg == nil {
			cfg = audit.DefaultConfig()
		}

		// Read enabled setting from environment
		if config.AuditLogEnabled.Value != "" {
			cfg.Enabled = config.AuditLogEnabled.ToBool()
		}
		if storage == nil {
			return
		}

		logger = audit.NewLoggerWithWriter(storage, cfg)
	})
}

// InitDefault configures audit events to stdout. No-op logging is used only
// when audit logging is explicitly disabled.
func InitDefault() error {
	cfg := audit.DefaultConfig()
	cfg.Enabled = config.AuditLogEnabled.ToBool()
	if !cfg.Enabled {
		Init(audit.NewNoopStorage(), cfg)
		return nil
	}
	format := config.AuditLogFormat.String()
	if format != "json" && format != "text" {
		return fmt.Errorf("unsupported audit log format %q", format)
	}
	Init(NewStreamStorage(os.Stdout, format), cfg)
	return nil
}

// GetLogger returns the audit logger instance
func GetLogger() *audit.Logger {
	return logger
}

type StreamStorage struct {
	writer io.Writer
	format string
	mu     sync.Mutex
}

func NewStreamStorage(writer io.Writer, format string) *StreamStorage {
	return &StreamStorage{writer: writer, format: format}
}

func (s *StreamStorage) Write(_ context.Context, record *audit.Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.format == "text" {
		_, err := fmt.Fprintf(s.writer, "timestamp=%d event=%s result=%s user_id=%q ip=%q reason=%q metadata=%v\n",
			record.Timestamp, record.EventType, record.Result, record.UserID, record.IP, record.Reason, record.Metadata)
		return err
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(s.writer, string(data))
	return err
}

func (s *StreamStorage) Query(context.Context, *audit.QueryFilter) ([]*audit.Record, error) {
	return nil, fmt.Errorf("stream audit storage does not support queries")
}

func (s *StreamStorage) Close() error { return nil }

// Stop stops the audit logger
func Stop() error {
	if logger != nil {
		return logger.Stop()
	}
	return nil
}

// LogLogin records a login event
func LogLogin(ctx context.Context, userID, method, ip string, success bool, reason string) {
	l := GetLogger()
	if l == nil {
		return
	}

	eventType := audit.EventLoginSuccess
	result := audit.ResultSuccess
	if !success {
		eventType = audit.EventLoginFailed
		result = audit.ResultFailure
	}

	l.LogAuth(ctx, eventType, userID, result,
		audit.WithRecordIP(ip),
		audit.WithRecordReason(reason),
		audit.WithRecordMetadata("method", method),
	)
}

// LogLogout records a logout event
func LogLogout(ctx context.Context, userID, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAuth(ctx, audit.EventLogout, userID, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}

// LogVerifyCodeSend records a verification code send event
func LogVerifyCodeSend(ctx context.Context, userID, channel, destination, ip string, success bool, reason string) {
	l := GetLogger()
	if l == nil {
		return
	}

	eventType := audit.EventSendSuccess
	result := audit.ResultSuccess
	if !success {
		eventType = audit.EventSendFailed
		result = audit.ResultFailure
	}

	l.LogChallenge(ctx, eventType, "", userID, result,
		audit.WithRecordChannel(channel),
		audit.WithRecordDestination(destination),
		audit.WithRecordIP(ip),
		audit.WithRecordReason(reason),
	)
}

// LogVerifyCodeCheck records a verification code check event
func LogVerifyCodeCheck(ctx context.Context, userID, ip string, success bool, reason string) {
	l := GetLogger()
	if l == nil {
		return
	}

	eventType := audit.EventVerificationSuccess
	result := audit.ResultSuccess
	if !success {
		eventType = audit.EventVerificationFailed
		result = audit.ResultFailure
	}

	l.LogChallenge(ctx, eventType, "", userID, result,
		audit.WithRecordIP(ip),
		audit.WithRecordReason(reason),
	)
}

// LogSessionCreate records a session creation event
func LogSessionCreate(ctx context.Context, userID, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAuth(ctx, audit.EventSessionCreate, userID, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}

// LogSessionDestroy records a session destruction event
func LogSessionDestroy(ctx context.Context, userID, ip string) {
	l := GetLogger()
	if l == nil {
		return
	}

	l.LogAuth(ctx, audit.EventSessionExpire, userID, audit.ResultSuccess,
		audit.WithRecordIP(ip),
	)
}
