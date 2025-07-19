package manualcorrection

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func TestApplyManualCorrections_Fields(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем временную БД
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Создаем таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS receipts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			folder TEXT,
			receipt_id TEXT,
			hash TEXT UNIQUE,
			sender TEXT,
			date_time DATETIME,
			subject TEXT,
			is_receipt BOOLEAN,
			link TEXT,
			template TEXT,
			total REAL DEFAULT 0,
			delta_sum REAL DEFAULT 0
		)
	`)
	require.NoError(t, err)

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT,
			name TEXT,
			quantity REAL,
			unit_price REAL,
			total REAL,
			category TEXT,
			sub_quantity TEXT,
			name_cleaned TEXT,
			FOREIGN KEY(hash) REFERENCES receipts(hash)
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовый чек
	testHash := "test_hash_123"
	_, err = db.Exec(`
		INSERT INTO receipts (hash, sender, subject, date_time, is_receipt, total)
		VALUES (?, ?, ?, ?, ?, ?)
	`, testHash, "old@example.com", "Old Subject", time.Now(), true, 100.0)
	require.NoError(t, err)

	// Создаем файл с исправлениями
	correctionsDir := filepath.Join(tempDir, "corrections")
	err = os.MkdirAll(correctionsDir, 0755)
	require.NoError(t, err)

	correctionsFile := filepath.Join(correctionsDir, "test_corrections.json")
	correctionsJSON := `[
		{
			"hash": "test_hash_123",
			"fields": [
				{
					"field": "sender",
					"value": "new@example.com"
				},
				{
					"field": "subject",
					"value": "New Subject"
				},
				{
					"field": "total",
					"value": 200.0
				}
			]
		}
	]`

	err = os.WriteFile(correctionsFile, []byte(correctionsJSON), 0644)
	require.NoError(t, err)

	// Применяем исправления
	receiptsUpdated, itemsUpdated, err := ApplyManualCorrections(db, correctionsDir)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, 1, receiptsUpdated)
	assert.Equal(t, 0, itemsUpdated)

	// Проверяем, что поля обновились
	var sender, subject string
	var total float64

	err = db.QueryRow("SELECT sender, subject, total FROM receipts WHERE hash = ?", testHash).
		Scan(&sender, &subject, &total)
	require.NoError(t, err)

	assert.Equal(t, "new@example.com", sender)
	assert.Equal(t, "New Subject", subject)
	assert.Equal(t, 200.0, total)
}

func TestApplyManualCorrections_InvalidField(t *testing.T) {
	// Создаем временную директорию для тестов
	tempDir := t.TempDir()

	// Создаем временную БД
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	// Создаем таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS receipts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT UNIQUE,
			sender TEXT
		)
	`)
	require.NoError(t, err)

	// Вставляем тестовый чек
	testHash := "test_hash_123"
	_, err = db.Exec("INSERT INTO receipts (hash, sender) VALUES (?, ?)", testHash, "old@example.com")
	require.NoError(t, err)

	// Создаем файл с исправлениями
	correctionsDir := filepath.Join(tempDir, "corrections")
	err = os.MkdirAll(correctionsDir, 0755)
	require.NoError(t, err)

	correctionsFile := filepath.Join(correctionsDir, "test_corrections.json")
	correctionsJSON := `[
		{
			"hash": "test_hash_123",
			"fields": [
				{
					"field": "invalid_field",
					"value": "should_be_ignored"
				},
				{
					"field": "sender",
					"value": "new@example.com"
				}
			]
		}
	]`

	err = os.WriteFile(correctionsFile, []byte(correctionsJSON), 0644)
	require.NoError(t, err)

	// Применяем исправления
	receiptsUpdated, itemsUpdated, err := ApplyManualCorrections(db, correctionsDir)

	// Проверяем результаты
	assert.NoError(t, err)
	assert.Equal(t, 1, receiptsUpdated)
	assert.Equal(t, 0, itemsUpdated)

	// Проверяем, что только валидное поле обновилось
	var sender string
	err = db.QueryRow("SELECT sender FROM receipts WHERE hash = ?", testHash).Scan(&sender)
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", sender)
}
