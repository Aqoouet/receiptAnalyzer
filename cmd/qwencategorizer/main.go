package main

import (
	"log"
	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/qwen/categorizer"
)

func main() {
	if err := config.SetupLogging("qwencategorizer"); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	categorizer.StartServer()
}
