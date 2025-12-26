package main

import (
	"log/slog"
	"os"

	"github.com/CDavidSV/GopherStore/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	storage := server.NewInMemoryKVStore()
	server := server.NewServer(logger, ":5001", storage)

	// Start server
	err := server.Start()
	if err != nil {
		logger.Error("Server failed to start", "error", err)
	}
}
