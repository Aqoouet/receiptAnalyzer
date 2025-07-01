package main

import (
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
