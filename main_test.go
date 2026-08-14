package main

import (
	"log/slog"
	"testing"
)

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
