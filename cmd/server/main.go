package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/CDavidSV/GopherStore/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	storage := server.NewInMemoryKVStore()
	server := server.NewServer(logger, ":5001", storage)

	// Start server
	log.Fatal(server.Start())
}
