package main

import (
	"log"
	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/mailfetcher"
)

func main() {
	if err := config.SetupLogging("mailfetcher"); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	mailfetcher.StartServer()
}
