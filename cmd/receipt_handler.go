package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client" // Для *client.Client
	"github.com/emersion/go-message"

	"checkAnalyzer/internal/mailer"
	"checkAnalyzer/internal/receipt"
	"checkAnalyzer/internal/storage"
)

func InitializeStorage(cfg *Config) (storage.Storage, error) {
	return storage.NewSQLiteStorage(cfg.Storage.DBPath)
}

func ProcessAndSaveReceipts(client *client.Client, messages <-chan *imap.Message, store storage.Storage) {
	for msg := range messages {
		subject := msg.Envelope.Subject
		if !strings.Contains(strings.ToLower(subject), "чек") {
			continue
		}

		fmt.Printf("Обработка письма: %s\n", subject)

		// Поиск тела письма
		var bodyContent imap.Literal
		for k := range msg.Body {
			if k.Specifier == "" || k.Specifier == "TEXT" {
				bodyContent = msg.Body[k]
				break
			}
		}

		if bodyContent == nil {
			fmt.Println("Тело письма отсутствует")
			continue
		}

		// Читаем тело письма в буфер
		bodyReader := bytes.NewBuffer(nil)
		_, err := io.Copy(bodyReader, bodyContent)
		if err != nil {
			fmt.Printf("Ошибка чтения тела письма: %v\n", err)
			continue
		}

		// Парсим письмо
		entity, err := message.Read(bodyReader)
		if err != nil {
			fmt.Printf("Ошибка парсинга письма: %v\n", err)
			continue
		}

		// Извлекаем текст письма
		text, err := mailer.ProcessEmail(entity)
		if err != nil {
			fmt.Printf("Ошибка извлечения текста: %v\n", err)
			continue
		}

		fmt.Println("Содержимое письма:")
		fmt.Println(text)

		// Сохраняем чек
		receipt := receipt.Receipt{
			ID:        msg.Envelope.MessageId,
			Shop:      "Пример магазина",
			DateTime:  time.Now(),
			Total:     485.00,
			Source:    "email",
			CreatedAt: time.Now(),
		}

		store.SaveReceipt(receipt)
	}
}
