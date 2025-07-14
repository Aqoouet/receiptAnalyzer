package mailfetcher

import (
	"log"

	"receiptAnalyzer/internal/config"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
)

// ConnectToEmail устанавливает TLS-соединение с IMAP-сервером и проходит аутентификацию.
// Возвращает готовый к работе клиент или ошибку.
func ConnectToEmail(cfg *config.Config) (*client.Client, error) {
	log.Printf("Подключаемся к IMAP-серверу %s", cfg.Email.IMAPServer)
	c, err := client.DialTLS(cfg.Email.IMAPServer, nil)
	if err != nil {
		return nil, err
	}
	// c.SetDebug(os.Stdout) // ОТКЛЮЧЕНО: IMAP debug-режим

	log.Printf("Соединение установлено. Аутентифицируемся как %s", cfg.Email.Username)
	auth := sasl.NewPlainClient("", cfg.Email.Username, cfg.Email.OAuthToken)
	if err := c.Authenticate(auth); err != nil {
		_ = c.Logout()
		return nil, err
	}
	log.Println("Аутентификация успешна")
	return c, nil
}

// FetchMessages запрашивает письма начиная с lastUID (UID последнего обработанного письма).
// При limit > 0 загружает не более limit писем, иначе – все доступные.
func FetchMessages(c *client.Client, limit int, lastUID int) ([]*imap.Message, error) {
	log.Println("Выбираем папку INBOX")
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, err
	}
	log.Printf("INBOX выбран. Всего писем: %d", mbox.Messages)
	if mbox.Messages == 0 {
		return nil, nil
	}
	log.Printf("Последний обработанный UID: %d", lastUID)
	section := &imap.BodySectionName{} // пустой раздел = всё письмо целиком
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem(), imap.FetchUid}
	seqset := new(imap.SeqSet)
	startUID := uint32(lastUID + 1)
	if startUID > mbox.Messages {
		log.Println("Новых писем нет — выходим")
		return nil, nil
	}
	endUID := mbox.Messages
	if limit > 0 {
		if calc := startUID + uint32(limit) - 1; calc < endUID {
			endUID = calc
		}
	}
	seqset.AddRange(startUID, endUID)
	log.Printf("Запрашиваем письма UID %d…%d", startUID, endUID)
	ch := make(chan *imap.Message, mbox.Messages)
	go func() {
		_ = c.Fetch(seqset, items, ch) // библиотека сама закроет канал
	}()
	var messages []*imap.Message
	processed := 0
	for msg := range ch {
		messages = append(messages, msg)
		processed++
		if processed%100 == 0 {
			log.Printf("Загружено %d писем…", processed)
		}
	}
	log.Printf("Загружено всего %d писем", len(messages))
	return messages, nil
}
