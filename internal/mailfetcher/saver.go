package mailfetcher

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
	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/storage"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-sasl"
	"github.com/jhillyerd/enmime"
)

// htmlDir устанавливается из main на основании конфигурации.
var htmlDir = "output/msg_html" // значение по умолчанию

var savedHashes map[string]struct{}

// loadSavedHashes читает index.txt и заполняет карту
func loadSavedHashes() {
	log.Println("Загружаем сохранённые хэши из msg_html/index.txt")
	savedHashes = make(map[string]struct{})

	// Создаем директорию если её нет
	_ = os.MkdirAll(htmlDir, 0755)

	f, err := os.Open(htmlDir + "/index.txt")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("Файл index.txt не найден - создаем новый")
			// Создаем пустой файл
			if f, err := os.Create(htmlDir + "/index.txt"); err == nil {
				f.Close()
			}
		}
		return // файла нет – значит карта пустая
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue // пропускаем пустые строки
		}
		savedHashes[line] = struct{}{}
	}
	log.Printf("Загружено сохранённых хэшей: %d", len(savedHashes))
}

// appendHash дописывает новый хэш в index.txt
func appendHash(h string) {
	f, err := os.OpenFile(htmlDir+"/index.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
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
func InitializeStorage(cfg *config.Config) (storage.Storage, error) {
	log.Printf("Инициализируем хранилище SQLite: %s", cfg.Paths.DBPath)
	return storage.NewSQLiteStorage(cfg.Paths.DBPath)
}

// Удаляю устаревшую функцию SaveMessages и все старые вызовы saveEmailHTML

// sanitizeString подготавливает строку для безопасного использования в имени файла —
// удаляет или заменяет символы, потенциально запрещённые в файловых системах.
func sanitizeString(s string) string {
	repl := strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	s = repl.Replace(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// existsFileWithHash проверяет, есть ли файл с данным хэшем в htmlDir
func existsFileWithHash(htmlDir, hash string) bool {
	files, err := os.ReadDir(htmlDir)
	if err != nil {
		return false
	}
	for _, f := range files {
		if strings.HasSuffix(f.Name(), hash+".html") {
			return true
		}
	}
	return false
}

// saveEmailHTML сохраняет письмо с именем <ПАПКА>_<UID>_<ХЭШ>.html, дата, тема и отправитель добавляются внутрь файла
func saveEmailHTML(mboxName string, uid uint32, sender, subject string, date time.Time, body []byte, htmlDir string) {
	h := sha256.Sum256(body)
	hashStr := hex.EncodeToString(h[:])
	fname := fmt.Sprintf("%s/%s_%d_%s.html", htmlDir, sanitizeString(mboxName), uid, hashStr)
	if existsFileWithHash(htmlDir, hashStr) {
		log.Printf("[DETAIL] Пропущено письмо: Папка=%s UID=%d Отправитель=%s Тема=%s Хэш=%s (уже сохранено)", mboxName, uid, sender, subject, hashStr)
		return
	}
	// Формируем HTML с датой, темой и отправителем
	metaBlock := fmt.Sprintf("<div><b>Дата:</b> %s<br><b>Тема:</b> %s<br><b>Отправитель:</b> %s</div>\n", html.EscapeString(date.Format("2006-01-02 15:04:05")), html.EscapeString(subject), html.EscapeString(sender))
	fullBody := append([]byte(metaBlock), body...)
	if err := os.WriteFile(fname, fullBody, 0o644); err != nil {
		log.Printf("[DETAIL] Ошибка при сохранении письма: Папка=%s UID=%d Отправитель=%s Тема=%s Файл=%s Ошибка=%v", mboxName, uid, sender, subject, fname, err)
		return
	}
	log.Printf("Папка %s UID=%d: письмо сохранено в %s (отправитель: %s, тема: %s)", mboxName, uid, fname, sender, subject)
}

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

// FetchAndSave скачивает все письма из всех папок и сохраняет их в HTML с уникальным хэшем
func FetchAndSave(cfg *config.Config) (int, error) {
	htmlDir := cfg.Paths.HTMLDirPath
	os.MkdirAll(htmlDir, 0755)
	log.Println("=== Начало выгрузки писем ===")
	c, err := ConnectToEmail(cfg)
	if err != nil {
		return 0, fmt.Errorf("ошибка подключения к почте: %w", err)
	}
	defer c.Logout()
	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)
	go func() { done <- c.List("", "*", mailboxes) }()

	// Сначала собираем все папки в слайс, только уникальные
	mailboxesList := []*imap.MailboxInfo{}
	seenMailboxes := make(map[string]struct{})
	for m := range mailboxes {
		if _, exists := seenMailboxes[m.Name]; exists {
			continue
		}
		seenMailboxes[m.Name] = struct{}{}
		mailboxesList = append(mailboxesList, m)
	}
	if err := <-done; err != nil {
		return 0, fmt.Errorf("ошибка получения списка папок: %w", err)
	}

	doProcess := func(name string) bool {
		if len(cfg.MailboxPrefixes) == 0 {
			return true // если не задано ни одного префикса, обрабатываем все папки
		}
		for _, prefix := range cfg.MailboxPrefixes {
			if name == prefix || (len(prefix) > 0 && len(name) >= len(prefix) && name[:len(prefix)] == prefix) {
				return true
			}
		}
		return false
	}
	totalNew, totalSkipped, totalErr := 0, 0, 0
	for _, m := range mailboxesList {
		mboxIMAPName := m.Name
		if !doProcess(mboxIMAPName) {
			log.Printf("[FetchAndSave] Пропускаю папку %s (не входит в mailbox_prefixes)", mboxIMAPName)
			continue
		}
		log.Printf("[FetchAndSave] Начинаю обработку папки (IMAP-имя): %s", mboxIMAPName)
		log.Printf("[DEBUG] Перед SELECT %s", mboxIMAPName)
		_, err := c.Select(mboxIMAPName, false)
		log.Printf("[DEBUG] После SELECT %s (err=%v)", mboxIMAPName, err)
		if err != nil {
			log.Printf("[DETAIL] SELECT %s завершился ошибкой: %v", mboxIMAPName, err)
			totalErr++
			continue
		}
		criteria := imap.NewSearchCriteria()
		if len(cfg.SearchKeywords) > 0 {
			criteria.Text = cfg.SearchKeywords
		}
		uids, err := c.Search(criteria)
		if err != nil {
			log.Printf("[DETAIL] SEARCH в папке %s завершился ошибкой: %v", mboxIMAPName, err)
			totalErr++
			continue
		}
		if len(uids) == 0 {
			log.Printf("[DETAIL] В папке %s не найдено писем по ключевым словам", mboxIMAPName)
			log.Printf("[FetchAndSave] Завершена обработка папки: %s (нет совпадений)", mboxIMAPName)
			continue
		}
		log.Printf("[DETAIL] В папке %s найдено %d писем по ключевым словам", mboxIMAPName, len(uids))
		seqset := new(imap.SeqSet)
		for _, uid := range uids {
			seqset.AddNum(uid)
		}
		items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchUid, (&imap.BodySectionName{}).FetchItem()}
		ch := make(chan *imap.Message, len(uids))
		go func() { _ = c.UidFetch(seqset, items, ch) }()
		newCount, skipCount, errCount := 0, 0, 0
		for msg := range ch {
			subject := msg.Envelope.Subject
			sender := "unknown"
			if len(msg.Envelope.From) > 0 {
				sender = msg.Envelope.From[0].MailboxName + "@" + msg.Envelope.From[0].HostName
			}
			var rawMsg []byte
			for _, lit := range msg.Body {
				b, err := io.ReadAll(lit)
				if err == nil {
					rawMsg = b
					break
				}
			}
			if rawMsg == nil {
				log.Printf("Папка %s UID=%d: не удалось извлечь тело письма — пропущено", mboxIMAPName, msg.Uid)
				skipCount++
				continue
			}
			// MIME разбор
			env, err := enmime.ReadEnvelope(bytes.NewReader(rawMsg))
			if err != nil {
				h := sha256.Sum256(rawMsg)
				hashStr := hex.EncodeToString(h[:])
				if existsFileWithHash(htmlDir, hashStr) {
					log.Printf("Папка %s UID=%d: письмо с хэшем %s уже сохранено — пропущено (MIME error: %v)", mboxIMAPName, msg.Uid, hashStr, err)
					skipCount++
					continue
				}
				fname := fmt.Sprintf("%s/%s_%d_%s.html", htmlDir, sanitizeString(mboxIMAPName), msg.Uid, hashStr)
				metaBlock := fmt.Sprintf("<div><b>Дата:</b> %s<br><b>Тема:</b> %s<br><b>Отправитель:</b> %s</div>\n", html.EscapeString(msg.Envelope.Date.Format("2006-01-02 15:04:05")), html.EscapeString(subject), html.EscapeString(sender))
				fullBody := append([]byte(metaBlock), rawMsg...)
				if err := os.WriteFile(fname, fullBody, 0o644); err != nil {
					log.Printf("Папка %s UID=%d: ошибка при сохранении письма: %v", mboxIMAPName, msg.Uid, err)
					errCount++
					continue
				}
				log.Printf("Папка %s UID=%d: письмо сохранено в %s (отправитель: %s, тема: %s, MIME error: %v)", mboxIMAPName, msg.Uid, fname, sender, subject, err)
				newCount++
				continue
			}
			var htmlBody []byte
			if env.HTML != "" {
				htmlBody = []byte(env.HTML)
			} else if env.Text != "" {
				escaped := html.EscapeString(env.Text)
				htmlBody = []byte("<pre>" + escaped + "</pre>")
			} else {
				log.Printf("Папка %s UID=%d: в письме нет текстовых частей — пропущено", mboxIMAPName, msg.Uid)
				skipCount++
				continue
			}
			// Фильтрация по ключевым словам в htmlBody (только отдельные слова, без регулярок)
			containsKeyword := false
			textToSearch := strings.ToLower(string(htmlBody))
			// Оставляем только буквы, цифры и пробелы
			cleaned := make([]rune, 0, len(textToSearch))
			for _, r := range textToSearch {
				if (r >= 'a' && r <= 'z') || (r >= 'а' && r <= 'я') || (r >= '0' && r <= '9') || r == 'ё' || r == ' ' {
					cleaned = append(cleaned, r)
				} else {
					cleaned = append(cleaned, ' ')
				}
			}
			words := strings.Fields(string(cleaned))
			for _, kw := range cfg.SearchKeywords {
				kwLower := strings.ToLower(kw)
				for _, w := range words {
					if w == kwLower {
						containsKeyword = true
						break
					}
				}
				if containsKeyword {
					break
				}
			}
			if !containsKeyword {
				log.Printf("Папка %s UID=%d: письмо пропущено — не найдено ключевых слов (отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, sender, subject)
				skipCount++
				continue
			}
			h := sha256.Sum256(htmlBody)
			hashStr := hex.EncodeToString(h[:])
			if existsFileWithHash(htmlDir, hashStr) {
				log.Printf("Папка %s UID=%d: письмо с хэшем %s уже сохранено — пропущено (отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, hashStr, sender, subject)
				skipCount++
				continue
			}
			fname := fmt.Sprintf("%s/%s_%d_%s.html", htmlDir, sanitizeString(mboxIMAPName), msg.Uid, hashStr)
			metaBlock := fmt.Sprintf("<div><b>Дата:</b> %s<br><b>Тема:</b> %s<br><b>Отправитель:</b> %s</div>\n", html.EscapeString(msg.Envelope.Date.Format("2006-01-02 15:04:05")), html.EscapeString(subject), html.EscapeString(sender))
			fullBody := append([]byte(metaBlock), htmlBody...)
			if err := os.WriteFile(fname, fullBody, 0o644); err != nil {
				log.Printf("Папка %s UID=%d: ошибка при сохранении письма: %v (отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, err, sender, subject)
				errCount++
				continue
			}
			log.Printf("Папка %s UID=%d: письмо сохранено в %s (отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, fname, sender, subject)
			newCount++
		}
		log.Printf("Папка %s: цикл по письмам завершён", mboxIMAPName)
		log.Printf("Папка %s: новых писем — %d, пропущено — %d, ошибок — %d", mboxIMAPName, newCount, skipCount, errCount)
		log.Printf("[FetchAndSave] Завершена обработка папки: %s", mboxIMAPName)
		totalNew += newCount
		totalSkipped += skipCount
		totalErr += errCount
	}
	log.Printf("=== Конец выгрузки писем: новых — %d, пропущено — %d, ошибок — %d ===", totalNew, totalSkipped, totalErr)
	return totalNew, nil
}
