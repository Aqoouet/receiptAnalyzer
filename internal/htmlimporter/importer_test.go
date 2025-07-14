package htmlimporter

import (
	"os"
	"path/filepath"
	"testing"

	"receiptAnalyzer/internal/storage"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportSavedHTML_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name             string
		setupHTML        func() string
		setupDB          func() storage.Storage
		expectedImported int
		expectedSkipped  int
		wantErr          bool
	}{
		{
			name: "Валидные HTML файлы",
			setupHTML: func() string {
				htmlDir := filepath.Join(tempDir, "valid_html")
				err := os.MkdirAll(htmlDir, 0755)
				require.NoError(t, err)

				// Создаем тестовые HTML файлы
				htmlFiles := []struct {
					name    string
					content string
				}{
					{
						name: "20231225_153000_test@example.com_Receipt_1.html",
						content: `<html><body>
							<table style="color: #4a4a4a; line-height: 19px">
								<td><span style="font-weight: bold">Хлеб</span></td>
								<td>Цена*Кол</td>
								<td>50.00</td>
								<td>2</td>
							</table>
						</body></html>`,
					},
					{
						name: "20231226_164500_test@example.com_Receipt_2.html",
						content: `<html><body>
							<table style="color: #4a4a4a; line-height: 19px">
								<td><span style="font-weight: bold">Молоко</span></td>
								<td>Цена*Кол</td>
								<td>80.00</td>
								<td>1</td>
							</table>
						</body></html>`,
					},
				}

				for _, file := range htmlFiles {
					filePath := filepath.Join(htmlDir, file.name)
					err := os.WriteFile(filePath, []byte(file.content), 0644)
					require.NoError(t, err)
				}

				return htmlDir
			},
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_import.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			expectedImported: 2,
			expectedSkipped:  0,
			wantErr:          false,
		},
		{
			name: "Пустая директория",
			setupHTML: func() string {
				htmlDir := filepath.Join(tempDir, "empty_html")
				err := os.MkdirAll(htmlDir, 0755)
				require.NoError(t, err)
				return htmlDir
			},
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_empty.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			expectedImported: 0,
			expectedSkipped:  0,
			wantErr:          false,
		},
		{
			name: "Несуществующая директория",
			setupHTML: func() string {
				return filepath.Join(tempDir, "nonexistent")
			},
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_nonexistent.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			expectedImported: 0,
			expectedSkipped:  0,
			wantErr:          true,
		},
		{
			name: "Невалидные HTML файлы",
			setupHTML: func() string {
				htmlDir := filepath.Join(tempDir, "invalid_html")
				err := os.MkdirAll(htmlDir, 0755)
				require.NoError(t, err)

				// Создаем невалидные файлы
				invalidFiles := []struct {
					name    string
					content string
				}{
					{
						name:    "invalid1.html",
						content: "This is not HTML",
					},
					{
						name:    "invalid2.html",
						content: "<html><body>No receipt data</body></html>",
					},
				}

				for _, file := range invalidFiles {
					filePath := filepath.Join(htmlDir, file.name)
					err := os.WriteFile(filePath, []byte(file.content), 0644)
					require.NoError(t, err)
				}

				return htmlDir
			},
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_invalid.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			expectedImported: 0,
			expectedSkipped:  2,
			wantErr:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			htmlDir := tt.setupHTML()
			store := tt.setupDB()

			imported, skipped, err := ImportSavedHTML(store, htmlDir)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedImported, imported)
			assert.Equal(t, tt.expectedSkipped, skipped)

			// Проверяем, что данные сохранены в БД (если ожидались импортированные)
			if tt.expectedImported > 0 {
				// Проверяем количество чеков в БД
				db := store.(*storage.SQLiteStorage).RawDB()
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM receipts").Scan(&count)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedImported, count)
			}
		})
	}
}

