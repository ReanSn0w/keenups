package main

import (
	"testing"
	"time"
)

func TestParseOptionsDefaults(t *testing.T) {
	opts, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parse defaults: %v", err)
	}

	if opts.NUT.Listen != ":3493" || opts.NUT.UPSName != "ups" {
		t.Fatalf("unexpected NUT defaults: %+v", opts.NUT)
	}
	if opts.HTTP.Listen != ":8080" {
		t.Fatalf("unexpected HTTP default: %q", opts.HTTP.Listen)
	}
	if got := time.Duration(opts.Driver.Warmup); got != 8*time.Second {
		t.Fatalf("unexpected warmup default: %s", got)
	}
}

func TestParseOptionsEnvironmentAndFlags(t *testing.T) {
	t.Setenv("KEENUPS_LISTEN", ":1111")
	t.Setenv("KEENUPS_WARMUP", "12s")

	envOpts, err := parseOptions(nil)
	if err != nil {
		t.Fatalf("parse environment: %v", err)
	}
	if envOpts.NUT.Listen != ":1111" || time.Duration(envOpts.Driver.Warmup) != 12*time.Second {
		t.Fatalf("environment was not applied: %+v", envOpts)
	}

	opts, err := parseOptions([]string{"--listen", ":2222", "--http-listen=", "--warmup", "3s"})
	if err != nil {
		t.Fatalf("parse options: %v", err)
	}

	if opts.NUT.Listen != ":2222" {
		t.Fatalf("flag must override environment, got %q", opts.NUT.Listen)
	}
	if opts.HTTP.Listen != "" {
		t.Fatalf("empty HTTP address must be preserved, got %q", opts.HTTP.Listen)
	}
	if got := time.Duration(opts.Driver.Warmup); got != 3*time.Second {
		t.Fatalf("flag must override environment, got %s", got)
	}
}
