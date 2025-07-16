package xlsxexporter

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"receiptAnalyzer/internal/config"
	"receiptAnalyzer/internal/receipt"
	"receiptAnalyzer/internal/storage"

	"github.com/xuri/excelize/v2"
)

// ExportAll экспортирует данные в XLSX (заглушка)
func ExportAll(cfg *config.Config) error {
	store, err := storage.NewSQLiteStorage(cfg.Paths.DBPath)
	if err != nil {
		return err
	}
	return ExportToXLSX(store, cfg.Paths.XLSXPath)
}

// ExportToXLSX экспортирует все чеки и их позиции в XLSX файл
func ExportToXLSX(store storage.Storage, xlsxPath string) error {
	log.Printf("Экспортируем данные в XLSX файл: %s", xlsxPath)

	// Создаем директорию если её нет
	dir := filepath.Dir(xlsxPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("не удалось создать директорию %s: %w", dir, err)
	}

	// Создаем новый Excel файл
	f := excelize.NewFile()
	defer f.Close()

	// Получаем все чеки из базы
	receipts, err := getAllReceipts(store)
	if err != nil {
		return fmt.Errorf("ошибка получения чеков: %w", err)
	}

	// Создаем лист "Чеки"
	sheetName := "Чеки"
	f.SetSheetName("Sheet1", sheetName)

	// Заголовки для листа чеков
	headers := []string{"Hash", "Sender", "Subject", "Date", "IsReceipt"}
	for i, header := range headers {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(sheetName, cell, header)
	}

	// Заполняем данные чеков
	for i, r := range receipts {
		row := i + 2 // начинаем со второй строки (после заголовков)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), r.Hash)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), r.Sender)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), r.Subject)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), r.DateTime.Format("02.01.2006 15:04"))
		f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), r.IsReceipt)
	}

	// Создаем лист "Позиции"
	itemsSheetName := "Позиции"
	f.NewSheet(itemsSheetName)

	// Заголовки для листа позиций
	itemHeaders := []string{"ID чека", "Название", "Количество", "Цена за ед.", "Сумма"}
	for i, header := range itemHeaders {
		cell := fmt.Sprintf("%c1", 'A'+i)
		f.SetCellValue(itemsSheetName, cell, header)
	}

	// Заполняем данные позиций
	row := 2
	for _, r := range receipts {
		items, err := getItemsForReceipt(store, r.Hash)
		if err != nil {
			log.Printf("Ошибка получения позиций для чека %s: %v", r.Hash, err)
			continue
		}

		for _, item := range items {
			f.SetCellValue(itemsSheetName, fmt.Sprintf("A%d", row), r.Hash)
			f.SetCellValue(itemsSheetName, fmt.Sprintf("B%d", row), item.Name)
			f.SetCellValue(itemsSheetName, fmt.Sprintf("C%d", row), item.Quantity)
			f.SetCellValue(itemsSheetName, fmt.Sprintf("D%d", row), item.Price)

			// Вычисляем сумму
			price, _ := receipt.ParseFloat(item.Price)
			qty, _ := receipt.ParseFloat(item.Quantity)
			total := price * qty
			f.SetCellValue(itemsSheetName, fmt.Sprintf("E%d", row), total)

			row++
		}
	}

	// Сохраняем файл
	if err := f.SaveAs(xlsxPath); err != nil {
		return fmt.Errorf("ошибка сохранения XLSX файла: %w", err)
	}

	log.Printf("XLSX файл успешно создан: %s", xlsxPath)
	return nil
}

// getAllReceipts получает все чеки из базы данных
func getAllReceipts(store storage.Storage) ([]receipt.Receipt, error) {
	sqlDB := store.(*storage.SQLiteStorage).RawDB()

	rows, err := sqlDB.Query(`
		SELECT hash, sender, subject, date_time, is_receipt
		FROM receipts
		ORDER BY date_time DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []receipt.Receipt
	for rows.Next() {
		var r receipt.Receipt
		err := rows.Scan(&r.Hash, &r.Sender, &r.Subject, &r.DateTime, &r.IsReceipt)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, r)
	}

	return receipts, nil
}

// getItemsForReceipt получает все позиции для конкретного чека
func getItemsForReceipt(store storage.Storage, receiptHash string) ([]receipt.Item, error) {
	sqlDB := store.(*storage.SQLiteStorage).RawDB()

	rows, err := sqlDB.Query(`
		SELECT name, quantity, unit_price 
		FROM items 
		WHERE receipt_id = ? 
		ORDER BY id
	`, receiptHash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []receipt.Item
	for rows.Next() {
		var item receipt.Item
		var unitPrice float64
		err := rows.Scan(&item.Name, &item.Quantity, &unitPrice)
		if err != nil {
			return nil, err
		}
		item.Price = fmt.Sprintf("%.2f", unitPrice)
		items = append(items, item)
	}

	return items, nil
}
