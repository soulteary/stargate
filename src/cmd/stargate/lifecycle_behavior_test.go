package main

import (
	"net"
	"os"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/soulteary/stargate/src/internal/config"
	"github.com/stretchr/testify/require"
)

func TestApplicationReportsListenFailure(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	for key, value := range map[string]string{
		"AUTH_HOST": "auth.example.com", "PASSWORDS": "plaintext:test123", "PORT": port,
		"AUDIT_LOG_ENABLED": "false", "WARDEN_ENABLED": "false", "HERALD_ENABLED": "false", "OTLP_ENABLED": "false",
	} {
		t.Setenv(key, value)
	}
	require.ErrorContains(t, runApplication(), "address already in use")
	require.ErrorContains(t, runApplicationWithApp(fiber.New()), "address already in use")
}

func TestApplicationShutsDownOnTerminationSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process termination signals require Unix")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	require.NoError(t, listener.Close())
	for key, value := range map[string]string{
		"AUTH_HOST": "auth.example.com", "PASSWORDS": "plaintext:test123", "PORT": port,
		"AUDIT_LOG_ENABLED": "false", "WARDEN_ENABLED": "false", "HERALD_ENABLED": "false", "OTLP_ENABLED": "false", "SESSION_STORAGE_ENABLED": "false",
	} {
		t.Setenv(key, value)
	}
	done := make(chan error, 1)
	go func() { done <- runApplication() }()
	ready := false
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-done:
			t.Fatalf("server exited before listening: %v", err)
		default:
		}
		connection, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 50*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			ready = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !ready {
		t.Fatal("server did not start within five seconds")
	}
	process, err := os.FindProcess(os.Getpid())
	require.NoError(t, err)
	require.NoError(t, process.Signal(syscall.SIGTERM))
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(12 * time.Second):
		t.Fatal("graceful shutdown timed out")
	}
}

func TestStartServerAcceptsPrefixedPort(t *testing.T) {
	initLogger()
	previous := config.Port.Value
	t.Cleanup(func() { config.Port.Value = previous })
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer func() { require.NoError(t, listener.Close()) }()
	config.Port.Value = ":" + strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	require.ErrorContains(t, startServer(fiber.New()), "address already in use")
}
