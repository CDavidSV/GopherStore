package main

import (
	"log"
	"log/slog"
	"os"

	"github.com/CDavidSV/GopherStore/internal/server"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	server := server.NewServer(logger, ":5001")

	// Start server
	log.Fatal(server.Start())
}
