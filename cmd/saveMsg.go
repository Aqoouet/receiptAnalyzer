package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"log"
	"os"
	"receiptAnalyzer/internal/storage"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/jhillyerd/enmime"
)

var savedHashes map[string]struct{}

// loadSavedHashes читает index.txt и заполняет карту
func loadSavedHashes() {
	log.Println("Загружаем сохранённые хэши из msg_html/index.txt")
	savedHashes = make(map[string]struct{})
	f, err := os.Open("msg_html/index.txt")
	if err != nil {
		return // файла нет – значит карта пустая
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		savedHashes[scanner.Text()] = struct{}{}
	}
	log.Printf("Загружено сохранённых хэшей: %d", len(savedHashes))
}

// appendHash дописывает новый хэш в index.txt
func appendHash(h string) {
	f, err := os.OpenFile("msg_html/index.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Не удалось открыть index.txt для дозаписи: %v", err)
		return
	}
	defer f.Close()
	f.WriteString(h + "\n")
	savedHashes[h] = struct{}{}
	log.Printf("Записан новый хэш %s в index.txt", h)
}

// InitializeStorage создаёт (или открывает) SQLite-файл и гарантирует наличие
// таблицы для чеков. Возвращает интерфейс Storage, который используется в коде
// выше по уровню для записи чеков.
func InitializeStorage(cfg *Config) (storage.Storage, error) {
	log.Printf("Инициализируем хранилище SQLite: %s", cfg.Storage.DBPath)
	return storage.NewSQLiteStorage(cfg.Storage.DBPath)
}

// SaveMessages сохраняет каждое письмо из списка messages на диск в каталог msg_html.
// Дубликаты (по SHA-256 тела письма) пропускаются.
func SaveMessages(messages []*imap.Message) {
	if savedHashes == nil {
		os.MkdirAll("msg_html", 0755)
		loadSavedHashes()
	}
	log.Printf("Сохраняем %d писем (режим сохранения включён)", len(messages))
	for _, msg := range messages {
		subject := msg.Envelope.Subject
		sender := "unknown"
		if len(msg.Envelope.From) > 0 {
			sender = msg.Envelope.From[0].MailboxName + "@" + msg.Envelope.From[0].HostName
		}
		log.Printf("Обработка письма от %s с темой \"%s\"", sender, subject)

		// Читаем сырое письмо полностью (BODY[])
		var rawMsg []byte
		for _, lit := range msg.Body {
			b, err := io.ReadAll(lit)
			if err == nil {
				rawMsg = b
				break
			}
		}
		if rawMsg == nil {
			log.Println("Не удалось извлечь тело письма – пропускаем")
			continue
		}

		// Парсим MIME-структуру и вытаскиваем HTML или текст
		env, err := enmime.ReadEnvelope(bytes.NewReader(rawMsg))
		if err != nil {
			log.Printf("Не удалось разобрать MIME (%v) – сохраняем как есть", err)
			saveEmailHTML(sender, subject, msg.Envelope.Date, rawMsg)
			continue
		}

		var htmlBody []byte
		if env.HTML != "" {
			htmlBody = []byte(env.HTML)
		} else if env.Text != "" {
			// Оборачиваем текст в <pre> и esc-кодируем
			escaped := html.EscapeString(env.Text)
			htmlBody = []byte("<pre>" + escaped + "</pre>")
		} else {
			log.Println("В письме нет текстовых частей – пропускаем")
			continue
		}

		saveEmailHTML(sender, subject, msg.Envelope.Date, htmlBody)
	}
}

// sanitizeString подготавливает строку для безопасного использования в имени файла —
// удаляет или заменяет символы, потенциально запрещённые в файловых системах.
func sanitizeString(s string) string {
	repl := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	s = repl.Replace(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// saveEmailHTML сохраняет тело письма в виде HTML-файла в каталоге msg_html.
// Проверяет дубликаты по SHA-256-хэшу: если файл с таким хэшем уже был
// сохранён ранее, запись пропускается. Глобальная карта savedHashes должна
// быть инициализирована вызовом loadSavedHashes.
func saveEmailHTML(sender, subject string, date time.Time, body []byte) {
	// Вычисляем хэш тела письма
	h := sha256.Sum256(body)
	hashStr := hex.EncodeToString(h[:])

	// Проверяем, не сохраняли ли уже такое письмо
	if _, exists := savedHashes[hashStr]; exists {
		log.Printf("Дубликат письма (хэш %s) – пропускаем", hashStr)
		return
	}

	// Формируем имя файла: YYYYMMDD_HHMMSS_отправитель_тема.html
	dateStr := date.Format("20060102_150405")
	fname := fmt.Sprintf("msg_html/%s_%s_%s.html", dateStr, sanitizeString(sender), sanitizeString(subject))

	// Сохраняем файл и обновляем индекс хэшей
	if err := os.WriteFile(fname, body, 0o644); err != nil {
		log.Printf("Ошибка при сохранении письма %s: %v", fname, err)
		return
	}
	appendHash(hashStr)
	log.Printf("Письмо сохранено в %s", fname)
}
