package main

import (
	"log"

	"github.com/base-go/base/services/auth/internal/bootstrap"
)

func main() {
	app, err := bootstrap.NewApp()
	if err != nil {
		log.Fatalf("failed to initialize auth service: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatalf("auth service exited with error: %v", err)
	}
}
