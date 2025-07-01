package storage

import (
	"checkAnalyzer/internal/receipt"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

type Storage interface {
	SaveReceipt(r receipt.Receipt) error
	// Можно добавить другие методы: GetReceipt, GetAll и т.д.
}

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(path string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", path)
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

	return &SQLiteStorage{db: db}, nil
}

func (s *SQLiteStorage) SaveReceipt(r receipt.Receipt) error {
	_, err := s.db.Exec(`
        INSERT INTO receipts (id, shop, date_time, total, source, created_at)
        VALUES (?, ?, ?, ?, ?, ?)`,
		r.ID, r.Shop, r.DateTime, r.Total, r.Source, r.CreatedAt,
	)
	return err
}
