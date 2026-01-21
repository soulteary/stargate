package audit

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/soulteary/stargate/src/internal/config"
)

// AuditLogger handles audit log recording
type AuditLogger struct {
	enabled bool
	format  string // "json" or "text"
	logger  *logrus.Logger
}

var (
	auditLogger     *AuditLogger
	auditLoggerInit sync.Once
)

// AuditEvent represents an audit log event
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	UserID      string                 `json:"user_id,omitempty"`
	Method      string                 `json:"method,omitempty"`
	IP          string                 `json:"ip,omitempty"`
	Channel     string                 `json:"channel,omitempty"`
	Destination string                 `json:"destination,omitempty"`
	Result      string                 `json:"result"` // "success" or "failure"
	Reason      string                 `json:"reason,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Event types
const (
	EventTypeLogin           = "login"
	EventTypeLoginFailure    = "login_failure"
	EventTypeLogout          = "logout"
	EventTypeVerifyCodeSend  = "verify_code_send"
	EventTypeVerifyCodeCheck = "verify_code_check"
	EventTypeSessionCreate   = "session_create"
	EventTypeSessionDestroy  = "session_destroy"
)

// Result values
const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

// InitAuditLogger initializes the audit logger
func InitAuditLogger() {
	auditLoggerInit.Do(func() {
		enabled := true
		if config.AuditLogEnabled.Value != "" {
			enabled = config.AuditLogEnabled.ToBool()
		}

		format := "json"
		if config.AuditLogFormat.Value != "" {
			format = config.AuditLogFormat.String()
		}

		auditLogger = &AuditLogger{
			enabled: enabled,
			format:  format,
			logger:  logrus.StandardLogger(),
		}
	})
}

// GetAuditLogger returns the audit logger instance
func GetAuditLogger() *AuditLogger {
	InitAuditLogger()
	return auditLogger
}

// Log records an audit event
func (a *AuditLogger) Log(event *AuditEvent) {
	if !a.enabled {
		return
	}

	// Set timestamp if not set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Log based on format
	if a.format == "json" {
		eventJSON, err := json.Marshal(event)
		if err != nil {
			a.logger.WithError(err).Error("Failed to marshal audit event")
			return
		}
		a.logger.WithField("audit", true).Info(string(eventJSON))
	} else {
		// Text format
		a.logger.WithFields(logrus.Fields{
			"audit":       true,
			"event_type":  event.EventType,
			"user_id":     event.UserID,
			"method":      event.Method,
			"ip":          event.IP,
			"channel":     event.Channel,
			"destination": event.Destination,
			"result":      event.Result,
			"reason":      event.Reason,
		}).Info("Audit event")
	}
}

// LogLogin records a login event
func (a *AuditLogger) LogLogin(userID, method, ip string, success bool, reason string) {
	event := &AuditEvent{
		EventType: EventTypeLogin,
		UserID:    userID,
		Method:    method,
		IP:        ip,
		Result:    ResultSuccess,
	}
	if !success {
		event.EventType = EventTypeLoginFailure
		event.Result = ResultFailure
		event.Reason = reason
	}
	a.Log(event)
}

// LogLogout records a logout event
func (a *AuditLogger) LogLogout(userID, ip string) {
	event := &AuditEvent{
		EventType: EventTypeLogout,
		UserID:    userID,
		IP:        ip,
		Result:    ResultSuccess,
	}
	a.Log(event)
}

// LogVerifyCodeSend records a verification code send event
func (a *AuditLogger) LogVerifyCodeSend(userID, channel, destination, ip string, success bool, reason string) {
	event := &AuditEvent{
		EventType:   EventTypeVerifyCodeSend,
		UserID:      userID,
		Channel:     channel,
		Destination: destination,
		IP:          ip,
		Result:      ResultSuccess,
	}
	if !success {
		event.Result = ResultFailure
		event.Reason = reason
	}
	a.Log(event)
}

// LogVerifyCodeCheck records a verification code check event
func (a *AuditLogger) LogVerifyCodeCheck(userID, ip string, success bool, reason string) {
	event := &AuditEvent{
		EventType: EventTypeVerifyCodeCheck,
		UserID:    userID,
		IP:        ip,
		Result:    ResultSuccess,
	}
	if !success {
		event.Result = ResultFailure
		event.Reason = reason
	}
	a.Log(event)
}

// LogSessionCreate records a session creation event
func (a *AuditLogger) LogSessionCreate(userID, ip string) {
	event := &AuditEvent{
		EventType: EventTypeSessionCreate,
		UserID:    userID,
		IP:        ip,
		Result:    ResultSuccess,
	}
	a.Log(event)
}

// LogSessionDestroy records a session destruction event
func (a *AuditLogger) LogSessionDestroy(userID, ip string) {
	event := &AuditEvent{
		EventType: EventTypeSessionDestroy,
		UserID:    userID,
		IP:        ip,
		Result:    ResultSuccess,
	}
	a.Log(event)
}
