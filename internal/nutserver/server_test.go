package nutserver

import (
	"testing"
	"time"

	"github.com/reansnow/keenups/internal/state"
)

func readyStore() *state.Store {
	s := state.New(0)
	s.Connected()
	s.Set("ups.status", "OL")
	s.Set("battery.charge", "100")
	s.SetDataOK(true)
	return s
}

func TestGetVar(t *testing.T) {
	s := &Server{UPSName: "ups", Description: "test", Store: readyStore(), Version: "test"}
	lines, closeConn := s.handle(&session{}, "GET VAR ups ups.status")
	if closeConn || len(lines) != 1 || lines[0] != `VAR ups ups.status "OL"` {
		t.Fatalf("got %#v close=%v", lines, closeConn)
	}
}

func TestSynologyLogin(t *testing.T) {
	s := &Server{UPSName: "ups", Username: "monuser", Password: "secret", Store: readyStore()}
	sess := &session{}
	for _, command := range []string{"USERNAME monuser", "PASSWORD secret", "LOGIN ups"} {
		lines, _ := s.handle(sess, command)
		if len(lines) != 1 || lines[0] != "OK" {
			t.Fatalf("%q: %#v", command, lines)
		}
	}
	if !sess.attached || s.attached.Load() != 1 {
		t.Fatal("session was not attached")
	}
}

func TestStaleData(t *testing.T) {
	store := state.New(time.Hour)
	store.Connected()
	store.Set("ups.status", "OL")
	store.SetDataOK(true)
	s := &Server{UPSName: "ups", Store: store}
	lines, _ := s.handle(&session{}, "GET VAR ups ups.status")
	if len(lines) != 1 || lines[0] != "ERR DATA-STALE" {
		t.Fatalf("got %#v", lines)
	}
}
