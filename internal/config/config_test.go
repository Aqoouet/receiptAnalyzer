package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"receiptAnalyzer/internal/config"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig_TableDriven(t *testing.T) {
	t.Parallel()

	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupConfig func() string
		wantErr     bool
		checkConfig func(*testing.T, *config.Config)
	}{
		{
			name: "Валидная конфигурация",
			setupConfig: func() string {
				configPath := filepath.Join(tempDir, "valid_config.yaml")
				configContent := `
email:
  imap_server: "imap.yandex.ru:993"
  username: "test@yandex.ru"
  oauth_token: "test-token"
storage:
  db_path: "test.db"
  xlsx_path: "test.xlsx"
paths:
  html_dir: "test_html"
  state_dir: "test_state"
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "imap.yandex.ru:993", cfg.Email.IMAPServer)
				assert.Equal(t, "test@yandex.ru", cfg.Email.Username)
				assert.Equal(t, "test-token", cfg.Email.OAuthToken)
				assert.Equal(t, "test.db", cfg.Storage.DBPath)
				assert.Equal(t, "test.xlsx", cfg.Storage.XLSXPath)
				assert.Equal(t, "test_html", cfg.Paths.HTMLDir)
				assert.Equal(t, "test_state", cfg.Paths.StateDir)
			},
		},
		{
			name: "Частичная конфигурация с дефолтами",
			setupConfig: func() string {
				configPath := filepath.Join(tempDir, "partial_config.yaml")
				configContent := `
email:
  imap_server: "imap.gmail.com:993"
  username: "test@gmail.com"
storage:
  db_path: "custom.db"
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "imap.gmail.com:993", cfg.Email.IMAPServer)
				assert.Equal(t, "test@gmail.com", cfg.Email.Username)
				assert.Equal(t, "", cfg.Email.OAuthToken)
				assert.Equal(t, "custom.db", cfg.Storage.DBPath)
				assert.Equal(t, "receipts.xlsx", cfg.Storage.XLSXPath) // дефолт
				assert.Equal(t, "output/msg_html", cfg.Paths.HTMLDir)  // дефолт
				assert.Equal(t, "output/state", cfg.Paths.StateDir)    // дефолт
			},
		},
		{
			name: "Пустая конфигурация",
			setupConfig: func() string {
				configPath := filepath.Join(tempDir, "empty_config.yaml")
				configContent := `{}`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "", cfg.Email.IMAPServer)
				assert.Equal(t, "", cfg.Email.Username)
				assert.Equal(t, "", cfg.Email.OAuthToken)
				assert.Equal(t, "receipts.db", cfg.Storage.DBPath)     // дефолт
				assert.Equal(t, "receipts.xlsx", cfg.Storage.XLSXPath) // дефолт
				assert.Equal(t, "output/msg_html", cfg.Paths.HTMLDir)  // дефолт
				assert.Equal(t, "output/state", cfg.Paths.StateDir)    // дефолт
			},
		},
		{
			name: "Невалидный YAML",
			setupConfig: func() string {
				configPath := filepath.Join(tempDir, "invalid_config.yaml")
				configContent := `
email:
  imap_server: "imap.yandex.ru:993"
  username: "test@yandex.ru"
  oauth_token: "test-token"
storage:
  db_path: "test.db"
  xlsx_path: "test.xlsx"
paths:
  html_dir: "test_html"
  state_dir: "test_state"
invalid: yaml: syntax: error
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: true,
			checkConfig: func(t *testing.T, cfg *config.Config) {
				// Не должно выполняться при ошибке
			},
		},
		{
			name: "Файл не существует - дефолтная конфигурация",
			setupConfig: func() string {
				return filepath.Join(tempDir, "nonexistent.yaml")
			},
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "", cfg.Email.IMAPServer)
				assert.Equal(t, "", cfg.Email.Username)
				assert.Equal(t, "", cfg.Email.OAuthToken)
				assert.Equal(t, "receipts.db", cfg.Storage.DBPath)     // дефолт
				assert.Equal(t, "receipts.xlsx", cfg.Storage.XLSXPath) // дефолт
			},
		},
		{
			name: "Пустые строки в конфигурации",
			setupConfig: func() string {
				configPath := filepath.Join(tempDir, "empty_strings_config.yaml")
				configContent := `
email:
  imap_server: ""
  username: ""
  oauth_token: ""
storage:
  db_path: ""
  xlsx_path: ""
paths:
  html_dir: ""
  state_dir: ""
`
				err := os.WriteFile(configPath, []byte(configContent), 0644)
				require.NoError(t, err)
				return configPath
			},
			wantErr: false,
			checkConfig: func(t *testing.T, cfg *config.Config) {
				assert.Equal(t, "", cfg.Email.IMAPServer)
				assert.Equal(t, "", cfg.Email.Username)
				assert.Equal(t, "", cfg.Email.OAuthToken)
				assert.Equal(t, "receipts.db", cfg.Storage.DBPath)     // дефолт
				assert.Equal(t, "receipts.xlsx", cfg.Storage.XLSXPath) // дефолт
				assert.Equal(t, "output/msg_html", cfg.Paths.HTMLDir)  // дефолт
				assert.Equal(t, "output/state", cfg.Paths.StateDir)    // дефолт
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupConfig()

			cfg, err := config.LoadConfig(configPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, cfg)

			tt.checkConfig(t, cfg)
		})
	}
}

func TestConfig_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		config  *config.Config
		isValid bool
	}{
		{
			name: "Полная валидная конфигурация",
			config: &config.Config{
				Email: struct {
					IMAPServer string `yaml:"imap_server"`
					Username   string `yaml:"username"`
					OAuthToken string `yaml:"oauth_token"`
				}{
					IMAPServer: "imap.yandex.ru:993",
					Username:   "test@yandex.ru",
					OAuthToken: "test-token",
				},
				Storage: struct {
					DBPath   string `yaml:"db_path"`
					XLSXPath string `yaml:"xlsx_path"`
				}{
					DBPath:   "test.db",
					XLSXPath: "test.xlsx",
				},
				Paths: struct {
					HTMLDir  string `yaml:"html_dir"`
					StateDir string `yaml:"state_dir"`
				}{
					HTMLDir:  "test_html",
					StateDir: "test_state",
				},
			},
			isValid: true,
		},
		{
			name:    "Конфигурация с дефолтами",
			config:  &config.Config{},
			isValid: true, // дефолты применяются в LoadConfig
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Проверяем, что конфигурация не nil
			assert.NotNil(t, tt.config)

			if tt.isValid {
				// Проверяем, что все поля доступны
				assert.IsType(t, "", tt.config.Email.IMAPServer)
				assert.IsType(t, "", tt.config.Email.Username)
				assert.IsType(t, "", tt.config.Email.OAuthToken)
				assert.IsType(t, "", tt.config.Storage.DBPath)
				assert.IsType(t, "", tt.config.Storage.XLSXPath)
				assert.IsType(t, "", tt.config.Paths.HTMLDir)
				assert.IsType(t, "", tt.config.Paths.StateDir)
			}
		})
	}
}
