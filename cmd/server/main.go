package main

import (
	"log"
	"os"

	"go-shoot/internal/config"
	"go-shoot/internal/server"
)

func main() {
	cfg := config.Load()

	srv := server.New(cfg)
	if err := srv.Run(); err != nil {
		log.Printf("server exited with error: %v", err)
		os.Exit(1)
	}
}
