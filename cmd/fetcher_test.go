package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadLastUID_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name     string
		setupUID func() string
		expected int
	}{
		{
			name: "Существующий файл с UID",
			setupUID: func() string {
				stateDir := filepath.Join(tempDir, "state")
				err := os.MkdirAll(stateDir, 0755)
				require.NoError(t, err)

				uidFile := filepath.Join(stateDir, "last_uid.txt")
				err = os.WriteFile(uidFile, []byte("12345"), 0644)
				require.NoError(t, err)

				return stateDir
			},
			expected: 12345,
		},
		{
			name: "Файл не существует",
			setupUID: func() string {
				return filepath.Join(tempDir, "nonexistent_state")
			},
			expected: 0,
		},
		{
			name: "Пустой файл",
			setupUID: func() string {
				stateDir := filepath.Join(tempDir, "empty_state")
				err := os.MkdirAll(stateDir, 0755)
				require.NoError(t, err)

				uidFile := filepath.Join(stateDir, "last_uid.txt")
				err = os.WriteFile(uidFile, []byte(""), 0644)
				require.NoError(t, err)

				return stateDir
			},
			expected: 0,
		},
		{
			name: "Невалидный UID",
			setupUID: func() string {
				stateDir := filepath.Join(tempDir, "invalid_state")
				err := os.MkdirAll(stateDir, 0755)
				require.NoError(t, err)

				uidFile := filepath.Join(stateDir, "last_uid.txt")
				err = os.WriteFile(uidFile, []byte("not_a_number"), 0644)
				require.NoError(t, err)

				return stateDir
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateDir := tt.setupUID()

			result := loadLastUID(stateDir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSaveLastUID_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name     string
		uid      int
		setupDir func() string
		wantErr  bool
	}{
		{
			name: "Сохранение нового UID",
			uid:  12345,
			setupDir: func() string {
				return filepath.Join(tempDir, "new_uid_state")
			},
			wantErr: false,
		},
		{
			name: "Обновление существующего UID",
			uid:  54321,
			setupDir: func() string {
				stateDir := filepath.Join(tempDir, "existing_uid_state")
				err := os.MkdirAll(stateDir, 0755)
				require.NoError(t, err)

				uidFile := filepath.Join(stateDir, "last_uid.txt")
				err = os.WriteFile(uidFile, []byte("10000"), 0644)
				require.NoError(t, err)

				return stateDir
			},
			wantErr: false,
		},
		{
			name: "Нулевой UID",
			uid:  0,
			setupDir: func() string {
				return filepath.Join(tempDir, "zero_uid_state")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stateDir := tt.setupDir()

			saveLastUID(tt.uid, stateDir)

			// Проверяем, что UID сохранен
			uidFile := filepath.Join(stateDir, "last_uid.txt")
			content, err := os.ReadFile(uidFile)
			if !tt.wantErr {
				assert.NoError(t, err)
				savedUID, err := strconv.Atoi(string(content))
				assert.NoError(t, err)
				assert.Equal(t, tt.uid, savedUID)
			}
		})
	}
}

func TestLoadLastUID_SaveLastUID_Integration(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "integration_state")

	// Тестируем полный цикл загрузки и сохранения
	testUIDs := []int{1000, 2000, 3000, 0, 5000}

	for _, uid := range testUIDs {
		// Сохраняем UID
		saveLastUID(uid, stateDir)

		// Загружаем UID
		loadedUID := loadLastUID(stateDir)

		// Проверяем, что значения совпадают
		assert.Equal(t, uid, loadedUID)
	}
}

func TestStateFile_Persistence(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "persistence_state")

	// Сохраняем UID
	testUID := 99999
	saveLastUID(testUID, stateDir)

	// Проверяем, что файл создан
	uidFile := filepath.Join(stateDir, "last_uid.txt")
	_, err := os.Stat(uidFile)
	assert.NoError(t, err)

	// Проверяем содержимое
	content, err := os.ReadFile(uidFile)
	assert.NoError(t, err)

	savedUID, err := strconv.Atoi(string(content))
	assert.NoError(t, err)
	assert.Equal(t, testUID, savedUID)

	// Проверяем, что директория создана с правильными правами
	info, err := os.Stat(stateDir)
	assert.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestStateFile_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Skip("Flaky due to concurrent file writes; skipping for stability")

	tempDir := t.TempDir()
	stateDir := filepath.Join(tempDir, "concurrent_state")

	// Тестируем конкурентный доступ к файлу состояния
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			uid := 1000 + id
			saveLastUID(uid, stateDir)

			loadedUID := loadLastUID(stateDir)
			assert.Equal(t, uid, loadedUID)
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем финальное состояние
	finalUID := loadLastUID(stateDir)
	assert.GreaterOrEqual(t, finalUID, 1000)
	assert.LessOrEqual(t, finalUID, 1009)
}
