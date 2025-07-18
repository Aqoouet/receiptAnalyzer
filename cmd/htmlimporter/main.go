package main

import (
	"log"
	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/htmlimporter"
)

func main() {
	if err := config.SetupLogging("htmlimporter"); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	htmlimporter.StartServer()
}
