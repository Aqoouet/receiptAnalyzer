package xlsxexporter

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"receiptAnalyzer/internal/config"
)

// Глобальная переменная для конфигурации
var globalConfig *config.Config

func StartServer() {
	// Загружаем конфигурацию один раз при старте
	var err error
	globalConfig, err = config.LoadConfig("")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	log.Printf("Конфигурация загружена для xlsxexporter")

	http.HandleFunc("/export-xlsx", exportXLSXHandler)

	port := os.Getenv("XLSXEXPORTER_PORT")
	if port == "" {
		port = fmt.Sprintf("%d", globalConfig.Ports.XLSXExporter)
	}
	log.Printf("Xlsxexporter listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func exportXLSXHandler(w http.ResponseWriter, r *http.Request) {
	err := ExportAll(globalConfig)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("OK"))
}
