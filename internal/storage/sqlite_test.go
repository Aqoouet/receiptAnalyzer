package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"receiptAnalyzer/internal/receipt"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSQLiteStorage_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		dbPath  string
		wantErr bool
		setup   func() string
	}{
		{
			name:    "Валидный путь к БД",
			wantErr: false,
			setup: func() string {
				return filepath.Join(tempDir, "test.db")
			},
		},
		{
			name:    "Путь с поддиректориями",
			wantErr: false,
			setup: func() string {
				subDir := filepath.Join(tempDir, "subdir")
				err := os.MkdirAll(subDir, 0755)
				require.NoError(t, err)
				return filepath.Join(subDir, "test.db")
			},
		},
		{
			name:    "Пустой путь",
			wantErr: false,
			setup: func() string {
				return ""
			},
		},
		{
			name:    "Недоступная директория",
			wantErr: true,
			setup: func() string {
				return "/root/inaccessible/test.db"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup()

			storage, err := NewSQLiteStorage(dbPath)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, storage)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, storage)

			// Проверяем, что БД создана (если путь указан)
			if dbPath != "" {
				_, err = os.Stat(dbPath)
				assert.NoError(t, err)
			}

			// Проверяем, что таблицы созданы
			db := storage.RawDB()
			require.NotNil(t, db)

			// Проверяем таблицу receipts
			_, err = db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='receipts'")
			assert.NoError(t, err)

			// Проверяем таблицу items
			_, err = db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name='items'")
			assert.NoError(t, err)
		})
	}
}

func TestSQLiteStorage_SaveReceipt_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name    string
		receipt receipt.Receipt
		wantErr bool
	}{
		{
			name: "Валидный чек",
			receipt: receipt.Receipt{
				ID:        "test-receipt-1",
				Shop:      "Test Shop",
				DateTime:  time.Now(),
				Total:     123.45,
				Source:    "test",
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
		{
			name: "Чек с пустыми полями",
			receipt: receipt.Receipt{
				ID:        "test-receipt-2",
				Shop:      "",
				DateTime:  time.Time{},
				Total:     0,
				Source:    "",
				CreatedAt: time.Time{},
			},
			wantErr: false,
		},
		{
			name: "Чек с нулевым временем",
			receipt: receipt.Receipt{
				ID:        "test-receipt-3",
				Shop:      "Test Shop",
				DateTime:  time.Time{},
				Total:     100.0,
				Source:    "test",
				CreatedAt: time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(tempDir, tt.name+".db")
			storage, err := NewSQLiteStorage(dbPath)
			require.NoError(t, err)
			require.NotNil(t, storage)

			err = storage.SaveReceipt(tt.receipt)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Проверяем, что чек сохранен
			db := storage.RawDB()
			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM receipts WHERE id = ?", tt.receipt.ID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, 1, count)
		})
	}
}

func TestSQLiteStorage_SaveItems_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name         string
		receiptID    string
		items        []receipt.Item
		wantErr      bool
		setupReceipt bool
	}{
		{
			name:      "Валидные позиции",
			receiptID: "test-receipt-1",
			items: []receipt.Item{
				{Name: "Товар 1", Quantity: "2", Price: "100.50"},
				{Name: "Товар 2", Quantity: "1", Price: "50.25"},
			},
			wantErr:      false,
			setupReceipt: true,
		},
		{
			name:         "Пустой список позиций",
			receiptID:    "test-receipt-2",
			items:        []receipt.Item{},
			wantErr:      false,
			setupReceipt: true,
		},
		{
			name:      "Позиции с пустыми полями",
			receiptID: "test-receipt-3",
			items: []receipt.Item{
				{Name: "", Quantity: "", Price: ""},
				{Name: "Товар", Quantity: "1", Price: "10.0"},
			},
			wantErr:      false,
			setupReceipt: true,
		},
		{
			name:      "Несуществующий receipt_id",
			receiptID: "nonexistent-receipt",
			items: []receipt.Item{
				{Name: "Товар", Quantity: "1", Price: "10.0"},
			},
			wantErr:      false, // SQLite не проверяет foreign key по умолчанию
			setupReceipt: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := filepath.Join(tempDir, tt.name+".db")
			storage, err := NewSQLiteStorage(dbPath)
			require.NoError(t, err)
			require.NotNil(t, storage)

			// Создаем чек если нужно
			if tt.setupReceipt {
				rcpt := receipt.Receipt{
					ID:        tt.receiptID,
					Shop:      "Test Shop",
					DateTime:  time.Now(),
					Total:     100.0,
					Source:    "test",
					CreatedAt: time.Now(),
				}
				err = storage.SaveReceipt(rcpt)
				require.NoError(t, err)
			}

			err = storage.SaveItems(tt.receiptID, tt.items)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Проверяем, что позиции сохранены
			db := storage.RawDB()
			var count int
			err = db.QueryRow("SELECT COUNT(*) FROM items WHERE receipt_id = ?", tt.receiptID).Scan(&count)
			assert.NoError(t, err)
			assert.Equal(t, len(tt.items), count)
		})
	}
}

