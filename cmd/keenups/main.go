package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	flags "github.com/jessevdk/go-flags"
	"github.com/reansnow/keenups/internal/driver"
	"github.com/reansnow/keenups/internal/httpapi"
	"github.com/reansnow/keenups/internal/nutserver"
	"github.com/reansnow/keenups/internal/state"
)

var version = "dev"

type duration time.Duration

func (d *duration) UnmarshalFlag(value string) error {
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return err
	}
	*d = duration(parsed)
	return nil
}

func (d duration) MarshalFlag() (string, error) {
	return time.Duration(d).String(), nil
}

type nutOptions struct {
	Listen      string `long:"listen" env:"KEENUPS_LISTEN" default:":3493" description:"NUT listen address"`
	UPSName     string `long:"ups-name" env:"KEENUPS_UPS_NAME" default:"ups" description:"Network-visible UPS name"`
	Description string `long:"ups-description" env:"KEENUPS_UPS_DESCRIPTION" default:"Powercom WOW-500U via Keenetic" description:"UPS description"`
	Username    string `long:"username" env:"KEENUPS_USERNAME" default:"monuser" description:"NUT monitor username"`
	Password    string `long:"password" env:"KEENUPS_PASSWORD" default:"secret" default-mask:"-" description:"NUT monitor password"`
}

type httpOptions struct {
	Listen string `long:"http-listen" env:"KEENUPS_HTTP_LISTEN" default:":8080" description:"HTTP listen address (empty disables HTTP)"`
}

type driverOptions struct {
	Socket string   `long:"driver-socket" env:"KEENUPS_DRIVER_SOCKET" default:"/opt/var/run/usbhid-ups-keenups" description:"NUT driver state socket"`
	Binary string   `long:"driver-bin" env:"KEENUPS_DRIVER_BIN" default:"/opt/lib/nut/usbhid-ups" description:"Driver binary to supervise (empty disables supervision)"`
	Name   string   `long:"driver-name" env:"KEENUPS_DRIVER_NAME" default:"keenups" description:"UPS section name in ups.conf"`
	User   string   `long:"driver-user" env:"KEENUPS_DRIVER_USER" default:"root" description:"User passed to the NUT driver"`
	Warmup duration `long:"warmup" env:"KEENUPS_WARMUP" default:"8s" description:"Time to suppress startup readings after driver connection"`
}

type options struct {
	NUT    nutOptions
	HTTP   httpOptions
	Driver driverOptions
}

func parseOptions(args []string) (options, error) {
	var opts options
	parser := flags.NewNamedParser("keenups", flags.Default)
	if _, err := parser.AddGroup("NUT server", "NUT-compatible network service", &opts.NUT); err != nil {
		return opts, err
	}
	if _, err := parser.AddGroup("HTTP server", "Status page and JSON API", &opts.HTTP); err != nil {
		return opts, err
	}
	if _, err := parser.AddGroup("NUT USB driver", "Local usbhid-ups integration", &opts.Driver); err != nil {
		return opts, err
	}
	_, err := parser.ParseArgs(args)
	return opts, err
}

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		if flags.WroteHelp(err) {
			return
		}
		os.Exit(2)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	store := state.New(time.Duration(opts.Driver.Warmup))

	if opts.Driver.Binary != "" {
		supervisor := driver.Supervisor{
			Binary:     opts.Driver.Binary,
			DriverName: opts.Driver.Name,
			User:       opts.Driver.User,
			RetryDelay: 2 * time.Second,
			Logger:     logger,
		}
		go supervisor.Run(ctx)
	}

	source := driver.Source{
		SocketPath: opts.Driver.Socket,
		RetryDelay: 500 * time.Millisecond,
		Store:      store,
		Logger:     logger,
	}
	go source.Run(ctx)

	nut := nutserver.Server{
		Address:     opts.NUT.Listen,
		UPSName:     opts.NUT.UPSName,
		Description: opts.NUT.Description,
		Username:    opts.NUT.Username,
		Password:    opts.NUT.Password,
		Store:       store,
		Logger:      logger,
		Version:     version,
	}
	errCh := make(chan error, 2)
	go func() { errCh <- nut.ListenAndServe(ctx) }()

	var httpServer *http.Server
	if opts.HTTP.Listen != "" {
		httpServer = &http.Server{
			Addr:              opts.HTTP.Listen,
			Handler:           httpapi.Handler(store, version),
			ReadHeaderTimeout: 5 * time.Second,
		}
		go func() {
			logger.Info("HTTP server listening", "address", opts.HTTP.Listen)
			err := httpServer.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				errCh <- fmt.Errorf("HTTP server: %w", err)
			}
		}()
	}

	select {
	case <-ctx.Done():
		logger.Info("shutting down")
	case err := <-errCh:
		if err != nil {
			logger.Error("server stopped", "error", err)
			stop()
		}
	}

	if httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}
}
