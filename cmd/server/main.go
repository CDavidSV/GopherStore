package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/CDavidSV/GopherStore/internal/server"
)

func main() {
	addr := flag.String("addr", "0.0.0.0:5001", "Server network address")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	storage := server.NewInMemoryKVStore()
	server := server.NewServer(logger, *addr, storage)

	// Start server
	err := server.Start()
	if err != nil {
		logger.Error("Server failed to start", "error", err)
	}
}
