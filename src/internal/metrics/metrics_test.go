package metrics

import (
	"testing"
	"time"
)

func TestInit_RegistersMetrics(t *testing.T) {
	if Registry == nil {
		t.Fatal("Registry must not be nil after init")
	}
	if Auth == nil {
		t.Error("Auth must not be nil after init")
	}
	if Herald == nil {
		t.Error("Herald must not be nil after init")
	}
	if Warden == nil {
		t.Error("Warden must not be nil after init")
	}
	if AuthRefreshTotal == nil {
		t.Error("AuthRefreshTotal must not be nil after init")
	}
	if AuthRefreshDuration == nil {
		t.Error("AuthRefreshDuration must not be nil after init")
	}
}

func TestRecordAuthRequest_DoesNotPanic(t *testing.T) {
	RecordAuthRequest("password", "ok")
	RecordAuthRequest("header", "denied")
}

func TestRecordHeraldCall_DoesNotPanic(t *testing.T) {
	RecordHeraldCall("create_challenge", "success", 10*time.Millisecond)
	RecordHeraldCall("verify", "failure", 5*time.Millisecond)
}

func TestRecordWardenCall_DoesNotPanic(t *testing.T) {
	RecordWardenCall("get_user", "success", 20*time.Millisecond)
}

func TestRecordSessionCreated_DoesNotPanic(t *testing.T) {
	RecordSessionCreated()
}

func TestRecordSessionDestroyed_DoesNotPanic(t *testing.T) {
	RecordSessionDestroyed()
}

func TestRecordAuthRefresh_DoesNotPanic(t *testing.T) {
	RecordAuthRefresh("success", 50*time.Millisecond)
	RecordAuthRefresh("skipped", 0)
}
