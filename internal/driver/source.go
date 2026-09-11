package driver

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/reansnow/keenups/internal/state"
)

type Source struct {
	SocketPath string
	RetryDelay time.Duration
	Store      *state.Store
	Logger     *slog.Logger
}

func (s Source) Run(ctx context.Context) {
	if s.RetryDelay <= 0 {
		s.RetryDelay = time.Second
	}
	failures := 0
	for ctx.Err() == nil {
		if err := s.consume(ctx); err != nil && ctx.Err() == nil {
			failures++
			// The Powercom can disappear between enumerations, and the supervised
			// driver may need several attempts. Keep retrying quickly without
			// flooding persistent router logs.
			if failures == 1 || failures%20 == 0 {
				s.Logger.Warn("waiting for NUT driver state socket", "error", err, "attempt", failures)
			}
		} else {
			failures = 0
		}
		s.Store.Disconnected()
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.RetryDelay):
		}
	}
}

func (s Source) consume(ctx context.Context) error {
	dialer := net.Dialer{Timeout: time.Second}
	conn, err := dialer.DialContext(ctx, "unix", s.SocketPath)
	if err != nil {
		return err
	}
	defer conn.Close()

	s.Store.Connected()
	s.Logger.Info("connected to NUT driver state socket", "path", s.SocketPath)
	if _, err := conn.Write([]byte("DUMPALL\n")); err != nil {
		return fmt.Errorf("request state dump: %w", err)
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-done:
		}
	}()
	defer close(done)

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	for scanner.Scan() {
		if err := s.apply(scanner.Text()); err != nil {
			s.Logger.Debug("ignored driver protocol line", "line", scanner.Text(), "error", err)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("driver closed state socket")
}

func (s Source) apply(line string) error {
	tokens, err := splitLine(line)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return nil
	}
	switch tokens[0] {
	case "SETINFO":
		if len(tokens) < 3 {
			return fmt.Errorf("short SETINFO")
		}
		s.Store.Set(tokens[1], tokens[2])
	case "DATAOK":
		s.Store.SetDataOK(true)
	case "DATASTALE":
		s.Store.SetDataOK(false)
	}
	return nil
}
