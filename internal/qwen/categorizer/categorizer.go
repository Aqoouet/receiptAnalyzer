package categorizer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"golang.org/x/text/unicode/norm"

	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/qwen"
	"receiptAnalyzer/internal/storage"
)

// Глобальная переменная для конфигурации
var globalConfig *config.Config

// StartServer запускает HTTP сервер категоризации
func StartServer() {
	// Создаем папку logs если её нет
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Ошибка создания папки logs: %v", err)
	}

	// Настраиваем логирование в файл
	logFile, err := os.OpenFile("logs/qwencategorizer.log", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Printf("Ошибка открытия файла лога: %v", err)
	} else {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	// Загружаем конфигурацию один раз при старте
	globalConfig, err = config.LoadConfig("")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	log.Printf("Конфигурация загружена для qwencategorizer")

	// Загружаем кэш категорий из файла
	loadCategoryCache()
	log.Printf("Загружено %d записей кэша категорий", len(categoryCache))

	http.HandleFunc("/categorize", categorizeHandler)

	port := os.Getenv("QWEN_PORT")
	if port == "" {
		port = fmt.Sprintf("%d", globalConfig.Ports.QwenCategorizer)
	}
	log.Printf("QwenCategorizer listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func categorizeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed", http.StatusMethodNotAllowed)
		return
	}

	stor, err := storage.NewSQLiteStorage(globalConfig.Paths.DBPath)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	db := stor.RawDB()
	if err := ensureCategoryColumn(db); err != nil {
		http.Error(w, "DB alter error: "+err.Error(), 500)
		return
	}
	items, err := getItemsWithoutCategory(db)
	if err != nil {
		http.Error(w, "DB read error: "+err.Error(), 500)
		return
	}
	if len(items) == 0 {
		log.Printf("✅ Все товары уже категоризированы - нечего добавлять")
		w.Write([]byte("No uncategorized items found"))
		return
	}

	// Разделяем товары на уже закэшированные и новые, с учётом нормализации
	var toQuery []string            // нормализованные имена, которых нет в кэше
	cats := make(map[string]string) // оригинальное имя -> категория для обновления БД
	nameVariants := make(map[string][]string)

	cacheMu.RLock()
	for _, orig := range items {
		normName := normalizeName(orig)
		if normName == "" {
			continue // пропускаем пустые
		}
		// Логируем упрощение названия
		if normName != orig {
			log.Printf("🔄 Упрощено: \"%s\" → \"%s\"", orig, normName)
		}
		nameVariants[normName] = append(nameVariants[normName], orig)

		if cat, ok := categoryCache[normName]; ok {
			cats[orig] = cat // категория уже известна
		} else {
			// в toQuery добавляем только один раз для каждого normName
			if len(nameVariants[normName]) == 1 {
				// В запрос отправляем упрощённое (нормализованное) название
				toQuery = append(toQuery, normName)
			}
		}
	}
	cacheMu.RUnlock()

	// Запрашиваем Qwen только для новых товаров
	if len(toQuery) == 0 {
		log.Printf("✅ Все %d товаров уже есть в кэше - нечего запрашивать у Qwen", len(items))
	} else {
		log.Printf("📤 Отправляем %d новых товаров в Qwen (из %d всего)", len(toQuery), len(items))
	}

	if len(toQuery) > 0 {
		const batchSize = 50 // Увеличено до 50 для более эффективной обработки
		const maxRetries = 10
		const retryDelay = 5 * time.Second   // Уменьшено с 10 до 5 секунд
		const requestDelay = 8 * time.Second // Уменьшено с 15 до 8 секунд

		for i := 0; i < len(toQuery); i += batchSize {
			end := i + batchSize
			if end > len(toQuery) {
				end = len(toQuery)
			}
			batch := toQuery[i:end]

			// Запоминаем размер кэша до обработки
			cacheMu.RLock()
			cacheSizeBefore := len(categoryCache)
			cacheMu.RUnlock()

			// Retry логика
			var freshCats map[string]string
			var err error
			for retry := 0; retry < maxRetries; retry++ {
				freshCats, err = qwen.CategorizeItems(batch)
				if err == nil && len(freshCats) > 0 {
					break // успешный ответ
				}

				// Проверяем на код 429 (rate limit)
				if err != nil && strings.Contains(err.Error(), "rate limit exceeded (429)") {
					log.Printf("🚫 Получен код 429. Прекращаем отправку запросов.")
					break // выходим из retry цикла
				}

				if retry < maxRetries-1 {
					log.Printf("Попытка %d/%d неудачна (batch %d-%d): %v, повтор через %v",
						retry+1, maxRetries, i, end, err, retryDelay)
					time.Sleep(retryDelay)
				}
			}

			// Если получили 429, прекращаем обработку всех батчей
			if err != nil && strings.Contains(err.Error(), "rate limit exceeded (429)") {
				log.Printf("🚫 Прекращаем обработку из-за rate limit (429)")
				break // выходим из цикла батчей
			}

			if err != nil || len(freshCats) == 0 {
				log.Printf("Qwen error после %d попыток (batch %d-%d): %v", maxRetries, i, end, err)
				continue // пропускаем ошибочный батч, продолжаем остальные
			}

			// Обновляем кэш и сопоставляем оригинальные варианты
			cacheMu.Lock()
			for origName, v := range freshCats {
				// Сохраняем в кэш по упрощённому названию
				normName := normalizeName(origName)
				categoryCache[normName] = v
				for _, orig := range nameVariants[normName] {
					cats[orig] = v
				}
			}
			cacheMu.Unlock()

			// Получаем размер кэша после обновления
			cacheMu.RLock()
			cacheSizeAfter := len(categoryCache)
			cacheMu.RUnlock()

			// Сохраняем кэш на диск после каждого батча
			saveCategoryCache()

			// Логируем изменение размера кэша
			cacheIncrease := cacheSizeAfter - cacheSizeBefore
			log.Printf("Обработан батч %d-%d (%d товаров), кэш: %d→%d (+%d)", i, end, len(batch), cacheSizeBefore, cacheSizeAfter, cacheIncrease)

			// Задержка между запросами для соблюдения rate limit
			if end < len(toQuery) {
				log.Printf("⏳ Ожидание %v между запросами...", requestDelay)
				time.Sleep(requestDelay)
			}
		}
	}

	// Обновляем БД
	if err := updateCategories(db, cats); err != nil {
		http.Error(w, "DB update error: "+err.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cats)
}

