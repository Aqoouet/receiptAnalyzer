package manualcorrection

// импортируемых пакеты и код из предыдущей версии
import (
	"database/sql"
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ItemCorrection описывает исправление для позиции товара
// Если Total == 0, то он будет вычислен как Quantity * UnitPrice
// Category, SubQuantity, NameCleaned опускаются, т.к. для ручных правок не нужны.
type ItemCorrection struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Total     float64 `json:"total,omitempty"`
}

// ReceiptFieldCorrection описывает исправление для конкретного поля чека
// Позволяет обновить любое поле в таблице receipts
type ReceiptFieldCorrection struct {
	Field string      `json:"field"` // название поля в БД (folder, sender, subject, date_time, is_receipt, link, template, total, delta_sum)
	Value interface{} `json:"value"` // новое значение поля
}

// ReceiptCorrection описывает исправление для конкретного чека
// Hash обязателен. Можно задать новые Items, новые значения полей и/или новое значение Total.
// Если Items заданы, старые позиции будут удалены и заменены новыми.
// Если Total не задан, но заданы Items, то Total будет вычислен как сумма total по Items.
// delta_sum при этом обнуляется.
// Fields позволяет обновить любые поля в таблице receipts.
type ReceiptCorrection struct {
	Hash   string                   `json:"hash"`
	Total  *float64                 `json:"total,omitempty"`
	Items  []ItemCorrection         `json:"items,omitempty"`
	Fields []ReceiptFieldCorrection `json:"fields,omitempty"`
}

