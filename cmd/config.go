package main

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
}

func LoadConfig(path string) (*Config, error) {
	log.Printf("Читаем конфигурацию из %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// файл конфигурации отсутствует — возвращаем конфиг по умолчанию
			log.Printf("Файл %s не найден — используем значения по умолчанию", path)
			return &Config{Storage: struct {
				DBPath   string `yaml:"db_path"`
				XLSXPath string `yaml:"xlsx_path"`
			}{DBPath: "receipts.db", XLSXPath: "receipts.xlsx"}}, nil
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
		cfg.Paths.HTMLDir = "cmd/msg_html"
	}
	if cfg.Paths.StateDir == "" {
		cfg.Paths.StateDir = "cmd/state"
	}

	log.Printf("Конфигурация загружена: IMAP=%s, пользователь=%s, БД=%s, HTML=%s", cfg.Email.IMAPServer, cfg.Email.Username, cfg.Storage.DBPath, cfg.Paths.HTMLDir)

	return &cfg, nil
}
