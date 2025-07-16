package htmlimporter

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/receipt"
	"receiptAnalyzer/internal/storage"

	"github.com/PuerkitoBio/goquery"
)

// ImportSavedHTML проходит по каталогу msg_html, добавляет сведения о каждом
// HTML-файле в базу и сохраняет позиции товаров.
// Возвращает количество успешно импортированных чеков и количество пропущенных файлов.
// Файл считается уже импортированным, если в таблице существует запись с таким id.
func ImportSavedHTML(store storage.Storage, htmlDir string) (int, int, error) {
	log.Printf("[ImportSavedHTML] Начинаем импорт из директории: %s", htmlDir)
	absHtmlDir, err := filepath.Abs(htmlDir)
	if err != nil {
		absHtmlDir = htmlDir
	}
	files, err := ioutil.ReadDir(htmlDir)
	if err != nil {
		return 0, 0, fmt.Errorf("каталог %s недоступен: %w", htmlDir, err)
	}

	log.Printf("[ImportSavedHTML] Найдено файлов: %d", len(files))
	imported := 0
	skipped := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".html") {
			log.Printf("[ImportSavedHTML] Пропускаем файл (не .html): %s", f.Name())
			continue
		}
		log.Printf("[ImportSavedHTML] Обрабатываем файл: %s", f.Name())

		// --- вычисляем hash файла ---
		filePath := filepath.Join(htmlDir, f.Name())
		hashFull := strings.TrimSuffix(f.Name(), ".html")
		folder, receiptID, hash := splitHashParts(hashFull)
		log.Printf("[ImportSavedHTML] Парсинг имени файла: folder='%s', receiptID='%s', hash='%s'", folder, receiptID, hash)

		sender, subject, dt := extractMetaFromHTML(filePath)
		if dt.IsZero() {
			dt = f.ModTime()
		}
		log.Printf("[ImportSavedHTML] Мета-данные: sender='%s', subject='%s', date='%s'", sender, subject, dt.Format("2006-01-02 15:04:05"))

		absPath := filepath.Join(absHtmlDir, f.Name())
		link := "file://" + filepath.ToSlash(absPath)

		// --- парсинг чека и позиций ---
		templateNames := []string{
			"DefaultBeelineTemplate",
			"BelineOFD100Template",
			"DefaultTaxcomTemplate",
			"DefaultMusicTemplate",
			"DefaultYandexOFDTemplate",
			"DefaultYandexMarketTemplate",
			"DefaultOFDruTemplate",
			"DefaultFirstOFDTemplate",
			"DefaultPlatformaOFDTemplate",
			"YandexMarketTemplate",
			"YandexOFDPlainTableTemplate",
			"MTSPaymentTemplate",
			"UnitellerTemplate",
			"OfdYaKassaTemplate",
			"OFDruNestedTableTemplate",
		}
		log.Printf("[ImportSavedHTML] Пробуем распарсить чек с помощью %d шаблонов", len(templateNames))
		itemsJSON, usedIdx, perr := receipt.ParseReceiptAuto(filePath, []receipt.Template{
			receipt.BelineOFD100Template, // BelineOFD100 для ofdreceipt@beeline.ru
			receipt.DefaultBeelineTemplate,
			receipt.DefaultTaxcomTemplate,
			receipt.DefaultMusicTemplate,
			receipt.DefaultYandexOFDTemplate,
			receipt.DefaultYandexMarketTemplate,
			receipt.DefaultOFDruTemplate,
			receipt.DefaultFirstOFDTemplate,
			receipt.DefaultPlatformaOFDTemplate,
			receipt.YandexMarketTemplate,        // новый шаблон
			receipt.YandexOFDPlainTableTemplate, // Яндекс.ОФД (простая таблица)
			receipt.MTSPaymentTemplate,          // MTS платеж
			receipt.UnitellerTemplate,           // Uniteller
			receipt.OfdYaKassaTemplate,          // ofd_ya_kassa
			receipt.OFDruNestedTableTemplate,    // ofd.ru (вложенная таблица)
		})
		if perr != nil {
			log.Printf("[ImportSavedHTML] Парсинг не удался: %v", perr)
			log.Printf("[ImportSavedHTML] Сохраняем как не-чек (is_receipt=false)")
			// Добавляем запись в receipts с is_receipt=false
			r := receipt.Receipt{
				Folder:    folder,
				ReceiptID: receiptID,
				Hash:      hash,
				Sender:    sender,
				DateTime:  dt,
				Subject:   subject,
				IsReceipt: false,
				Link:      link,
				Total:     0,
				DeltaSum:  0,
			}
			if err := store.SaveReceipt(r); err != nil {
				if strings.Contains(err.Error(), "UNIQUE") {
					log.Printf("[ImportSavedHTML] Чек %s уже существует, пропускаем", hash)
					skipped++
					continue
				}
				log.Printf("[ImportSavedHTML] Ошибка сохранения чека %s: %v", hash, err)
				return imported, skipped, fmt.Errorf("ошибка сохранения чека %s: %w", hash, err)
			}
			log.Printf("[ImportSavedHTML] Чек сохранен как не-чек: %s", hash)
			skipped++
			continue
		}
		var items []receipt.Item
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil || len(items) == 0 {
			log.Printf("[ImportSavedHTML] Не удалось декодировать позиции или их нет: %v", err)
			skipped++
			continue
		}

		templateName := ""
		if usedIdx >= 0 && usedIdx < len(templateNames) {
			templateName = templateNames[usedIdx]
		}
		log.Printf("[ImportSavedHTML] Успешно распарсен с шаблоном: %s, найдено позиций: %d", templateName, len(items))

		// --- вычисляем заявленную сумму Итого из HTML ---
		extractedTotal := 0.0
		if tot, ok := extractTotalFromHTML(filePath); ok {
			extractedTotal = tot
		}

		// --- создаём Receipt ---
		r := receipt.Receipt{
			Folder:    folder,
			ReceiptID: receiptID,
			Hash:      hash,
			Sender:    sender,
			DateTime:  dt,
			Subject:   subject,
			IsReceipt: true,
			Link:      link,
			Template:  templateName,
			Total:     extractedTotal, // сохраняем заявленную сумму
			DeltaSum:  0,
		}

		if err := store.SaveReceipt(r); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				log.Printf("[ImportSavedHTML] Чек %s уже существует, пропускаем", hash)
				skipped++
				continue
			}
			log.Printf("[ImportSavedHTML] Ошибка сохранения чека %s: %v", hash, err)
			return imported, skipped, fmt.Errorf("ошибка сохранения чека %s: %w", hash, err)
		}

		// --- обновляем hash и новые поля в items ---
		for i := range items {
			items[i].Hash = hash
			// category, sub_quantity, name_cleaned — пока пустые, будут заполняться микросервисами
		}
		if err := store.SaveItems(hash, items); err != nil {
			log.Printf("[ImportSavedHTML] Ошибка сохранения позиций по чеку %s: %v", hash, err)
		}
		imported++
		log.Printf("[ImportSavedHTML] Чек %s добавлен: sender=%s, subject=%s, позиций=%d", hash, sender, subject, len(items))
	}
	log.Printf("[ImportSavedHTML] Итого: добавлено %d чеков, пропущено %d", imported, skipped)
	return imported, skipped, nil
}