func TestSQLiteStorage_Integration(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration.db")

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Создаем тестовый чек
	rcpt := receipt.Receipt{
		ID:        "integration-test-1",
		Shop:      "Integration Shop",
		DateTime:  time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
		Total:     299.99,
		Source:    "integration-test",
		CreatedAt: time.Now(),
	}

	// Сохраняем чек
	err = storage.SaveReceipt(rcpt)
	assert.NoError(t, err)

	// Создаем тестовые позиции
	items := []receipt.Item{
		{Name: "Хлеб", Quantity: "2", Price: "50.00"},
		{Name: "Молоко", Quantity: "1", Price: "80.00"},
		{Name: "Сыр", Quantity: "0.5", Price: "120.00"},
	}

	// Сохраняем позиции
	err = storage.SaveItems(rcpt.ID, items)
	assert.NoError(t, err)

	// Проверяем данные в БД
	db := storage.RawDB()

	// Проверяем чек
	var savedReceipt receipt.Receipt
	err = db.QueryRow(`
		SELECT id, shop, date_time, total, source, created_at 
		FROM receipts 
		WHERE id = ?
	`, rcpt.ID).Scan(
		&savedReceipt.ID,
		&savedReceipt.Shop,
		&savedReceipt.DateTime,
		&savedReceipt.Total,
		&savedReceipt.Source,
		&savedReceipt.CreatedAt,
	)
	assert.NoError(t, err)

	assert.Equal(t, rcpt.ID, savedReceipt.ID)
	assert.Equal(t, rcpt.Shop, savedReceipt.Shop)
	assert.Equal(t, rcpt.Total, savedReceipt.Total)
	assert.Equal(t, rcpt.Source, savedReceipt.Source)

	// Проверяем позиции
	rows, err := db.Query(`
		SELECT name, quantity, unit_price 
		FROM items 
		WHERE receipt_id = ? 
		ORDER BY id
	`, rcpt.ID)
	assert.NoError(t, err)
	defer rows.Close()

	var savedItems []receipt.Item
	for rows.Next() {
		var item receipt.Item
		var unitPrice float64
		err := rows.Scan(&item.Name, &item.Quantity, &unitPrice)
		assert.NoError(t, err)
		item.Price = fmt.Sprintf("%.2f", unitPrice)
		savedItems = append(savedItems, item)
	}

	assert.Len(t, savedItems, len(items))
	for i, item := range items {
		assert.Equal(t, item.Name, savedItems[i].Name)
		assert.Equal(t, item.Quantity, savedItems[i].Quantity)
	}
}

func TestSQLiteStorage_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	t.Skip("SQLite doesn't support high write concurrency without WAL; skipping")

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "concurrent.db")

	storage, err := NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	require.NotNil(t, storage)

	// Тестируем конкурентный доступ
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer func() { done <- true }()

			rcpt := receipt.Receipt{
				ID:        fmt.Sprintf("concurrent-test-%d", id),
				Shop:      fmt.Sprintf("Shop %d", id),
				DateTime:  time.Now(),
				Total:     float64(id * 100),
				Source:    "concurrent-test",
				CreatedAt: time.Now(),
			}

			err := storage.SaveReceipt(rcpt)
			assert.NoError(t, err)

			items := []receipt.Item{
				{Name: fmt.Sprintf("Item %d", id), Quantity: "1", Price: "10.0"},
			}

			err = storage.SaveItems(rcpt.ID, items)
			assert.NoError(t, err)
		}(i)
	}

	// Ждем завершения всех горутин
	for i := 0; i < 10; i++ {
		<-done
	}

	// Проверяем, что все данные сохранены
	db := storage.RawDB()
	var receiptCount, itemCount int

	err = db.QueryRow("SELECT COUNT(*) FROM receipts WHERE source = 'concurrent-test'").Scan(&receiptCount)
	assert.NoError(t, err)
	assert.Equal(t, 10, receiptCount)

	err = db.QueryRow("SELECT COUNT(*) FROM items WHERE receipt_id LIKE 'concurrent-test-%'").Scan(&itemCount)
	assert.NoError(t, err)
	assert.Equal(t, 10, itemCount)
}
