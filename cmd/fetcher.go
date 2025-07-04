package main

import (
	"fmt"
	"log"
	"os"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
)

// ConnectToEmail устанавливает TLS-соединение с IMAP-сервером и проходит аутентификацию.
// Возвращает готовый к работе клиент или ошибку.
func ConnectToEmail(cfg *Config) (*client.Client, error) {
	log.Printf("Подключаемся к IMAP-серверу %s", cfg.Email.IMAPServer)
	c, err := client.DialTLS(cfg.Email.IMAPServer, nil)
	if err != nil {
		return nil, err
	}

	log.Printf("Соединение установлено. Аутентифицируемся как %s", cfg.Email.Username)
	auth := sasl.NewPlainClient("", cfg.Email.Username, cfg.Email.OAuthToken)
	if err := c.Authenticate(auth); err != nil {
		_ = c.Logout()
		return nil, err
	}
	log.Println("Аутентификация успешна")
	return c, nil
}

// FetchMessages запрашивает письма начиная с UID, сохранённого в state/last_uid.txt.
// При limit > 0 загружает не более limit писем, иначе – все доступные.
// Функция также обновляет last_uid.txt, чтобы при следующем запуске не обрабатывать
// одни и те же сообщения повторно.
func FetchMessages(c *client.Client, limit int) ([]*imap.Message, error) {
	log.Println("Выбираем папку INBOX")
	mbox, err := c.Select("INBOX", false)
	if err != nil {
		return nil, err
	}
	log.Printf("INBOX выбран. Всего писем: %d", mbox.Messages)
	if mbox.Messages == 0 {
		return nil, nil
	}

	// UID последнего обработанного письма, чтобы не обрабатывать дубликаты
	lastUID := loadLastUID()
	log.Printf("Последний обработанный UID: %d", lastUID)

	section := &imap.BodySectionName{} // пустой раздел = всё письмо целиком
	items := []imap.FetchItem{imap.FetchEnvelope, section.FetchItem(), imap.FetchUid}

	// Формируем диапазон UID для выборки
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

	// Канал, в который библиотека будет отправлять письма
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

	// Сохраняем максимальный UID из загруженных, чтобы не скачивать их снова
	highest := lastUID
	for _, m := range messages {
		if int(m.Uid) > highest {
			highest = int(m.Uid)
		}
	}
	if highest > lastUID {
		saveLastUID(highest)
	}

	return messages, nil
}

// loadLastUID читает UID последнего обработанного письма из файла state/last_uid.txt.
// Если файл отсутствует или повреждён – возвращается 0.
func loadLastUID() int {
	f, err := os.Open("state/last_uid.txt")
	if err != nil {
		return 0
	}
	defer f.Close()

	var uid int
	if _, err := fmt.Fscan(f, &uid); err != nil {
		return 0
	}
	return uid
}

// saveLastUID сохраняет максимальный UID в файл state/last_uid.txt для будущих запусков.
func saveLastUID(uid int) {
	_ = os.MkdirAll("state", 0o755)
	f, err := os.Create("state/last_uid.txt")
	if err != nil {
		log.Printf("Не удалось записать last UID: %v", err)
		return
	}
	defer f.Close()

	fmt.Fprint(f, uid)
	log.Printf("Сохранён last UID: %d", uid)
}
