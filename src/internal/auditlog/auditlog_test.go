package auditlog

import (
	"bytes"
	"context"
	"sync"
	"testing"

	audit "github.com/soulteary/audit-kit"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestAuditLogFunctions(t *testing.T) {
	// Initialize with no-op storage for testing
	storage := audit.NewNoopStorage()
	cfg := audit.DefaultConfig()
	cfg.Enabled = true

	// Reset logger for testing
	logger = nil
	loggerInit = sync.Once{}

	Init(storage, cfg)

	l := GetLogger()
	assert.NotNil(t, l)

	ctx := context.Background()

	// Test all logging functions (should not panic)
	t.Run("LogLogin Success", func(t *testing.T) {
		LogLogin(ctx, "user1", "password", "127.0.0.1", true, "")
	})

	t.Run("LogLogin Failure", func(t *testing.T) {
		LogLogin(ctx, "user1", "password", "127.0.0.1", false, "invalid_password")
	})

	t.Run("LogLogout", func(t *testing.T) {
		LogLogout(ctx, "user1", "127.0.0.1")
	})

	t.Run("LogVerifyCodeSend Success", func(t *testing.T) {
		LogVerifyCodeSend(ctx, "user1", "sms", "13800000000", "127.0.0.1", true, "")
	})

	t.Run("LogVerifyCodeSend Failure", func(t *testing.T) {
		LogVerifyCodeSend(ctx, "user1", "sms", "13800000000", "127.0.0.1", false, "limit_exceeded")
	})

	t.Run("LogVerifyCodeCheck Success", func(t *testing.T) {
		LogVerifyCodeCheck(ctx, "user1", "127.0.0.1", true, "")
	})

	t.Run("LogVerifyCodeCheck Failure", func(t *testing.T) {
		LogVerifyCodeCheck(ctx, "user1", "127.0.0.1", false, "invalid_code")
	})

	t.Run("LogSessionCreate", func(t *testing.T) {
		LogSessionCreate(ctx, "user1", "127.0.0.1")
	})

	t.Run("LogSessionDestroy", func(t *testing.T) {
		LogSessionDestroy(ctx, "user1", "127.0.0.1")
	})

	// Test Stop
	err := Stop()
	assert.NoError(t, err)
}

func TestGetLoggerWithoutInit(t *testing.T) {
	// Reset logger
	logger = nil
	loggerInit = sync.Once{}

	l := GetLogger()
	assert.Nil(t, l)
}

func TestStreamStorageHonorsFormats(t *testing.T) {
	record := audit.NewRecord(audit.EventLoginSuccess, audit.ResultSuccess)
	record.UserID = "user1"

	var jsonOutput bytes.Buffer
	assert.NoError(t, NewStreamStorage(&jsonOutput, "json").Write(context.Background(), record))
	assert.Contains(t, jsonOutput.String(), "\"event_type\"")

	var textOutput bytes.Buffer
	assert.NoError(t, NewStreamStorage(&textOutput, "text").Write(context.Background(), record))
	assert.Contains(t, textOutput.String(), "event=")
	assert.Contains(t, textOutput.String(), "user_id=\"user1\"")
}

func TestInitDefaultAcceptsCaseInsensitiveFormat(t *testing.T) {
	previousEnabled := config.AuditLogEnabled.Value
	previousFormat := config.AuditLogFormat.Value
	t.Cleanup(func() {
		config.AuditLogEnabled.Value = previousEnabled
		config.AuditLogFormat.Value = previousFormat
		logger = nil
		loggerInit = sync.Once{}
	})
	config.AuditLogEnabled.Value = "true"
	config.AuditLogFormat.Value = " TEXT "
	logger = nil
	loggerInit = sync.Once{}

	assert.NoError(t, InitDefault())
	assert.NotNil(t, GetLogger())
	assert.NoError(t, Stop())
}

func TestStop_WhenLoggerNil(t *testing.T) {
	// Reset so logger is nil
	logger = nil
	loggerInit = sync.Once{}

	err := Stop()
	assert.NoError(t, err)
}
