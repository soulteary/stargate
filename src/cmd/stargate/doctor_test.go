package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/soulteary/stargate/src/internal/config"
)

func TestDoctorDispatchDoesNotInitializeApplication(t *testing.T) {
	t.Setenv("AUTH_HOST", "")
	t.Setenv("OTLP_ENABLED", "true")
	t.Setenv("AUDIT_LOG_ENABLED", "true")
	t.Setenv("SESSION_STORAGE_ENABLED", "true")
	t.Setenv("SESSION_STORAGE_REDIS_ADDR", "unreachable.invalid:1")
	priorLogger, priorHost := log, config.AuthHost.Value
	var out, errOut bytes.Buffer
	handled, code := dispatchDoctor([]string{"doctor", "--help"}, &out, &errOut)
	if !handled || code != 0 || !strings.Contains(out.String(), "Usage:") {
		t.Fatalf("handled=%t exit=%d", handled, code)
	}
	if log != priorLogger || config.AuthHost.Value != priorHost {
		t.Fatal("diagnostics initialized application globals")
	}
	if handled, _ := dispatchDoctor(nil, &out, &errOut); handled {
		t.Fatal("normal no-argument startup was intercepted")
	}
	out.Reset()
	errOut.Reset()
	if handled, code := dispatchDoctor([]string{"doctor", "--private-argument"}, &out, &errOut); !handled || code != 2 {
		t.Fatal("invalid diagnostic arguments were not handled")
	}
	if strings.Contains(out.String()+errOut.String(), "private-argument") {
		t.Fatal("sensitive invalid argument echoed")
	}
}
