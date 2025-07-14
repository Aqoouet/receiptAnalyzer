package htmlimporter

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"receiptAnalyzer/internal/config"
)

func importHTMLHandler(w http.ResponseWriter, r *http.Request) {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		http.Error(w, "Config error: "+err.Error(), 500)
		return
	}
	err = ImportAll(cfg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("OK"))
}

func StartServer() {
	http.HandleFunc("/import-html", importHTMLHandler)
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	port := os.Getenv("HTMLIMPORTER_PORT")
	if port == "" {
		port = fmt.Sprintf("%d", cfg.Ports.HTMLImporter)
	}
	log.Printf("Htmlimporter listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
