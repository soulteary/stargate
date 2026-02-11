// Package handlers provides HTTP request handlers for authentication and authorization.
package handlers

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/rs/zerolog"
	forwardauth "github.com/soulteary/forwardauth-kit"
	logger "github.com/soulteary/logger-kit"
	"github.com/soulteary/stargate/src/internal/auth"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/soulteary/stargate/src/internal/i18n"
)

// forwardAuthHandler is the global ForwardAuth handler instance.
var forwardAuthHandler *forwardauth.Handler

// forwardAuthLogger wraps logger-kit to implement forwardauth.Logger interface.
type forwardAuthLogger struct {
	log *logger.Logger
}

func (l *forwardAuthLogger) Debug() forwardauth.LogEvent {
	return &forwardAuthLogEvent{event: l.log.Debug()}
}
func (l *forwardAuthLogger) Info() forwardauth.LogEvent {
	return &forwardAuthLogEvent{event: l.log.Info()}
}
func (l *forwardAuthLogger) Warn() forwardauth.LogEvent {
	return &forwardAuthLogEvent{event: l.log.Warn()}
}
func (l *forwardAuthLogger) Error() forwardauth.LogEvent {
	return &forwardAuthLogEvent{event: l.log.Error()}
}

// forwardAuthLogEvent wraps zerolog log event.
type forwardAuthLogEvent struct {
	event *zerolog.Event
}

func (e *forwardAuthLogEvent) Str(key, val string) forwardauth.LogEvent {
	e.event = e.event.Str(key, val)
	return e
}
func (e *forwardAuthLogEvent) Bool(key string, val bool) forwardauth.LogEvent {
	e.event = e.event.Bool(key, val)
	return e
}
func (e *forwardAuthLogEvent) Int(key string, val int) forwardauth.LogEvent {
	e.event = e.event.Int(key, val)
	return e
}
func (e *forwardAuthLogEvent) Int64(key string, val int64) forwardauth.LogEvent {
	e.event = e.event.Int64(key, val)
	return e
}
func (e *forwardAuthLogEvent) Dur(key string, val time.Duration) forwardauth.LogEvent {
	e.event = e.event.Dur(key, val)
	return e
}
func (e *forwardAuthLogEvent) Err(err error) forwardauth.LogEvent {
	e.event = e.event.Err(err)
	return e
}
func (e *forwardAuthLogEvent) Msg(msg string) {
	e.event.Msg(msg)
}

// InitForwardAuthHandler initializes the ForwardAuth handler with current configuration.
func InitForwardAuthHandler(l *logger.Logger) {
	log = l

	// Parse password configuration
	algo, validPasswords := auth.GetValidPasswords()

	// Build ForwardAuth config from Stargate config
	faConfig := forwardauth.Config{
		// Session configuration
		SessionEnabled: true,

		// Password authentication
		PasswordEnabled:   algo != "" && len(validPasswords) > 0,
		PasswordHeader:    "Stargate-Password",
		ValidPasswords:    validPasswords,
		PasswordAlgorithm: algo,
		PasswordCheckFunc: func(password string) bool {
			return auth.CheckPassword(password)
		},

		// Header-based authentication (Warden)
		HeaderAuthEnabled:   config.WardenEnabled.ToBool(),
		HeaderAuthUserPhone: "X-User-Phone",
		HeaderAuthUserMail:  "X-User-Mail",
		HeaderAuthCheckFunc: func(phone, mail string) bool {
			return auth.CheckUserInList(context.Background(), phone, mail)
		},
		HeaderAuthGetInfoFunc: func(phone, mail string) *forwardauth.UserInfo {
			userInfo := auth.GetUserInfo(context.Background(), phone, mail)
			if userInfo == nil {
				return nil
			}
			return &forwardauth.UserInfo{
				UserID: userInfo.UserID,
				Email:  userInfo.Mail,
				Phone:  userInfo.Phone,
				Name:   userInfo.Name,
				Scopes: userInfo.Scope, // Warden uses 'Scope', forwardauth-kit uses 'Scopes'
				Role:   userInfo.Role,
				Status: userInfo.Status,
			}
		},

		// Step-up authentication
		StepUpEnabled:    config.StepUpEnabled.ToBool(),
		StepUpPaths:      parseStepUpPaths(),
		StepUpURL:        "/_step_up",
		StepUpSessionKey: "step_up_verified",

		// Auth refresh
		AuthRefreshEnabled:  config.AuthRefreshEnabled.ToBool(),
		AuthRefreshInterval: config.AuthRefreshInterval.ToDuration(),

		// Response headers
		UserHeaderName:   config.UserHeaderName.String(),
		AuthUserHeader:   "X-Auth-User",
		AuthEmailHeader:  "X-Auth-Email",
		AuthNameHeader:   "X-Auth-Name",
		AuthScopesHeader: "X-Auth-Scopes",
		AuthRoleHeader:   "X-Auth-Role",
		AuthAMRHeader:    "X-Auth-AMR",

		// Login redirect
		AuthHost:      config.AuthHost.String(),
		LoginPath:     "/_login",
		CallbackParam: "callback",

		// i18n support
		TranslateFunc: func(c forwardauth.Context, key string) string {
			// Get the underlying Fiber context if available
			if fc, ok := c.(*forwardauth.FiberContext); ok {
				return i18n.T(fc.Underlying(), key)
			}
			return key
		},

		// Logging
		Logger: &forwardAuthLogger{log: l},
	}

	// Set default refresh interval if not configured
	if faConfig.AuthRefreshInterval == 0 {
		faConfig.AuthRefreshInterval = 5 * time.Minute
	}

	forwardAuthHandler = forwardauth.NewHandler(&faConfig)
	log.Info().Msg("ForwardAuth handler initialized")
}

// parseStepUpPaths parses the step-up paths configuration.
func parseStepUpPaths() []string {
	pathsStr := config.StepUpPaths.String()
	if pathsStr == "" {
		return nil
	}

	paths := strings.Split(pathsStr, ",")
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// GetForwardAuthHandler returns the global ForwardAuth handler.
func GetForwardAuthHandler() *forwardauth.Handler {
	return forwardAuthHandler
}

// ForwardAuthCheckRoute creates a Fiber handler for the ForwardAuth check route.
// This is the main entry point for Traefik/Nginx ForwardAuth integration.
func ForwardAuthCheckRoute(store *session.Store) fiber.Handler {
	return forwardauth.FiberCheckRoute(forwardAuthHandler, store)
}
