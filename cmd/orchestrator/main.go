package main

import (
	"log"
	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/orchestrator"
)

func main() {
	if err := config.SetupLogging("orchestrator"); err != nil {
		log.Fatalf("Failed to setup logging: %v", err)
	}

	err := orchestrator.RunAllServices()
	if err != nil {
		log.Fatalf("Ошибка orchestration: %v", err)
	}
	log.Println("Все микросервисы успешно вызваны")
}
