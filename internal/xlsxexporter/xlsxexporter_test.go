package xlsxexporter

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"receiptAnalyzer/internal/receipt"
	"receiptAnalyzer/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExportToXLSX_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name     string
		setupDB  func() storage.Storage
		xlsxPath string
		wantErr  bool
	}{
		{
			name: "Валидный экспорт с данными",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_export.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)

				// Добавляем тестовые данные
				rcpt := receipt.Receipt{
					ID:        "test-receipt-1",
					Shop:      "Test Shop",
					DateTime:  time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
					Total:     299.99,
					Source:    "test",
					CreatedAt: time.Now(),
				}
				err = store.SaveReceipt(rcpt)
				require.NoError(t, err)

				items := []receipt.Item{
					{Name: "Товар 1", Quantity: "2", Price: "100.50"},
					{Name: "Товар 2", Quantity: "1", Price: "99.00"},
				}
				err = store.SaveItems(rcpt.ID, items)
				require.NoError(t, err)

				return store
			},
			xlsxPath: filepath.Join(tempDir, "test_export.xlsx"),
			wantErr:  false,
		},
		{
			name: "Экспорт пустой базы",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "empty_export.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			xlsxPath: filepath.Join(tempDir, "empty_export.xlsx"),
			wantErr:  false,
		},
		{
			name: "Недоступная директория",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_export2.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			xlsxPath: "/root/inaccessible/test.xlsx",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupDB()

			err := ExportToXLSX(store, tt.xlsxPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Проверяем, что файл создан
			_, err = os.Stat(tt.xlsxPath)
			assert.NoError(t, err)

			// Проверяем размер файла (должен быть больше 0)
			info, err := os.Stat(tt.xlsxPath)
			assert.NoError(t, err)
			assert.Greater(t, info.Size(), int64(0))
		})
	}
}

func TestGetAllReceipts_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupDB     func() storage.Storage
		expectedLen int
		checkData   func(*testing.T, []receipt.Receipt)
	}{
		{
			name: "База с чеками",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_receipts.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)

				// Добавляем несколько чеков
				receipts := []receipt.Receipt{
					{
						ID:        "receipt-1",
						Shop:      "Shop 1",
						DateTime:  time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
						Total:     100.0,
						Source:    "test",
						CreatedAt: time.Now(),
					},
					{
						ID:        "receipt-2",
						Shop:      "Shop 2",
						DateTime:  time.Date(2023, 12, 26, 16, 45, 0, 0, time.UTC),
						Total:     200.0,
						Source:    "test",
						CreatedAt: time.Now(),
					},
				}

				for _, r := range receipts {
					err = store.SaveReceipt(r)
					require.NoError(t, err)
				}

				return store
			},
			expectedLen: 2,
			checkData: func(t *testing.T, receipts []receipt.Receipt) {
				assert.Len(t, receipts, 2)

				// Проверяем, что чеки отсортированы по дате (новые сначала)
				assert.Equal(t, "receipt-2", receipts[0].ID)
				assert.Equal(t, "receipt-1", receipts[1].ID)

				// Проверяем данные
				for _, r := range receipts {
					assert.NotEmpty(t, r.ID)
					assert.NotEmpty(t, r.Shop)
					assert.NotZero(t, r.Total)
				}
			},
		},
		{
			name: "Пустая база",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "empty_receipts.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			expectedLen: 0,
			checkData: func(t *testing.T, receipts []receipt.Receipt) {
				assert.Empty(t, receipts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupDB()

			receipts, err := getAllReceipts(store)

			assert.NoError(t, err)
			assert.Len(t, receipts, tt.expectedLen)
			tt.checkData(t, receipts)
		})
	}
}

