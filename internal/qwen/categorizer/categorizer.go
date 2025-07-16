package categorizer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/qwen"
	"receiptAnalyzer/internal/storage"
)

// Глобальная переменная для конфигурации
var globalConfig *config.Config

// StartServer запускает HTTP сервер категоризации
func StartServer() {
	// Загружаем конфигурацию один раз при старте
	var err error
	globalConfig, err = config.LoadConfig("")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	log.Printf("Конфигурация загружена для qwencategorizer")

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
		w.Write([]byte("No uncategorized items found"))
		return
	}
	cats, err := qwen.CategorizeItems(items)
	if err != nil {
		http.Error(w, "Qwen error: "+err.Error(), 500)
		return
	}
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
