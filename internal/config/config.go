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
	Storage struct {
		DBPath   string `yaml:"db_path"`
		XLSXPath string `yaml:"xlsx_path"`
	} `yaml:"storage"`
	Paths struct {
		HTMLDir  string `yaml:"html_dir"`
		StateDir string `yaml:"state_dir"`
	} `yaml:"paths"`
	Ports struct {
		Mailfetcher     int `yaml:"mailfetcher"`
		HTMLImporter    int `yaml:"htmlimporter"`
		XLSXExporter    int `yaml:"xlsxexporter"`
		QwenCategorizer int `yaml:"qwencategorizer"`
	} `yaml:"ports"`
	MailboxPrefixes []string `yaml:"mailbox_prefixes"`
}

func LoadConfig(path string) (*Config, error) {
	log.Printf("Читаем конфигурацию из %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// файл конфигурации отсутствует — возвращаем конфиг по умолчанию
			log.Printf("Файл %s не найден — используем значения по умолчанию", path)
			return &Config{
				Storage: struct {
					DBPath   string `yaml:"db_path"`
					XLSXPath string `yaml:"xlsx_path"`
				}{DBPath: "receipts.db", XLSXPath: "receipts.xlsx"},
				Ports: struct {
					Mailfetcher     int `yaml:"mailfetcher"`
					HTMLImporter    int `yaml:"htmlimporter"`
					XLSXExporter    int `yaml:"xlsxexporter"`
					QwenCategorizer int `yaml:"qwencategorizer"`
				}{Mailfetcher: 8081, HTMLImporter: 8082, XLSXExporter: 8083, QwenCategorizer: 8084},
				MailboxPrefixes: []string{"INBOX", "Receipts|"},
			}, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	// применяем дефолты, если не указаны
	if cfg.Storage.DBPath == "" {
		cfg.Storage.DBPath = "receipts.db"
	}
	if cfg.Storage.XLSXPath == "" {
		cfg.Storage.XLSXPath = "receipts.xlsx"
	}
	if cfg.Paths.HTMLDir == "" {
		cfg.Paths.HTMLDir = "output/msg_html"
	}
	if cfg.Paths.StateDir == "" {
		cfg.Paths.StateDir = "output/state"
	}
	if cfg.Ports.Mailfetcher == 0 {
		cfg.Ports.Mailfetcher = 8081
	}
	if cfg.Ports.HTMLImporter == 0 {
		cfg.Ports.HTMLImporter = 8082
	}
	if cfg.Ports.XLSXExporter == 0 {
		cfg.Ports.XLSXExporter = 8083
	}
	if cfg.Ports.QwenCategorizer == 0 {
		cfg.Ports.QwenCategorizer = 8084
	}
	if len(cfg.MailboxPrefixes) == 0 {
		cfg.MailboxPrefixes = []string{"INBOX", "Receipts|"}
	}

	log.Printf("Конфигурация загружена: IMAP=%s, пользователь=%s, БД=%s, HTML=%s, Порты: mailfetcher=%d, htmlimporter=%d, xlsxexporter=%d, qwencategorizer=%d, mailbox_prefixes=%v",
		cfg.Email.IMAPServer, cfg.Email.Username, cfg.Storage.DBPath, cfg.Paths.HTMLDir,
		cfg.Ports.Mailfetcher, cfg.Ports.HTMLImporter, cfg.Ports.XLSXExporter, cfg.Ports.QwenCategorizer, cfg.MailboxPrefixes)

	return &cfg, nil
}