// ApplyManualCorrections читает все *.json файлы из директории dir и
// применяет исправления к базе. Возвращает количество обновлённых чеков и позиций.
func ApplyManualCorrections(db *sql.DB, dir string) (receiptsUpdated int, itemsUpdated int, err error) {
	log.Printf("[ApplyManualCorrections] Начинаем обработку директории: %s", dir)

	if _, statErr := os.Stat(dir); os.IsNotExist(statErr) {
		log.Printf("[ApplyManualCorrections] директория %s отсутствует, ничего не делаем", dir)
		return 0, 0, nil
	}

	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(d.Name()) != ".json" {
			return nil
		}
		log.Printf("[ApplyManualCorrections] обрабатываем файл %s", path)

		data, readErr := os.ReadFile(path)
		if readErr != nil {
			log.Printf("[ApplyManualCorrections] ошибка чтения %s: %v", path, readErr)
			return nil
		}

		var corrections []ReceiptCorrection
		if unmarshalErr := json.Unmarshal(data, &corrections); unmarshalErr != nil {
			log.Printf("[ApplyManualCorrections] ошибка JSON %s: %v", path, unmarshalErr)
			return nil
		}

		log.Printf("[ApplyManualCorrections] найдено %d исправлений в файле %s", len(corrections), path)

		for _, corr := range corrections {
			if corr.Hash == "" {
				log.Printf("[ApplyManualCorrections] пропускаем запись с пустым hash")
				continue
			}
			log.Printf("[ApplyManualCorrections] обрабатываем чек с hash: %s", corr.Hash)

			tx, txErr := db.Begin()
			if txErr != nil {
				log.Printf("[ApplyManualCorrections] ошибка начала транзакции: %v", txErr)
				continue
			}

			var sumItems float64
			if len(corr.Items) > 0 {
				log.Printf("[ApplyManualCorrections] удаляем старые items для hash: %s", corr.Hash)
				if _, delErr := tx.Exec("DELETE FROM items WHERE hash = ?", corr.Hash); delErr != nil {
					_ = tx.Rollback()
					log.Printf("[ApplyManualCorrections] ошибка удаления items: %v", delErr)
					continue
				}

				stmt, prepErr := tx.Prepare(`INSERT INTO items (hash, name, quantity, unit_price, total, category, sub_quantity, name_cleaned) VALUES (?, ?, ?, ?, ?, '', '', '')`)
				if prepErr != nil {
					_ = tx.Rollback()
					log.Printf("[ApplyManualCorrections] ошибка подготовки INSERT: %v", prepErr)
					continue
				}
				for _, it := range corr.Items {
					totalVal := it.Total
					if totalVal == 0 {
						totalVal = it.Quantity * it.UnitPrice
					}
					sumItems += totalVal
					if _, insErr := stmt.Exec(corr.Hash, it.Name, it.Quantity, it.UnitPrice, totalVal); insErr != nil {
						_ = stmt.Close()
						_ = tx.Rollback()
						log.Printf("[ApplyManualCorrections] ошибка вставки item: %v", insErr)
						continue
					}
				}
				_ = stmt.Close()
				itemsUpdated++
				log.Printf("[ApplyManualCorrections] добавлено %d items для hash: %s", len(corr.Items), corr.Hash)
			}

			// Обрабатываем обновление полей чека
			if len(corr.Fields) > 0 {
				log.Printf("[ApplyManualCorrections] обновляем поля для hash: %s", corr.Hash)

				// Строим динамический UPDATE запрос
				var setClauses []string
				var values []interface{}

				for _, field := range corr.Fields {
					// Проверяем, что поле разрешено для обновления
					allowedFields := map[string]bool{
						"folder":     true,
						"receipt_id": true,
						"sender":     true,
						"date_time":  true,
						"subject":    true,
						"is_receipt": true,
						"link":       true,
						"template":   true,
						"total":      true,
						"delta_sum":  true,
						"shop":       true, // Поддержка поля shop, если оно будет добавлено
					}

					if !allowedFields[field.Field] {
						log.Printf("[ApplyManualCorrections] пропускаем неразрешенное поле: %s", field.Field)
						continue
					}

					setClauses = append(setClauses, field.Field+" = ?")

					// Обрабатываем специальные типы данных
					switch field.Field {
					case "date_time":
						if dateStr, ok := field.Value.(string); ok {
							if parsedTime, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
								values = append(values, parsedTime)
							} else {
								log.Printf("[ApplyManualCorrections] ошибка парсинга даты %s: %v", dateStr, err)
								continue
							}
						} else {
							values = append(values, field.Value)
						}
					case "is_receipt":
						if boolVal, ok := field.Value.(bool); ok {
							values = append(values, boolVal)
						} else {
							log.Printf("[ApplyManualCorrections] ошибка типа для is_receipt: ожидается bool")
							continue
						}
					default:
						values = append(values, field.Value)
					}
				}

				if len(setClauses) > 0 {
					query := "UPDATE receipts SET " + strings.Join(setClauses, ", ") + " WHERE hash = ?"
					values = append(values, corr.Hash)

					if _, upErr := tx.Exec(query, values...); upErr != nil {
						_ = tx.Rollback()
						log.Printf("[ApplyManualCorrections] ошибка UPDATE receipts fields: %v", upErr)
						continue
					}
					log.Printf("[ApplyManualCorrections] обновлены поля: %v", setClauses)
				}
			}

			// Обрабатываем обновление total
			var newTotal *float64 = corr.Total
			if newTotal == nil && len(corr.Items) > 0 {
				newTotal = &sumItems
			}
			if newTotal != nil {
				log.Printf("[ApplyManualCorrections] обновляем total для hash: %s на %.2f", corr.Hash, *newTotal)
				if _, upErr := tx.Exec("UPDATE receipts SET total = ?, delta_sum = 0 WHERE hash = ?", *newTotal, corr.Hash); upErr != nil {
					_ = tx.Rollback()
					log.Printf("[ApplyManualCorrections] ошибка UPDATE receipts: %v", upErr)
					continue
				}
			}

			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("[ApplyManualCorrections] ошибка commit: %v", commitErr)
				continue
			}
			receiptsUpdated++
			log.Printf("[ApplyManualCorrections] успешно обновлен чек с hash: %s", corr.Hash)
		}
		return nil
	})

	log.Printf("[ApplyManualCorrections] Итого обновлено: receipts=%d, items=%d", receiptsUpdated, itemsUpdated)
	return receiptsUpdated, itemsUpdated, err
}