// ImportAll импортирует все новые HTML-файлы в базу
func ImportAll(cfg *config.Config) error {
	htmlDir := cfg.Paths.HTMLDirPath
	_ = htmlDir // чтобы не было ошибки о неиспользуемой переменной
	return nil
}

// inferMetaFromFilename извлекает дату (DD.MM.YYYY) и название магазина из имени файла.
func inferMetaFromFilename(fname string) (shop string, date time.Time, ok bool) {
	reDate := regexp.MustCompile(`(0[1-9]|[12][0-9]|3[01])\.(0[1-9]|1[0-2])\.([0-9]{4})`)
	dateStr := reDate.FindString(fname)
	if dateStr != "" {
		if d, err := time.Parse("02.01.2006", dateStr); err == nil {
			date = d
			ok = true
		}
	}

	// удаляем префикс YYYYMMDD_HHMMSS_sender_
	namePart := fname
	if idx := strings.Index(namePart, "_"); idx != -1 {
		if len(namePart) > idx+1 {
			namePart = namePart[idx+1:]
			if idx2 := strings.Index(namePart, "_"); idx2 != -1 && len(namePart) > idx2+1 {
				namePart = namePart[idx2+1:]
			}
		}
	}

	underscored := strings.ReplaceAll(namePart, ".html", "")
	underscored = strings.ReplaceAll(underscored, "__", "_")
	underscored = strings.ReplaceAll(underscored, "_", " ")
	kasRe := regexp.MustCompile(`(?i)кассовый чек от ([^,]+)`) // insensitive search
	if m := kasRe.FindStringSubmatch(underscored); len(m) == 2 {
		shop = strings.TrimSpace(m[1])
	} else {
		parts := strings.Split(fname, ",")
		if len(parts) > 1 {
			shop = strings.TrimSuffix(parts[len(parts)-1], ".html")
		}
	}

	shop = strings.ReplaceAll(shop, "_", " ")
	shop = regexp.MustCompile(`\s+`).ReplaceAllString(shop, " ")
	shop = strings.TrimSpace(shop)
	return
}

