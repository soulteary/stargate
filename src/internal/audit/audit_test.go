package audit

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func parseAuditLog(t *testing.T, logBytes []byte) map[string]interface{} {
	var logEntry map[string]interface{}
	err := json.Unmarshal(logBytes, &logEntry)
	assert.NoError(t, err)

	msg, ok := logEntry["msg"].(string)
	assert.True(t, ok, "msg field should be a string")

	var eventData map[string]interface{}
	err = json.Unmarshal([]byte(msg), &eventData)
	assert.NoError(t, err)
	return eventData
}

func TestAuditLogger(t *testing.T) {
	// Initialize logger
	InitAuditLogger()
	logger := GetAuditLogger()
	assert.NotNil(t, logger)

	// Capture output
	var buf bytes.Buffer
	logger.logger.SetOutput(&buf)
	logger.logger.SetFormatter(&logrus.JSONFormatter{
		DisableTimestamp: true, // Disable timestamp for easier matching
	})

	// Test case 1: Log generic event (JSON format)
	logger.format = "json"
	event := &AuditEvent{
		Timestamp: time.Now(),
		EventType: "test_event",
		UserID:    "user123",
		Result:    ResultSuccess,
	}
	logger.Log(event)

	// Verify output
	eventData := parseAuditLog(t, buf.Bytes())
	assert.Equal(t, "test_event", eventData["event_type"])
	assert.Equal(t, "user123", eventData["user_id"])
	assert.Equal(t, "success", eventData["result"])

	// Reset buffer
	buf.Reset()

	// Test case 2: Log disabled
	logger.enabled = false
	logger.Log(event)
	assert.Empty(t, buf.String())
	logger.enabled = true // Re-enable

	// Test case 3: Text format
	logger.format = "text"
	buf.Reset()
	logger.logger.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
		DisableColors:    true,
	})

	logger.Log(event)
	output := buf.String()
	assert.Contains(t, output, "event_type=test_event")
	assert.Contains(t, output, "user_id=user123")
	assert.Contains(t, output, "result=success")

	// Reset to JSON for helper method tests
	logger.format = "json"
	logger.logger.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
}

func TestAuditHelperMethods(t *testing.T) {
	InitAuditLogger()
	logger := GetAuditLogger()

	// Capture output
	var buf bytes.Buffer
	logger.logger.SetOutput(&buf)
	logger.logger.SetFormatter(&logrus.JSONFormatter{DisableTimestamp: true})
	logger.format = "json"
	logger.enabled = true

	t.Run("LogLogin Success", func(t *testing.T) {
		buf.Reset()
		logger.LogLogin("user1", "password", "127.0.0.1", true, "")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeLogin, eventData["event_type"])
		assert.Equal(t, "user1", eventData["user_id"])
		assert.Equal(t, ResultSuccess, eventData["result"])
	})

	t.Run("LogLogin Failure", func(t *testing.T) {
		buf.Reset()
		logger.LogLogin("user1", "password", "127.0.0.1", false, "invalid_password")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeLoginFailure, eventData["event_type"])
		assert.Equal(t, ResultFailure, eventData["result"])
		assert.Equal(t, "invalid_password", eventData["reason"])
	})

	t.Run("LogLogout", func(t *testing.T) {
		buf.Reset()
		logger.LogLogout("user1", "127.0.0.1")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeLogout, eventData["event_type"])
	})

	t.Run("LogVerifyCodeSend Success", func(t *testing.T) {
		buf.Reset()
		logger.LogVerifyCodeSend("user1", "sms", "13800000000", "127.0.0.1", true, "")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeVerifyCodeSend, eventData["event_type"])
		assert.Equal(t, "sms", eventData["channel"])
		assert.Equal(t, ResultSuccess, eventData["result"])
	})

	t.Run("LogVerifyCodeSend Failure", func(t *testing.T) {
		buf.Reset()
		logger.LogVerifyCodeSend("user1", "sms", "13800000000", "127.0.0.1", false, "limit_exceeded")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeVerifyCodeSend, eventData["event_type"])
		assert.Equal(t, ResultFailure, eventData["result"])
		assert.Equal(t, "limit_exceeded", eventData["reason"])
	})

	t.Run("LogVerifyCodeCheck Success", func(t *testing.T) {
		buf.Reset()
		logger.LogVerifyCodeCheck("user1", "127.0.0.1", true, "")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeVerifyCodeCheck, eventData["event_type"])
		assert.Equal(t, ResultSuccess, eventData["result"])
	})

	t.Run("LogVerifyCodeCheck Failure", func(t *testing.T) {
		buf.Reset()
		logger.LogVerifyCodeCheck("user1", "127.0.0.1", false, "invalid_code")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeVerifyCodeCheck, eventData["event_type"])
		assert.Equal(t, ResultFailure, eventData["result"])
		assert.Equal(t, "invalid_code", eventData["reason"])
	})

	t.Run("LogSessionCreate", func(t *testing.T) {
		buf.Reset()
		logger.LogSessionCreate("user1", "127.0.0.1")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeSessionCreate, eventData["event_type"])
	})

	t.Run("LogSessionDestroy", func(t *testing.T) {
		buf.Reset()
		logger.LogSessionDestroy("user1", "127.0.0.1")

		eventData := parseAuditLog(t, buf.Bytes())
		assert.Equal(t, EventTypeSessionDestroy, eventData["event_type"])
	})
}
