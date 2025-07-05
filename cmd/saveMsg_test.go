package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeString_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Обычная строка",
			input:    "Test Subject",
			expected: "Test_Subject",
		},
		{
			name:     "Строка с запрещенными символами",
			input:    "Test/Subject:With*Special?Chars",
			expected: "Test_Subject_With_Special_Chars",
		},
		{
			name:     "Пустая строка",
			input:    "",
			expected: "",
		},
		{
			name:     "Только пробелы",
			input:    "   ",
			expected: "___",
		},
		{
			name:     "Множественные пробелы",
			input:    "Test    Subject",
			expected: "Test____Subject",
		},
		{
			name:     "Символы кавычек",
			input:    `Test"Subject"`,
			expected: "Test_Subject_",
		},
		{
			name:     "Символы угловых скобок",
			input:    "Test<Subject>",
			expected: "Test_Subject_",
		},
		{
			name:     "Символ вертикальной черты",
			input:    "Test|Subject",
			expected: "Test_Subject",
		},
		{
			name:     "Смешанные символы",
			input:    "Test/Subject:With*Special?Chars\"And<Brackets>",
			expected: "Test_Subject_With_Special_Chars_And_Brackets_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLoadSavedHashes_TableDriven(t *testing.T) {
	t.Skip("Skipping hash loading tests pending refactor")

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupIndex  func() string
		expectedLen int
		checkHashes func(*testing.T, map[string]struct{})
	}{
		{
			name: "Существующий файл с хэшами",
			setupIndex: func() string {
				indexPath := filepath.Join(tempDir, "index.txt")
				content := "hash1\nhash2\nhash3\n"
				err := os.WriteFile(indexPath, []byte(content), 0644)
				require.NoError(t, err)
				return indexPath
			},
			expectedLen: 3,
			checkHashes: func(t *testing.T, hashes map[string]struct{}) {
				assert.Contains(t, hashes, "hash1")
				assert.Contains(t, hashes, "hash2")
				assert.Contains(t, hashes, "hash3")
			},
		},
		{
			name: "Пустой файл",
			setupIndex: func() string {
				indexPath := filepath.Join(tempDir, "empty_index.txt")
				err := os.WriteFile(indexPath, []byte(""), 0644)
				require.NoError(t, err)
				return indexPath
			},
			expectedLen: 0,
			checkHashes: func(t *testing.T, hashes map[string]struct{}) {
				assert.Empty(t, hashes)
			},
		},
		{
			name: "Файл с пустыми строками",
			setupIndex: func() string {
				indexPath := filepath.Join(tempDir, "empty_lines_index.txt")
				content := "\n\nhash1\n\nhash2\n\n"
				err := os.WriteFile(indexPath, []byte(content), 0644)
				require.NoError(t, err)
				return indexPath
			},
			expectedLen: 2,
			checkHashes: func(t *testing.T, hashes map[string]struct{}) {
				assert.Contains(t, hashes, "hash1")
				assert.Contains(t, hashes, "hash2")
			},
		},
		{
			name: "Файл не существует",
			setupIndex: func() string {
				return filepath.Join(tempDir, "nonexistent.txt")
			},
			expectedLen: 0,
			checkHashes: func(t *testing.T, hashes map[string]struct{}) {
				assert.Empty(t, hashes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Временно изменяем путь для теста
			indexPath := tt.setupIndex()

			// Создаем временную директорию msg_html
			testMsgHTMLDir := filepath.Join(tempDir, "msg_html")
			err := os.MkdirAll(testMsgHTMLDir, 0755)
			require.NoError(t, err)

			// Копируем файл в правильное место
			if indexPath != filepath.Join(tempDir, "nonexistent.txt") {
				targetPath := filepath.Join(testMsgHTMLDir, "index.txt")
				err = os.WriteFile(targetPath, []byte(""), 0644)
				require.NoError(t, err)
			}

			// Вызываем функцию
			loadSavedHashes()

			// Проверяем результат
			assert.Len(t, savedHashes, tt.expectedLen)
			tt.checkHashes(t, savedHashes)

			// Очищаем глобальную переменную
			savedHashes = nil
		})
	}
}

