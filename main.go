package main

import (
	"log/slog"
	"os"

	"github.com/danielvm/bigbase/kernel"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	k := kernel.New(logger)

	if err := k.Start(); err != nil {
		logger.Error("failed to start kernel", "error", err)
		os.Exit(1)
	}

	logger.Info("bigbase started")
	select {}
}
