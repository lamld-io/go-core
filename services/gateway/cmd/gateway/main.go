package main

import (
	"log"

	"github.com/base-go/base/services/gateway/internal/bootstrap"
)

func main() {
	app, err := bootstrap.NewApp()
	if err != nil {
		log.Fatalf("failed to initialize gateway service: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("gateway service exited with error: %v", err)
	}
}
