package main

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// When New Relic is disabled (nil app), buildHandler must return a plain
// slog JSON handler so local/dev logging is unchanged.
func TestBuildHandler_NRDisabled(t *testing.T) {
	h := buildHandler(nil, &bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo})
	if _, ok := h.(*slog.JSONHandler); !ok {
		t.Fatalf("buildHandler(nil, ...) = %T, want *slog.JSONHandler", h)
	}
}

// When a New Relic app is present, buildHandler must wrap the base handler
// through nrslog so records are forwarded to New Relic — i.e. it must NOT be a
// bare *slog.JSONHandler. Uses a disabled app (no network/license required).
func TestBuildHandler_NREnabled(t *testing.T) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("BigBaseTest"),
		newrelic.ConfigEnabled(false),
	)
	if err != nil {
		t.Fatalf("NewApplication: %v", err)
	}
	h := buildHandler(app, &bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo})
	if _, ok := h.(*slog.JSONHandler); ok {
		t.Fatalf("buildHandler(app, ...) = *slog.JSONHandler, want nrslog-wrapped handler")
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
		err   bool
	}{
		{"debug", slog.LevelDebug, false},
		{"info", slog.LevelInfo, false},
		{"warn", slog.LevelWarn, false},
		{"error", slog.LevelError, false},
		{"DEBUG", slog.LevelDebug, false},
		{"INFO", slog.LevelInfo, false},
		{"", slog.LevelInfo, true},
		{"invalid", slog.LevelInfo, true},
		{"trace", slog.LevelInfo, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseLogLevel(tt.input)
			if tt.err {
				if err == nil {
					t.Errorf("parseLogLevel(%q): expected error, got level=%v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("parseLogLevel(%q): unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseLogLevel(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
