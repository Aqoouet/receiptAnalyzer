package orchestrator

import (
	"fmt"
	"io"
	"net/http"
	"receiptAnalyzer/internal/config"
)

// RunAllServices вызывает все микросервисы по порядку
func RunAllServices() error {
	cfg, err := config.LoadConfig("")
	if err != nil {
		return fmt.Errorf("ошибка загрузки конфига: %w", err)
	}

	return RunAllServicesWithConfig(cfg)
}

// RunAllServicesWithConfig вызывает все микросервисы по порядку с переданной конфигурацией
func RunAllServicesWithConfig(cfg *config.Config) error {
	steps := []struct {
		name   string
		url    string
		method string
	}{
		{"fetch-emails", fmt.Sprintf("http://localhost:%d/fetch-emails", cfg.Ports.Mailfetcher), "GET"},
		{"import-html", fmt.Sprintf("http://localhost:%d/import-html", cfg.Ports.HTMLImporter), "GET"},
		{"xlsx-export", fmt.Sprintf("http://localhost:%d/export-xlsx", cfg.Ports.XLSXExporter), "GET"},
		{"categorize-items", fmt.Sprintf("http://localhost:%d/categorize", cfg.Ports.QwenCategorizer), "POST"},
	}
	for _, step := range steps {
		var resp *http.Response
		var err error
		if step.method == "POST" {
			resp, err = http.Post(step.url, "application/json", nil)
		} else {
			resp, err = http.Get(step.url)
		}
		if err != nil {
			return fmt.Errorf("ошибка %s: %w", step.name, err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
	return nil
}