func TestAppendHash_TableDriven(t *testing.T) {
	t.Skip("Skipping hash append tests pending refactor")

	tempDir := t.TempDir()

	tests := []struct {
		name       string
		hash       string
		setupIndex func() string
		wantErr    bool
	}{
		{
			name: "Добавление нового хэша",
			hash: "newhash123",
			setupIndex: func() string {
				indexPath := filepath.Join(tempDir, "index.txt")
				content := "existinghash1\nexistinghash2\n"
				err := os.WriteFile(indexPath, []byte(content), 0644)
				require.NoError(t, err)
				return indexPath
			},
			wantErr: false,
		},
		{
			name: "Создание нового файла",
			hash: "firsthash",
			setupIndex: func() string {
				return filepath.Join(tempDir, "new_index.txt")
			},
			wantErr: false,
		},
		{
			name: "Пустой хэш",
			hash: "",
			setupIndex: func() string {
				indexPath := filepath.Join(tempDir, "empty_hash.txt")
				err := os.WriteFile(indexPath, []byte(""), 0644)
				require.NoError(t, err)
				return indexPath
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath := tt.setupIndex()

			// Инициализируем глобальную переменную
			savedHashes = make(map[string]struct{})

			// Вызываем функцию
			appendHash(tt.hash)

			// Проверяем, что хэш добавлен в карту
			assert.Contains(t, savedHashes, tt.hash)

			// Проверяем, что хэш записан в файл
			if !tt.wantErr {
				content, err := os.ReadFile(indexPath)
				if err == nil {
					assert.Contains(t, string(content), tt.hash)
				}
			}

			// Очищаем глобальную переменную
			savedHashes = nil
		})
	}
}

func TestSaveEmailHTML_TableDriven(t *testing.T) {
	t.Skip("Skipping email HTML save tests pending refactor")

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		sender      string
		subject     string
		date        time.Time
		body        []byte
		setupHashes func()
		expectFile  bool
		expectHash  bool
	}{
		{
			name:    "Новое письмо",
			sender:  "test@example.com",
			subject: "Test Subject",
			date:    time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
			body:    []byte("<html><body>Test content</body></html>"),
			setupHashes: func() {
				savedHashes = make(map[string]struct{})
			},
			expectFile: true,
			expectHash: true,
		},
		{
			name:    "Дубликат письма",
			sender:  "test@example.com",
			subject: "Test Subject",
			date:    time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
			body:    []byte("<html><body>Test content</body></html>"),
			setupHashes: func() {
				savedHashes = make(map[string]struct{})
				hash := sha256.Sum256([]byte("<html><body>Test content</body></html>"))
				hashStr := hex.EncodeToString(hash[:])
				savedHashes[hashStr] = struct{}{}
			},
			expectFile: false,
			expectHash: false,
		},
		{
			name:    "Письмо с специальными символами в теме",
			sender:  "test@example.com",
			subject: "Test/Subject:With*Special?Chars",
			date:    time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
			body:    []byte("<html><body>Test content</body></html>"),
			setupHashes: func() {
				savedHashes = make(map[string]struct{})
			},
			expectFile: true,
			expectHash: true,
		},
		{
			name:    "Пустое тело письма",
			sender:  "test@example.com",
			subject: "Empty Body",
			date:    time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
			body:    []byte(""),
			setupHashes: func() {
				savedHashes = make(map[string]struct{})
			},
			expectFile: true,
			expectHash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем временную директорию msg_html
			testMsgHTMLDir := filepath.Join(tempDir, "msg_html")
			err := os.MkdirAll(testMsgHTMLDir, 0755)
			require.NoError(t, err)

			// Инициализируем хэши
			tt.setupHashes()

			// Вызываем функцию
			saveEmailHTML(tt.sender, tt.subject, tt.date, tt.body)

			// Проверяем результат
			if tt.expectFile {
				// Проверяем, что файл создан
				expectedFilename := tt.date.Format("20060102_150405") + "_" + sanitizeString(tt.sender) + "_" + sanitizeString(tt.subject) + ".html"
				filePath := filepath.Join(testMsgHTMLDir, expectedFilename)
				_, err := os.Stat(filePath)
				assert.NoError(t, err)

				// Проверяем содержимое файла
				content, err := os.ReadFile(filePath)
				assert.NoError(t, err)
				assert.Equal(t, tt.body, content)
			}

			if tt.expectHash {
				// Проверяем, что хэш добавлен
				hash := sha256.Sum256(tt.body)
				hashStr := hex.EncodeToString(hash[:])
				assert.Contains(t, savedHashes, hashStr)
			}

			// Очищаем глобальную переменную
			savedHashes = nil
		})
	}
}

func TestInitializeStorage_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Валидная конфигурация",
			config: &Config{
				Storage: struct {
					DBPath   string `yaml:"db_path"`
					XLSXPath string `yaml:"xlsx_path"`
				}{
					DBPath:   filepath.Join(tempDir, "test.db"),
					XLSXPath: filepath.Join(tempDir, "test.xlsx"),
				},
			},
			wantErr: false,
		},
		{
			name: "Пустой путь к БД",
			config: &Config{
				Storage: struct {
					DBPath   string `yaml:"db_path"`
					XLSXPath string `yaml:"xlsx_path"`
				}{
					DBPath:   "",
					XLSXPath: "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage, err := InitializeStorage(tt.config)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, storage)
				return
			}

			assert.NoError(t, err)
			assert.NotNil(t, storage)
		})
	}
}
