package auditlog

import (
	"context"
	"errors"
	"sync"
	"testing"

	audit "github.com/soulteary/audit-kit/v2"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("injected write failure") }

func TestStreamStoragePropagatesWriteAndEncodingFailures(t *testing.T) {
	for _, format := range []string{"json", "text"} {
		storage := NewStreamStorage(failingWriter{}, format)
		require.ErrorContains(t, storage.Write(context.Background(), &audit.Record{}), "injected write failure")
		require.Error(t, storage.Write(context.Background(), &audit.Record{Metadata: map[string]any{"invalid": make(chan int)}}))
		_, err := storage.Query(context.Background(), nil)
		require.ErrorContains(t, err, "does not support queries")
		require.NoError(t, storage.Close())
	}
}

func TestDisabledOrMissingAuditStorageDoesNotInitializeLogger(t *testing.T) {
	previousLogger, previousEnabled := logger, config.AuditLogEnabled.Value
	t.Cleanup(func() {
		logger = previousLogger
		loggerInit = sync.Once{}
		config.AuditLogEnabled.Value = previousEnabled
	})
	logger = nil
	loggerInit = sync.Once{}
	Init(nil, nil)
	require.Nil(t, GetLogger())
	require.NoError(t, Stop())
	LogLogin(context.Background(), "", "", "", false, "")
	LogLogout(context.Background(), "", "")
	LogVerifyCodeSend(context.Background(), "", "", "", "", "", false, "")
	LogVerifyCodeCheck(context.Background(), "", "", "", "", "", false, "")
	LogSessionCreate(context.Background(), "", "")
	LogSessionDestroy(context.Background(), "", "")
	loggerInit = sync.Once{}
	config.AuditLogEnabled.Value = "false"
	require.NoError(t, InitDefault())
	require.NotNil(t, GetLogger())
	require.NoError(t, Stop())
}
