package auditlog

import (
	"context"
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
			// Use no-op storage if none provided
			storage = audit.NewNoopStorage()
		}

		logger = audit.NewLoggerWithWriter(storage, cfg)
	})
}

// GetLogger returns the audit logger instance
func GetLogger() *audit.Logger {
	if logger == nil {
		// Initialize with no-op storage if not initialized
		Init(nil, nil)
	}
	return logger
}

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
