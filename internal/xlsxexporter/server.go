package xlsxexporter

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"receiptAnalyzer/internal/config"
)

func StartServer() {
	http.HandleFunc("/export-xlsx", exportXLSXHandler)
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	port := os.Getenv("XLSXEXPORTER_PORT")
	if port == "" {
		port = fmt.Sprintf("%d", cfg.Ports.XLSXExporter)
	}
	log.Printf("Xlsxexporter listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func exportXLSXHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		http.Error(w, "Config error: "+err.Error(), 500)
		return
	}
	err = ExportAll(cfg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("OK"))
}
