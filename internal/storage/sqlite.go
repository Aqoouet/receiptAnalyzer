package storage

import (
	"database/sql"
	"log"
	"receiptAnalyzer/internal/receipt"

	_ "modernc.org/sqlite"
)

type Storage interface {
	SaveReceipt(r receipt.Receipt) error
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
