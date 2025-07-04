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
		DBPath string `yaml:"db_path"`
	} `yaml:"storage"`
}

func LoadConfig(path string) (*Config, error) {
	log.Printf("Читаем конфигурацию из %s", path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	log.Printf("Конфигурация загружена: IMAP=%s, пользователь=%s, БД=%s", cfg.Email.IMAPServer, cfg.Email.Username, cfg.Storage.DBPath)

	return &cfg, nil
}
