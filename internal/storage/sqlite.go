package storage

import (
	"database/sql"
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

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS receipts (
            id TEXT PRIMARY KEY,
            shop TEXT,
            date_time DATETIME,
            total REAL,
            source TEXT,
            created_at DATETIME
        )
    `)
	if err != nil {
		return nil, err
	}

	// create items table
	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS items (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            receipt_id TEXT,
            name TEXT,
            quantity REAL,
            unit_price REAL,
            total REAL,
            FOREIGN KEY(receipt_id) REFERENCES receipts(id)
        )
    `)
	if err != nil {
		return nil, err
	}

	log.Println("SQLite-хранилище инициализировано")
	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) SaveReceipt(r receipt.Receipt) error {
	log.Printf("Сохраняем чек в базу: %+v", r)
	_, err := s.db.Exec(`
        INSERT INTO receipts (id, shop, date_time, total, source, created_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Shop, r.DateTime, r.Total, r.Source, r.CreatedAt,
	)
	if err != nil {
		log.Printf("Ошибка при сохранении чека: %v", err)
	}
	return err
}

// SaveItems stores list of items related to receipt.
func (s *SQLiteStorage) SaveItems(receiptID string, items []receipt.Item) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO items (receipt_id, name, quantity, unit_price, total) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, it := range items {
		priceVal, _ := receipt.ParseFloat(it.Price)
		qtyVal, _ := receipt.ParseFloat(it.Quantity)
		total := priceVal * qtyVal
		if _, err := stmt.Exec(receiptID, it.Name, qtyVal, priceVal, total); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// RawDB returns underlying *sql.DB for advanced queries not covered by interface.
func (s *SQLiteStorage) RawDB() *sql.DB {
	return s.db
}
