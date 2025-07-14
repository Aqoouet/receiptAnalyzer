package main

import (
	"log"
	"receiptAnalyzer/internal/orchestrator"
)

func main() {
	err := orchestrator.RunAllServices()
	if err != nil {
		log.Fatalf("Ошибка orchestration: %v", err)
	}
	log.Println("Все микросервисы успешно вызваны")
}
