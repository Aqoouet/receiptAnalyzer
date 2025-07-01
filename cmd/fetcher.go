package main

import (
	"github.com/emersion/go-imap"        // Основной пакет для других типов (например, FetchItem)
	"github.com/emersion/go-imap/client" // Импортируем подпакет с Client

	"checkAnalyzer/internal/mailer"
)

func ConnectToEmail(cfg *Config) (*client.Client, error) { // Используем client.Client
	imapClient, err := mailer.ConnectToYandexOAuth(cfg.Email.Username, cfg.Email.OAuthToken)
	if err != nil {
		return nil, err
	}
	return imapClient, nil
}

func FetchMessages(client *client.Client) (<-chan *imap.Message, error) { // Используем client.Client
	mbox, err := client.Select("INBOX", false)
	if err != nil {
		return nil, err
	}

	seqSet := new(imap.SeqSet) // Типы из основного пакета
	seqSet.AddRange(1, mbox.Messages)

	messages := make(chan *imap.Message)
	go func() {
		if err := client.Fetch(seqSet, []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody}, messages); err != nil {
			panic(err)
		}
	}()

	return messages, nil
}
