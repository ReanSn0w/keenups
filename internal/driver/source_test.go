package driver

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/reansnow/keenups/internal/state"
)

func TestSourceConsumesDriverDump(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "keenups-test-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "driver.sock")
	listener, err := net.Listen("unix", path)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skip("sandbox does not permit Unix sockets")
		}
		t.Fatal(err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		line, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			serverDone <- err
			return
		}
		if line != "DUMPALL\n" {
			serverDone <- &unexpectedLine{line}
			return
		}
		_, err = io.WriteString(conn, "SETINFO ups.status \"OL\"\nSETINFO battery.charge \"100\"\nDATAOK\nDUMPDONE\n")
		serverDone <- err
	}()

	store := state.New(0)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := Source{
		SocketPath: path,
		RetryDelay: 10 * time.Millisecond,
		Store:      store,
		Logger:     slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	go source.Run(ctx)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		snap := store.Snapshot()
		if snap.DataOK && snap.Variables["ups.status"] == "OL" {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := store.Snapshot().Variables["battery.charge"]; got != "100" {
		t.Fatalf("battery.charge = %q", got)
	}
	if err := <-serverDone; err != nil {
		t.Fatal(err)
	}
}

type unexpectedLine struct{ line string }

func (e *unexpectedLine) Error() string { return "unexpected line: " + e.line }
