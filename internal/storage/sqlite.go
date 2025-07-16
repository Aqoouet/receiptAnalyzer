package storage

import (
	"database/sql"
	"fmt"
	"log"
	"receiptAnalyzer/internal/receipt"

	_ "modernc.org/sqlite"
)

type Storage interface {
	SaveReceipt(r receipt.Receipt) error
	SaveItems(receiptID string, items []receipt.Item) error
	// Можно добавить другие методы: GetReceipt, GetAll и т.д.
}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	log.Printf("Открываем файл базы данных SQLite: %s", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// Включаем WAL для уменьшения блокировок
	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	// Восстановить CREATE TABLE receipts без поля total
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
	if err != nil {
		return nil, err
	}

	// create items table
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
	if err != nil {
		return nil, err
	}

	log.Println("SQLite-хранилище инициализировано")
	return &SQLiteStorage{db: db}, nil
}

// SaveReceipt сохраняет чек (поле total не вычисляется автоматически)
func (s *SQLiteStorage) SaveReceipt(r receipt.Receipt) error {
	log.Printf("Сохраняем чек в базу: %+v", r)
	_, err := s.db.Exec(`
        INSERT INTO receipts (folder, receipt_id, hash, sender, date_time, subject, is_receipt, link, template, total, delta_sum)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`+
		"\nON CONFLICT(hash) DO NOTHING",
		r.Folder, r.ReceiptID, r.Hash, r.Sender, r.DateTime, r.Subject, r.IsReceipt, r.Link, r.Template, r.Total, r.DeltaSum,
	)
	return err
}

// SaveItems сохраняет items и обновляет receipts.total
func (s *SQLiteStorage) SaveItems(hash string, items []receipt.Item) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO items (hash, name, quantity, unit_price, total, category, sub_quantity, name_cleaned) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	totalSum := 0.0
	for _, it := range items {
		priceVal, _ := receipt.ParseFloat(it.Price)
		qtyVal, _ := receipt.ParseFloat(it.Quantity)
		total := priceVal * qtyVal
		totalSum += total
		if _, err := stmt.Exec(hash, it.Name, qtyVal, priceVal, total, it.Category, it.SubQuantity, it.NameCleaned); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	// Получаем ранее заявленную сумму, записанную в receipts.total
	var declared float64
	if err := tx.QueryRow("SELECT total FROM receipts WHERE hash = ?", hash).Scan(&declared); err != nil {
		if err != sql.ErrNoRows {
			_ = tx.Rollback()
			return err
		}
	}

	// Вычисляем дельту как разницу между фактической суммой и заявленной
	delta := totalSum - declared

	// Обновляем только delta_sum
	_, _ = tx.Exec("UPDATE receipts SET delta_sum = ? WHERE hash = ?", delta, hash)

	return tx.Commit()
}

// UpdateReceiptsTotal обновляет receipts.total как сумму по items для каждого чека
func UpdateReceiptsTotal(db *sql.DB) error {
	// Обновить receipts.total и delta_sum как сумму по items / разницу
	_, err := db.Exec(`UPDATE receipts SET total = (
		SELECT IFNULL(SUM(total),0) FROM items WHERE items.hash = receipts.hash
	),
	delta_sum = (
		SELECT IFNULL(SUM(total),0) FROM items WHERE items.hash = receipts.hash
	) - IFNULL(total,0)`)
	return err
}

// RawDB returns underlying *sql.DB for advanced queries not covered by interface.
func (s *SQLiteStorage) RawDB() *sql.DB {
	return s.db
}

// Close закрывает соединение с базой данных
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// GetReceiptsByDeltaDesc возвращает чеки, отсортированные по абсолютному значению delta_sum по убыванию
func (s *SQLiteStorage) GetReceiptsByDeltaDesc(limit int) ([]receipt.Receipt, error) {
	rows, err := s.db.Query(`
		SELECT folder, receipt_id, hash, sender, date_time, subject, is_receipt, link, template, total, delta_sum 
		FROM receipts 
		ORDER BY ABS(delta_sum) DESC 
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []receipt.Receipt
	for rows.Next() {
		var r receipt.Receipt
		err := rows.Scan(&r.Folder, &r.ReceiptID, &r.Hash, &r.Sender, &r.DateTime, &r.Subject, &r.IsReceipt, &r.Link, &r.Template, &r.Total, &r.DeltaSum)
		if err != nil {
			return nil, err
		}
		receipts = append(receipts, r)
	}
	return receipts, nil
}

// GetItems возвращает все товары для указанного чека
func (s *SQLiteStorage) GetItems(hash string) ([]receipt.Item, error) {
	rows, err := s.db.Query(`
		SELECT name, quantity, unit_price, total, category, sub_quantity, name_cleaned 
		FROM items 
		WHERE hash = ?`, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []receipt.Item
	for rows.Next() {
		var item receipt.Item
		var qty, unitPrice, total float64
		err := rows.Scan(&item.Name, &qty, &unitPrice, &total, &item.Category, &item.SubQuantity, &item.NameCleaned)
		if err != nil {
			return nil, err
		}
		item.Hash = hash
		item.Quantity = fmt.Sprintf("%.2f", qty)
		item.Price = fmt.Sprintf("%.2f", unitPrice)
		items = append(items, item)
	}
	return items, nil
}