func TestGetItemsForReceipt_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupDB     func() (storage.Storage, string)
		receiptID   string
		expectedLen int
		checkData   func(*testing.T, []receipt.Item)
	}{
		{
			name: "Чек с позициями",
			setupDB: func() (storage.Storage, string) {
				dbPath := filepath.Join(tempDir, "test_items.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)

				receiptID := "receipt-with-items"
				rcpt := receipt.Receipt{
					ID:        receiptID,
					Shop:      "Test Shop",
					DateTime:  time.Now(),
					Total:     300.0,
					Source:    "test",
					CreatedAt: time.Now(),
				}
				err = store.SaveReceipt(rcpt)
				require.NoError(t, err)

				items := []receipt.Item{
					{Name: "Товар 1", Quantity: "2", Price: "100.50"},
					{Name: "Товар 2", Quantity: "1", Price: "99.00"},
				}
				err = store.SaveItems(receiptID, items)
				require.NoError(t, err)

				return store, receiptID
			},
			expectedLen: 2,
			checkData: func(t *testing.T, items []receipt.Item) {
				assert.Len(t, items, 2)

				// Проверяем данные
				assert.Equal(t, "Товар 1", items[0].Name)
				assert.Equal(t, "2", items[0].Quantity)
				assert.Equal(t, "100.50", items[0].Price)

				assert.Equal(t, "Товар 2", items[1].Name)
				assert.Equal(t, "1", items[1].Quantity)
				assert.Equal(t, "99.00", items[1].Price)
			},
		},
		{
			name: "Чек без позиций",
			setupDB: func() (storage.Storage, string) {
				dbPath := filepath.Join(tempDir, "test_no_items.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)

				receiptID := "receipt-no-items"
				rcpt := receipt.Receipt{
					ID:        receiptID,
					Shop:      "Test Shop",
					DateTime:  time.Now(),
					Total:     0.0,
					Source:    "test",
					CreatedAt: time.Now(),
				}
				err = store.SaveReceipt(rcpt)
				require.NoError(t, err)

				return store, receiptID
			},
			expectedLen: 0,
			checkData: func(t *testing.T, items []receipt.Item) {
				assert.Empty(t, items)
			},
		},
		{
			name: "Несуществующий чек",
			setupDB: func() (storage.Storage, string) {
				dbPath := filepath.Join(tempDir, "test_nonexistent.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store, "nonexistent-receipt"
			},
			expectedLen: 0,
			checkData: func(t *testing.T, items []receipt.Item) {
				assert.Empty(t, items)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, receiptID := tt.setupDB()

			items, err := getItemsForReceipt(store, receiptID)

			assert.NoError(t, err)
			assert.Len(t, items, tt.expectedLen)
			tt.checkData(t, items)
		})
	}
}

func TestExportToXLSX_Integration(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "integration_export.db")
	xlsxPath := filepath.Join(tempDir, "integration_export.xlsx")

	// Создаем хранилище
	store, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)

	// Добавляем тестовые данные
	receipts := []receipt.Receipt{
		{
			ID:        "receipt-1",
			Shop:      "Shop A",
			DateTime:  time.Date(2023, 12, 25, 15, 30, 0, 0, time.UTC),
			Total:     150.0,
			Source:    "integration-test",
			CreatedAt: time.Now(),
		},
		{
			ID:        "receipt-2",
			Shop:      "Shop B",
			DateTime:  time.Date(2023, 12, 26, 16, 45, 0, 0, time.UTC),
			Total:     250.0,
			Source:    "integration-test",
			CreatedAt: time.Now(),
		},
	}

	items := map[string][]receipt.Item{
		"receipt-1": {
			{Name: "Хлеб", Quantity: "2", Price: "50.00"},
			{Name: "Молоко", Quantity: "1", Price: "50.00"},
		},
		"receipt-2": {
			{Name: "Сыр", Quantity: "0.5", Price: "200.00"},
			{Name: "Масло", Quantity: "1", Price: "50.00"},
		},
	}

	// Сохраняем данные
	for _, r := range receipts {
		err = store.SaveReceipt(r)
		require.NoError(t, err)

		err = store.SaveItems(r.ID, items[r.ID])
		require.NoError(t, err)
	}

	// Экспортируем в XLSX
	err = ExportToXLSX(store, xlsxPath)
	assert.NoError(t, err)

	// Проверяем, что файл создан
	_, err = os.Stat(xlsxPath)
	assert.NoError(t, err)

	// Проверяем размер файла
	info, err := os.Stat(xlsxPath)
	assert.NoError(t, err)
	assert.Greater(t, info.Size(), int64(1000)) // XLSX файл должен быть достаточно большим
}

func TestExportToXLSX_ErrorHandling(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name     string
		setupDB  func() storage.Storage
		xlsxPath string
		wantErr  bool
	}{
		{
			name: "Недоступная директория",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_error.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			xlsxPath: "/root/inaccessible/test.xlsx",
			wantErr:  true,
		},
		{
			name: "Путь с несуществующими поддиректориями",
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_error2.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			xlsxPath: filepath.Join(tempDir, "nonexistent", "subdir", "test.xlsx"),
			wantErr:  false, // Должен создать директории автоматически
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := tt.setupDB()

			err := ExportToXLSX(store, tt.xlsxPath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)

			// Проверяем, что файл создан
			_, err = os.Stat(tt.xlsxPath)
			assert.NoError(t, err)
		})
	}
}