// ensureCategoryColumn добавляет колонку category, если её нет
func ensureCategoryColumn(db *sql.DB) error {
	rows, err := db.Query("PRAGMA table_info(items)")
	if err != nil {
		return err
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "category" {
			found = true
			break
		}
	}
	if !found {
		_, err := db.Exec("ALTER TABLE items ADD COLUMN category TEXT")
		return err
	}
	return nil
}

// getItemsWithoutCategory возвращает уникальные имена товаров без категории
func getItemsWithoutCategory(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT DISTINCT name FROM items WHERE category IS NULL OR category = ''`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	return items, nil
}

// updateCategories обновляет категории в базе
func updateCategories(db *sql.DB, cats map[string]string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`UPDATE items SET category = ? WHERE name = ?`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()
	for name, cat := range cats {
		if _, err := stmt.Exec(cat, name); err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// normalizeName приводит строку к NFC, убирает непечатываемые символы и обрезает пробелы
func normalizeName(s string) string {
	// Приводим к NFC
	s = norm.NFC.String(s)

	// Убираем все непечатываемые символы и переносы строк
	var result strings.Builder
	for _, r := range s {
		if r >= 32 && r != 127 && r != '\n' && r != '\r' && r != '\t' { // печатаемые символы без переносов
			result.WriteRune(r)
		}
	}

	// Убираем множественные пробелы и обрезаем
	cleaned := result.String()
	cleaned = strings.ReplaceAll(cleaned, "  ", " ") // заменяем двойные пробелы на одинарные
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}

	// Убираем цифры и единицы измерения
	cleaned = removeNumbersAndUnits(cleaned)

	return strings.TrimSpace(cleaned)
}

// removeNumbersAndUnits удаляет цифры и единицы измерения
func removeNumbersAndUnits(s string) string {
	// Регулярные выражения для удаления цифр и единиц
	patterns := []string{
		`\d+[.,]?\d*\s*(г|кг|л|мл|шт|шт\.|уп|пак|банк|бутыл|короб|упаковк)`, // цифры + единицы
		`\d+[.,]?\d*%`, // проценты
		`\d+[.,]?\d*`,  // просто цифры
		`[0-9]+`,       // любые цифры
	}

	result := s
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	// Убираем лишние пробелы после удаления
	result = strings.ReplaceAll(result, "  ", " ")
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}

	return result
}
