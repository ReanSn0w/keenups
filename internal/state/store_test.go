package state

import (
	"testing"
	"time"
)

func TestRejectsPowercomStartupGarbage(t *testing.T) {
	store := New(0)
	store.Connected()
	store.SetDataOK(true)
	store.Set("ups.status", "OL")
	store.Set("input.voltage", "294")
	store.Set("input.frequency", "70")
	store.Set("output.voltage", "271")
	store.Set("output.frequency", "70")
	store.Set("ups.load", "100")
	if store.Snapshot().Ready {
		t.Fatal("startup garbage must not be published")
	}

	store.Set("input.voltage", "238")
	store.Set("input.frequency", "50")
	store.Set("output.voltage", "238")
	store.Set("output.frequency", "50")
	store.Set("ups.load", "10")
	if !store.Snapshot().Ready {
		t.Fatal("plausible state should be ready")
	}
}

func TestWarmup(t *testing.T) {
	store := New(time.Hour)
	store.Connected()
	store.SetDataOK(true)
	store.Set("ups.status", "OL")
	if store.Snapshot().Ready {
		t.Fatal("state must remain unavailable during warmup")
	}
}
