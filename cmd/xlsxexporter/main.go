package main

import (
	"log"
	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/xlsxexporter"
)

func main() {
	if err := config.SetupLogging("xlsxexporter"); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	xlsxexporter.StartServer()
}
