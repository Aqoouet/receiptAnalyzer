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
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap"
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
	log.Printf("Инициализируем хранилище SQLite: %s", cfg.Storage.DBPath)
	return storage.NewSQLiteStorage(cfg.Storage.DBPath)
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

// getMaxUIDForMailbox возвращает максимальный UID для папки по именам файлов в htmlDir
func getMaxUIDForMailbox(htmlDir, mailbox string) int {
	files, err := os.ReadDir(htmlDir)
	if err != nil {
		return 0
	}
	re := regexp.MustCompile("^" + regexp.QuoteMeta(sanitizeString(mailbox)) + "_([0-9]+)_")
	maxUID := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		matches := re.FindStringSubmatch(f.Name())
		if len(matches) == 2 {
			if uid, err := strconv.Atoi(matches[1]); err == nil && uid > maxUID {
				maxUID = uid
			}
		}
	}
	return maxUID
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

// saveEmailHTML сохраняет письмо с именем <ПАПКА>_<UID>_<дата>_<отправитель>_<тема>_<ХЭШ>.html
func saveEmailHTML(mboxName string, uid uint32, sender, subject string, date time.Time, body []byte, htmlDir string) {
	h := sha256.Sum256(body)
	hashStr := hex.EncodeToString(h[:])
	dateStr := date.Format("20060102_150405")
	fname := fmt.Sprintf("%s/%s_%d_%s_%s_%s_%s.html", htmlDir, sanitizeString(mboxName), uid, dateStr, sanitizeString(sender), sanitizeString(subject), hashStr)
	if existsFileWithHash(htmlDir, hashStr) {
		log.Printf("[DETAIL] Пропущено письмо: Папка=%s UID=%d Отправитель=%s Тема=%s Хэш=%s (уже сохранено)", mboxName, uid, sender, subject, hashStr)
		return
	}
	if err := os.WriteFile(fname, body, 0o644); err != nil {
		log.Printf("[DETAIL] Ошибка при сохранении письма: Папка=%s UID=%d Отправитель=%s Тема=%s Хэш=%s Файл=%s Ошибка=%v", mboxName, uid, sender, subject, hashStr, fname, err)
		return
	}
	log.Printf("[DETAIL] Сохранено письмо: Папка=%s UID=%d Отправитель=%s Тема=%s Хэш=%s Файл=%s", mboxName, uid, sender, subject, hashStr, fname)
}

// FetchAndSave скачивает все письма из всех папок и сохраняет их в HTML с уникальным хэшем
func FetchAndSave(cfg *config.Config) (int, error) {
	htmlDir := cfg.Paths.HTMLDir
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

	// Сначала собираем все папки в слайс
	mailboxesList := []*imap.MailboxInfo{}
	for m := range mailboxes {
		mailboxesList = append(mailboxesList, m)
	}
	if err := <-done; err != nil {
		return 0, fmt.Errorf("ошибка получения списка папок: %w", err)
	}

	doProcess := func(name string) bool {
		for _, prefix := range cfg.MailboxPrefixes {
			if name == prefix || (len(prefix) > 0 && len(name) >= len(prefix) && name[:len(prefix)] == prefix) {
				return true
			}
		}
		return false
	}
	totalNew, totalSkipped, totalErr := 0, 0, 0
	searchKeywords := []string{"чек", "ЧЕК", "Чек"} // можно добавить больше вариантов
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
		// SEARCH по ключевым словам (чек, ЧЕК, Чек)
		criteria := imap.NewSearchCriteria()
		criteria.Text = searchKeywords
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
				dateStr := msg.Envelope.Date.Format("20060102_150405")
				fname := fmt.Sprintf("%s/%s_%d_%s_%s_%s_%s.html", htmlDir, sanitizeString(mboxIMAPName), msg.Uid, dateStr, sanitizeString(sender), sanitizeString(subject), hashStr)
				if err := os.WriteFile(fname, rawMsg, 0o644); err != nil {
					log.Printf("Папка %s UID=%d: ошибка при сохранении письма: %v", mboxIMAPName, msg.Uid, err)
					errCount++
					continue
				}
				log.Printf("Папка %s UID=%d: письмо сохранено в %s (хэш %s, MIME error: %v)", mboxIMAPName, msg.Uid, fname, hashStr, err)
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
			h := sha256.Sum256(htmlBody)
			hashStr := hex.EncodeToString(h[:])
			if existsFileWithHash(htmlDir, hashStr) {
				log.Printf("Папка %s UID=%d: письмо с хэшем %s уже сохранено — пропущено (отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, hashStr, sender, subject)
				skipCount++
				continue
			}
			dateStr := msg.Envelope.Date.Format("20060102_150405")
			fname := fmt.Sprintf("%s/%s_%d_%s_%s_%s_%s.html", htmlDir, sanitizeString(mboxIMAPName), msg.Uid, dateStr, sanitizeString(sender), sanitizeString(subject), hashStr)
			if err := os.WriteFile(fname, htmlBody, 0o644); err != nil {
				log.Printf("Папка %s UID=%d: ошибка при сохранении письма: %v (отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, err, sender, subject)
				errCount++
				continue
			}
			log.Printf("Папка %s UID=%d: письмо сохранено в %s (хэш %s, отправитель: %s, тема: %s)", mboxIMAPName, msg.Uid, fname, hashStr, sender, subject)
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
