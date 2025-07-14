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
	files, err := ioutil.ReadDir(htmlDir)
	if err != nil {
		return 0, 0, fmt.Errorf("каталог %s недоступен: %w", htmlDir, err)
	}

	imported := 0
	skipped := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		log.Printf("Обработка файла %s", f.Name())
		id := f.Name()

		// --- парсинг чека и позиций ---
		itemsJSON, _, perr := receipt.ParseReceiptAuto(filepath.Join(htmlDir, f.Name()), []receipt.Template{
			receipt.DefaultBeelineTemplate,
			receipt.DefaultTaxcomTemplate,
			receipt.DefaultMusicTemplate,
			receipt.DefaultYandexOFDTemplate,
			receipt.DefaultYandexMarketTemplate,
			receipt.DefaultOFDruTemplate,
			receipt.DefaultFirstOFDTemplate,
			receipt.DefaultPlatformaOFDTemplate,
		})
		if perr != nil {
			log.Printf("Пропуск — не чек или неизвестный шаблон: %v", perr)
			skipped++
			continue
		}
		var items []receipt.Item
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil || len(items) == 0 {
			log.Printf("Пропуск — не удалось декодировать позиции или их нет")
			skipped++
			continue
		}

		// --- метаданные из имени файла ---
		shop, dt, _ := inferMetaFromFilename(f.Name())
		if dt.IsZero() {
			dt = f.ModTime()
		}

		// Если не удалось извлечь название магазина из имени файла, пробуем fallback
		if shop == "" {
			shopFilename := strings.TrimSuffix(f.Name(), ".html")
			shopFilename = strings.ReplaceAll(shopFilename, "_", " ")
			shopFilename = regexp.MustCompile(`\s+`).ReplaceAllString(shopFilename, " ")
			shop = strings.TrimSpace(shopFilename)
		}

		// --- попытка извлечь название магазина из HTML ---
		rawShop := extractShopFromHTML(filepath.Join(htmlDir, f.Name()))
		rawShop = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(rawShop, " "))
		if rawShop != "" {
			shop = rawShop
		}

		// Финальная проверка и fallback
		if strings.TrimSpace(shop) == "" {
			shop = "Неизвестно"
		}

		// финальная нормализация названия магазина — убираем суффиксы вида
		// "за 30.09.2024" или "от 30.09.2024" в конце строки.
		shop = normalizeShopName(shop)

		// --- итоговая сумма ---
		var total float64
		for _, it := range items {
			p, _ := receipt.ParseFloat(it.Price)
			q, _ := receipt.ParseFloat(it.Quantity)
			total += p * q
		}

		// Пропускаем, если итоговая сумма 0 – вероятно, файл не является чеком.
		if total == 0 {
			log.Printf("Пропуск — итоговая сумма 0, похоже, это не чек")
			skipped++
			continue
		}

		// Пропускаем, если сумма выглядит подозрительно большой (> 1 000 000 руб)
		if total > 1_000_000 {
			log.Printf("Пропуск — итоговая сумма %.2f слишком велика, вероятно, ошибка парсинга", total)
			skipped++
			continue
		}

		r := receipt.Receipt{
			ID:        id,
			Shop:      shop,
			DateTime:  dt,
			Total:     total,
			Source:    "saved_html",
			CreatedAt: time.Now(),
		}

		if err := store.SaveReceipt(r); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				if shop != "" && shop != "Неизвестно" {
					sqlDB := store.(*storage.SQLiteStorage).RawDB()
					sqlDB.Exec("UPDATE receipts SET shop=? WHERE id=? AND (shop='' OR shop='Неизвестно')", shop, id)
				}
				log.Printf("Чек %s уже существует, пропускаем", id)
				skipped++
				continue
			}
			return imported, skipped, fmt.Errorf("ошибка сохранения чека %s: %w", id, err)
		}
		if err := store.SaveItems(id, items); err != nil {
			log.Printf("Ошибка сохранения позиций по чеку %s: %v", id, err)
		}
		imported++
		log.Printf("Чек %s добавлен: магазин=%s, позиций=%d, сумма=%.2f", id, shop, len(items), total)
	}
	log.Printf("Итого: добавлено %d чеков, пропущено %d", imported, skipped)
	return imported, skipped, nil
}