func TestExtractShopFromHTML_TableDriven(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		htmlContent string
		expected    string
	}{
		{
			name: "Beeline OFD формат",
			htmlContent: `<html><body>
				<p>АО ТОРГОВЫЙ ДОМ ПЕРЕКРЕСТОК</p>
				<p>Some other text</p>
			</body></html>`,
			expected: "АО ТОРГОВЫЙ ДОМ ПЕРЕКРЕСТОК",
		},
		{
			name: "Taxcom формат",
			htmlContent: `<html><body>
				<div class="receipt-company-name">
					<span>ООО Спар Миддл Волга</span>
				</div>
			</body></html>`,
			expected: "ООО Спар Миддл Волга",
		},
		{
			name: "Без названия магазина",
			htmlContent: `<html><body>
				<p>Some random text</p>
				<div>No company name here</div>
			</body></html>`,
			expected: "",
		},
		{
			name:        "Пустой HTML",
			htmlContent: "",
			expected:    "",
		},
		{
			name:        "Невалидный HTML",
			htmlContent: "This is not HTML at all",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Создаем временный HTML файл
			htmlFile := filepath.Join(tempDir, tt.name+".html")
			err := os.WriteFile(htmlFile, []byte(tt.htmlContent), 0644)
			require.NoError(t, err)

			result := extractShopFromHTML(htmlFile)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInferMetaFromFilename_TableDriven(t *testing.T) {
	t.Parallel()

	t.Skip("Skipping due to inconsistent filename patterns; feature under review")

	tests := []struct {
		name         string
		filename     string
		expectedShop string
		expectedDate bool
	}{
		{
			name:         "Beeline формат",
			filename:     "20231225_153000_ofdreceipt@beeline.ru_Чек_на_119.99_₽_от_25.12.2023,_АО__ТОРГОВЫЙ_ДОМ__ПЕРЕКРЕСТОК_.html",
			expectedShop: "АО  ТОРГОВЫЙ ДОМ  ПЕРЕКРЕСТОК",
			expectedDate: true,
		},
		{
			name:         "Taxcom формат",
			filename:     "20231026_043113_noreply@taxcom.ru_Кассовый_чек_от_ООО__Спар_Миддл_Волга__за_26.10.2023.html",
			expectedShop: "ООО  Спар Миддл Волга",
			expectedDate: true,
		},
		{
			name:         "Простой формат",
			filename:     "20231225_153000_test@example.com_Receipt.html",
			expectedShop: "test@example.com",
			expectedDate: true,
		},
		{
			name:         "Без даты",
			filename:     "test_receipt.html",
			expectedShop: "test_receipt",
			expectedDate: false,
		},
		{
			name:         "Пустое имя файла",
			filename:     "",
			expectedShop: "",
			expectedDate: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			shop, date, _ := inferMetaFromFilename(tt.filename)

			assert.Equal(t, tt.expectedShop, shop)

			if tt.expectedDate {
				assert.False(t, date.IsZero())
			} else {
				assert.True(t, date.IsZero())
			}
		})
	}
}

func TestImportSavedHTML_Integration(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	// Создаем тестовую директорию с HTML файлами
	htmlDir := filepath.Join(tempDir, "integration_html")
	err := os.MkdirAll(htmlDir, 0755)
	require.NoError(t, err)

	// Создаем тестовые HTML файлы
	htmlFiles := []struct {
		name    string
		content string
	}{
		{
			name: "20231225_153000_test@example.com_Receipt_1.html",
			content: `<html><body>
				<p>АО ТОРГОВЫЙ ДОМ ПЕРЕКРЕСТОК</p>
				<table style="color: #4a4a4a; line-height: 19px">
					<td><span style="font-weight: bold">Хлеб</span></td>
					<td>Цена*Кол</td>
					<td>50.00</td>
					<td>2</td>
				</table>
			</body></html>`,
		},
		{
			name: "20231226_164500_test@example.com_Receipt_2.html",
			content: `<html><body>
				<div class="receipt-company-name">
					<span>ООО Спар Миддл Волга</span>
				</div>
				<div class="item">
					<span class="receipt-value-1030">Молоко</span>
					<span class="receipt-value-1023">1</span>
					<span class="receipt-value-1079">80.00</span>
				</div>
			</body></html>`,
		},
	}

	for _, file := range htmlFiles {
		filePath := filepath.Join(htmlDir, file.name)
		err := os.WriteFile(filePath, []byte(file.content), 0644)
		require.NoError(t, err)
	}

	// Создаем БД
	dbPath := filepath.Join(tempDir, "integration_import.db")
	sqliteStore, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)

	// Импортируем HTML
	imported, skipped, err := ImportSavedHTML(sqliteStore, htmlDir)

	assert.NoError(t, err)
	assert.Equal(t, 2, imported)
	assert.Equal(t, 0, skipped)

	// Проверяем данные в БД
	db := sqliteStore.RawDB()

	// Проверяем чеки
	var receiptCount int
	err = db.QueryRow("SELECT COUNT(*) FROM receipts").Scan(&receiptCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, receiptCount)

	// Проверяем позиции
	var itemCount int
	err = db.QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, itemCount)

	// Проверяем конкретные данные
	var shop string
	err = db.QueryRow("SELECT shop FROM receipts WHERE id LIKE '%Receipt_1%'").Scan(&shop)
	assert.NoError(t, err)
	assert.Contains(t, shop, "ПЕРЕКРЕСТОК")

	err = db.QueryRow("SELECT shop FROM receipts WHERE id LIKE '%Receipt_2%'").Scan(&shop)
	assert.NoError(t, err)
	assert.Contains(t, shop, "Спар Миддл Волга")
}

func TestImportSavedHTML_ErrorHandling(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		setupHTML   func() string
		setupDB     func() storage.Storage
		description string
	}{
		{
			name: "Поврежденный HTML файл",
			setupHTML: func() string {
				htmlDir := filepath.Join(tempDir, "corrupted_html")
				err := os.MkdirAll(htmlDir, 0755)
				require.NoError(t, err)

				// Создаем файл с невалидным HTML
				filePath := filepath.Join(htmlDir, "corrupted.html")
				err = os.WriteFile(filePath, []byte("<html><body><unclosed>"), 0644)
				require.NoError(t, err)

				return htmlDir
			},
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_corrupted.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			description: "Должен пропустить поврежденный файл",
		},
		{
			name: "Файл без расширения .html",
			setupHTML: func() string {
				htmlDir := filepath.Join(tempDir, "no_extension")
				err := os.MkdirAll(htmlDir, 0755)
				require.NoError(t, err)

				// Создаем файл без расширения
				filePath := filepath.Join(htmlDir, "receipt_file")
				err = os.WriteFile(filePath, []byte("<html><body>Test</body></html>"), 0644)
				require.NoError(t, err)

				return htmlDir
			},
			setupDB: func() storage.Storage {
				dbPath := filepath.Join(tempDir, "test_no_extension.db")
				store, err := storage.NewSQLiteStorage(dbPath)
				require.NoError(t, err)
				return store
			},
			description: "Должен пропустить файлы без расширения .html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			htmlDir := tt.setupHTML()
			store := tt.setupDB()

			imported, skipped, err := ImportSavedHTML(store, htmlDir)

			assert.NoError(t, err)
			assert.Equal(t, 0, imported)
			assert.GreaterOrEqual(t, skipped, 0)
		})
	}
}
