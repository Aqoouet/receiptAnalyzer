package main

import (
	"log"
)

func main() {
	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	imapClient, err := ConnectToEmail(cfg)
	if err != nil {
		log.Fatalf("Ошибка подключения к почте: %v", err)
	}
	defer imapClient.Logout()

	store, err := InitializeStorage(cfg)
	if err != nil {
		log.Fatalf("Ошибка инициализации хранилища: %v", err)
	}

	messages, err := FetchMessages(imapClient)
	if err != nil {
		log.Fatalf("Ошибка получения сообщений: %v", err)
	}

	ProcessAndSaveReceipts(imapClient, messages, store)
}