// ImportAll импортирует все новые HTML-файлы в базу
func ImportAll(cfg *config.Config) error {
	htmlDir := cfg.Paths.HTMLDir
	store, err := storage.NewSQLiteStorage(cfg.Storage.DBPath)
	if err != nil {
		return fmt.Errorf("ошибка инициализации хранилища: %w", err)
	}
	files, err := ioutil.ReadDir(htmlDir)
	if err != nil {
		return fmt.Errorf("каталог %s недоступен: %w", htmlDir, err)
	}
	imported := 0
	skipped := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		log.Printf("Обработка файла %s", f.Name())
		id := f.Name()
		// --- парсинг чека и позиций ---
		itemsJSON, _, perr := receipt.ParseReceiptAuto(filepath.Join(htmlDir, f.Name()), []receipt.Template{
			receipt.DefaultBeelineTemplate,
			receipt.DefaultTaxcomTemplate,
			receipt.DefaultMusicTemplate,
			receipt.DefaultYandexOFDTemplate,
			receipt.DefaultYandexMarketTemplate,
			receipt.DefaultOFDruTemplate,
			receipt.DefaultFirstOFDTemplate,
			receipt.DefaultPlatformaOFDTemplate,
		})
		if perr != nil {
			log.Printf("Пропуск — не чек или неизвестный шаблон: %v", perr)
			skipped++
			continue
		}
		var items []receipt.Item
		if err := json.Unmarshal([]byte(itemsJSON), &items); err != nil || len(items) == 0 {
			log.Printf("Пропуск — не удалось декодировать позиции или их нет")
			skipped++
			continue
		}
		// --- метаданные из имени файла ---
		shop, dt, _ := inferMetaFromFilename(f.Name())
		if dt.IsZero() {
			dt = f.ModTime()
		}
		if shop == "" {
			shopFilename := strings.TrimSuffix(f.Name(), ".html")
			shopFilename = strings.ReplaceAll(shopFilename, "_", " ")
			shopFilename = regexp.MustCompile(`\s+`).ReplaceAllString(shopFilename, " ")
			shop = strings.TrimSpace(shopFilename)
		}
		rawShop := extractShopFromHTML(filepath.Join(htmlDir, f.Name()))
		rawShop = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(rawShop, " "))
		if rawShop != "" {
			shop = rawShop
		}
		if strings.TrimSpace(shop) == "" {
			shop = "Неизвестно"
		}
		shop = normalizeShopName(shop)
		var total float64
		for _, it := range items {
			p, _ := receipt.ParseFloat(it.Price)
			q, _ := receipt.ParseFloat(it.Quantity)
			total += p * q
		}
		if total == 0 {
			log.Printf("Пропуск — итоговая сумма 0, похоже, это не чек")
			skipped++
			continue
		}
		if total > 1_000_000 {
			log.Printf("Пропуск — итоговая сумма %.2f слишком велика, вероятно, ошибка парсинга", total)
			skipped++
			continue
		}
		r := receipt.Receipt{
			ID:        id,
			Shop:      shop,
			DateTime:  dt,
			Total:     total,
			Source:    "saved_html",
			CreatedAt: time.Now(),
		}
		if err := store.SaveReceipt(r); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				if shop != "" && shop != "Неизвестно" {
					sqlDB := store.RawDB()
					sqlDB.Exec("UPDATE receipts SET shop=? WHERE id=? AND (shop='' OR shop='Неизвестно')", shop, id)
				}
				log.Printf("Чек %s уже существует, пропускаем", id)
				skipped++
				continue
			}
			return fmt.Errorf("ошибка сохранения чека %s: %w", id, err)
		}
		if err := store.SaveItems(id, items); err != nil {
			log.Printf("Ошибка сохранения позиций по чеку %s: %v", id, err)
		}
		imported++
		log.Printf("Чек %s добавлен: магазин=%s, позиций=%d, сумма=%.2f", id, shop, len(items), total)
	}
	log.Printf("Итого: добавлено %d чеков, пропущено %d", imported, skipped)
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
