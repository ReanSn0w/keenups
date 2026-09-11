package driver

import (
	"context"
	"io"
	"log/slog"
	"os/exec"
	"time"
)

type Supervisor struct {
	Binary     string
	DriverName string
	User       string
	RetryDelay time.Duration
	Logger     *slog.Logger
}

func (s Supervisor) Run(ctx context.Context) {
	if s.RetryDelay <= 0 {
		s.RetryDelay = 2 * time.Second
	}
	for ctx.Err() == nil {
		args := []string{"-a", s.DriverName, "-F"}
		if s.User != "" {
			args = append(args, "-u", s.User)
		}
		cmd := exec.CommandContext(ctx, s.Binary, args...)
		cmd.Stdout = logWriter{s.Logger, slog.LevelInfo}
		cmd.Stderr = logWriter{s.Logger, slog.LevelWarn}
		s.Logger.Info("starting NUT USB driver", "binary", s.Binary, "ups", s.DriverName)
		err := cmd.Run()
		if ctx.Err() != nil {
			return
		}
		s.Logger.Warn("NUT USB driver exited; retrying", "error", err, "delay", s.RetryDelay)
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.RetryDelay):
		}
	}
}

type logWriter struct {
	logger *slog.Logger
	level  slog.Level
}

func (w logWriter) Write(p []byte) (int, error) {
	line := string(p)
	if line != "" {
		w.logger.Log(context.Background(), w.level, "NUT driver", "output", line)
	}
	return len(p), nil
}

var _ io.Writer = logWriter{}
