package htmlimporter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"receiptAnalyzer/internal/config"
	mc "receiptAnalyzer/internal/htmlimporter/manual_correction"
	"receiptAnalyzer/internal/htmlimporter/shop"
	"receiptAnalyzer/internal/storage"
)

// Глобальная переменная для конфигурации
var globalConfig *config.Config

func importHTMLHandler(w http.ResponseWriter, r *http.Request) {
	store, err := storage.NewSQLiteStorage(globalConfig.Paths.DBPath)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), 500)
		return
	}
	imported, skipped, impErr := ImportSavedHTML(store, globalConfig.Paths.HTMLDirPath)
	resp := struct {
		Imported int    `json:"imported"`
		Skipped  int    `json:"skipped"`
		Error    string `json:"error,omitempty"`
	}{
		Imported: imported,
		Skipped:  skipped,
	}
	if impErr != nil {
		resp.Error = impErr.Error()
		w.WriteHeader(500)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("/import-html: imported=%d, skipped=%d, error=%v", imported, skipped, impErr)
}

func deleteDBHandler(w http.ResponseWriter, r *http.Request) {
	dbPath := globalConfig.Paths.DBPath
	err := os.Remove(dbPath)
	if err != nil {
		http.Error(w, "DB delete error: "+err.Error(), 500)
		return
	}
	w.Write([]byte("DB deleted"))
}

func updateDBHandler(w http.ResponseWriter, r *http.Request) {
	_, err := os.Stat(globalConfig.Paths.DBPath)
	if err == nil {
		// DB exists, try to open and recreate tables
		store, err := storage.NewSQLiteStorage(globalConfig.Paths.DBPath)
		if err != nil {
			http.Error(w, "DB open error: "+err.Error(), 500)
			return
		}
		_ = store // таблицы пересоздаются автоматически
	} else {
		// DB does not exist, will be created on next import
	}
	w.Write([]byte("DB structure updated"))
}

func updateShopsHandler(w http.ResponseWriter, r *http.Request) {
	// Открываем БД
	store, err := storage.NewSQLiteStorage(globalConfig.Paths.DBPath)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer store.Close()
	db := store.RawDB()

	// Обновляем магазины
	updated, skipped, err := shop.UpdateShops(db)
	if err != nil {
		http.Error(w, "Update shops error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		Updated int `json:"updated"`
		Skipped int `json:"skipped"`
	}{Updated: updated, Skipped: skipped}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("/update-shops: updated=%d, skipped=%d", updated, skipped)
}

func clearShopsHandler(w http.ResponseWriter, r *http.Request) {
	// Открываем БД
	store, err := storage.NewSQLiteStorage(globalConfig.Paths.DBPath)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer store.Close()
	db := store.RawDB()

	// Очищаем магазины
	cleared, err := shop.ClearShops(db)
	if err != nil {
		http.Error(w, "Clear shops error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		Cleared int `json:"cleared"`
	}{Cleared: cleared}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("/clear-shops: cleared=%d", cleared)
}

func manualCorrectionsHandler(w http.ResponseWriter, r *http.Request) {
	correctionDir := "internal/htmlimporter/manual_correction" // директория с JSON-файлами исправлений

	// Открываем БД
	store, err := storage.NewSQLiteStorage(globalConfig.Paths.DBPath)
	if err != nil {
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer store.Close()

	// Применяем исправления
	receiptsUpdated, itemsUpdated, err := mc.ApplyManualCorrections(store.RawDB(), correctionDir)
	if err != nil {
		http.Error(w, "Apply corrections error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	resp := struct {
		ReceiptsUpdated int `json:"receipts_updated"`
		ItemsUpdated    int `json:"items_updated"`
	}{ReceiptsUpdated: receiptsUpdated, ItemsUpdated: itemsUpdated}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
	log.Printf("/apply-corrections: receipts_updated=%d, items_updated=%d", receiptsUpdated, itemsUpdated)
}

func StartServer() {
	// Загружаем конфигурацию один раз при старте
	var err error
	globalConfig, err = config.LoadConfig("")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	log.Printf("Конфигурация загружена для htmlimporter")

	http.HandleFunc("/import-html", importHTMLHandler)
	http.HandleFunc("/delete-db", deleteDBHandler)
	http.HandleFunc("/update-db", updateDBHandler)
	http.HandleFunc("/update-shops", updateShopsHandler)
	http.HandleFunc("/clear-shops", clearShopsHandler)
	http.HandleFunc("/apply-corrections", manualCorrectionsHandler)

	port := os.Getenv("HTMLIMPORTER_PORT")
	if port == "" {
		port = fmt.Sprintf("%d", globalConfig.Ports.HTMLImporter)
	}
	log.Printf("Htmlimporter listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
