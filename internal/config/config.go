package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Email struct {
		IMAPServer string `yaml:"imap_server"`
		Username   string `yaml:"username"`
		OAuthToken string `yaml:"oauth_token"`
	} `yaml:"email"`
	Paths struct {
		DBPath      string `yaml:"db_path"`
		HTMLDirPath string `yaml:"html_dir_path"`
		XLSXPath    string `yaml:"xlsx_path"`
	} `yaml:"paths"`
	Ports struct {
		Mailfetcher     int `yaml:"mailfetcher"`
		HTMLImporter    int `yaml:"htmlimporter"`
		XLSXExporter    int `yaml:"xlsxexporter"`
		QwenCategorizer int `yaml:"qwencategorizer"`
	} `yaml:"ports"`
	MailboxPrefixes []string `yaml:"mailbox_prefixes"`
	SearchKeywords  []string `yaml:"search_keywords"`
}

func LoadConfig(path string) (*Config, error) {
	// Если путь не указан, используем переменную окружения или дефолт
	if path == "" {
		if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
			path = configPath
		} else {
			path = "config.yaml"
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// файл конфигурации отсутствует — возвращаем конфиг по умолчанию
			log.Printf("Файл %s не найден — используем значения по умолчанию", path)
			return &Config{
				Paths: struct {
					DBPath      string `yaml:"db_path"`
					HTMLDirPath string `yaml:"html_dir_path"`
					XLSXPath    string `yaml:"xlsx_path"`
				}{DBPath: "output/db_dir/receipts.db", HTMLDirPath: "output/msg_html", XLSXPath: "output/db_dir/receipts.xlsx"},
				Ports: struct {
					Mailfetcher     int `yaml:"mailfetcher"`
					HTMLImporter    int `yaml:"htmlimporter"`
					XLSXExporter    int `yaml:"xlsxexporter"`
					QwenCategorizer int `yaml:"qwencategorizer"`
				}{Mailfetcher: 8081, HTMLImporter: 8082, XLSXExporter: 8083, QwenCategorizer: 8084},
				MailboxPrefixes: []string{"INBOX", "Receipts|"},
				SearchKeywords:  []string{"чек", "ЧЕК", "Чек"},
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// применяем дефолты, если не указаны
	if len(cfg.Paths.DBPath) == 0 {
		cfg.Paths.DBPath = "output/db_dir/receipts.db"
	}
	if len(cfg.Paths.HTMLDirPath) == 0 {
		cfg.Paths.HTMLDirPath = "output/msg_html"
	}
	if len(cfg.Paths.XLSXPath) == 0 {
		cfg.Paths.XLSXPath = "output/db_dir/receipts.xlsx"
	}
	if len(cfg.MailboxPrefixes) == 0 {
		cfg.MailboxPrefixes = []string{"INBOX", "Receipts|"}
	}
	if len(cfg.SearchKeywords) == 0 {
		cfg.SearchKeywords = []string{"чек", "ЧЕК", "Чек"}
	}
	// Формируем полный путь к базе
	if len(cfg.Paths.DBPath) > 0 && len(cfg.Paths.DBPath) > 0 {
		cfg.Paths.DBPath = cfg.Paths.DBPath
	}

	// Логируем только при первой загрузке (можно убрать если не нужно)
	// log.Printf("Конфигурация загружена: IMAP=%s, пользователь=%s, БД=%s, HTML=%s, Порты: mailfetcher=%d, htmlimporter=%d, xlsxexporter=%d, qwencategorizer=%d, mailbox_prefixes=%v",
	//	cfg.Email.IMAPServer, cfg.Email.Username, cfg.Paths.DBPath, cfg.Paths.HTMLDirPath,
	//	cfg.Ports.Mailfetcher, cfg.Ports.HTMLImporter, cfg.Ports.XLSXExporter, cfg.Ports.QwenCategorizer, cfg.MailboxPrefixes)

	return &cfg, nil
}