func extractMetaFromHTML(path string) (sender, subject string, dt time.Time) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", time.Time{}
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return "", "", time.Time{}
	}
	meta := doc.Find("div").First().Text()
	re := regexp.MustCompile(`Дата:\s*([0-9\-: ]+)\s*Тема:\s*(.*?)\s*Отправитель:\s*([\w@.\-]+)`)
	match := re.FindStringSubmatch(meta)
	if len(match) == 4 {
		dt, _ = time.Parse("2006-01-02 15:04:05", strings.TrimSpace(match[1]))
		subject = strings.TrimSpace(match[2])
		sender = strings.TrimSpace(match[3])
	}
	return sender, subject, dt
}

func extractShopFromHTML(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return ""
	}
	return receipt.ExtractShop(doc)
}

func normalizeShopName(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")

	// Удаляем MIME-закодированные строки вида "= windows-1251 Q ..." или "=?utf-8?...?="
	mimeGarbage := regexp.MustCompile(`(?i)^=\s?(windows-1251|koi8-[a-z0-9-]+|utf-8).*`)
	if mimeGarbage.MatchString(s) {
		return "Неизвестно"
	}

	// Если имеется URL, усекаем строку до него
	if idx := strings.Index(s, "http"); idx != -1 {
		s = strings.TrimSpace(s[:idx])
	}

	// Если строка очень длинная (> 80) и содержит запятую — берём часть до первой запятой.
	if len([]rune(s)) > 80 {
		if idx := strings.Index(s, ","); idx > 5 {
			s = strings.TrimSpace(s[:idx])
		}
	}

	prefixRe := regexp.MustCompile(`^[0-9]{8}\s+[0-9]{6}\s+[^\s]+\s+`)
	s = prefixRe.ReplaceAllString(s, "")

	companyRe := regexp.MustCompile(`(?i)(АО|ООО|ОАО|ЗАО|ИП)`)
	if loc := companyRe.FindStringIndex(s); loc != nil && loc[0] > 0 {
		s = strings.TrimSpace(s[loc[0]:])
	}

	dateSuffixRe := regexp.MustCompile(`(?i)\s+(за|от)\s+[0-3]?[0-9]\.\d{2}\.\d{4}$`)
	s = dateSuffixRe.ReplaceAllString(s, "")

	// Усечение до 60 символов, чтобы убрать длинные служебные тексты
	if len([]rune(s)) > 60 {
		s = string([]rune(s)[:60])
	}

	// Удаляем распространённые фразы-дисклеймеры
	servicePhrases := []string{
		"это сообщение и любые документы",
		"если это сообщение",
		"данное сообщение",
		"windows-1251",
		"koi8-",
		"=?utf-8",
	}
	lower := strings.ToLower(s)
	for _, ph := range servicePhrases {
		if strings.Contains(lower, ph) {
			return "Неизвестно"
		}
	}

	// Финальная проверка
	if len([]rune(s)) < 3 {
		return "Неизвестно"
	}

	return strings.TrimSpace(s)
}

func splitHashParts(hashFull string) (folder, receiptID, hash string) {
	parts := strings.Split(hashFull, "_")
	if len(parts) < 3 {
		return hashFull, "", ""
	}
	folder = strings.Join(parts[:len(parts)-2], "_")
	receiptID = parts[len(parts)-2]
	hash = parts[len(parts)-1]
	return folder, receiptID, hash
}

// добавим helper для извлечения суммы
func extractTotalFromHTML(path string) (float64, bool) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		return 0, false
	}
	return receipt.ExtractReceiptTotal(doc)
}
