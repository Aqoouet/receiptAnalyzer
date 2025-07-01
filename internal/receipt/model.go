package receipt

import "time"

type Receipt struct {
	ID        string    `db:"id"`
	Shop      string    `db:"shop"`
	DateTime  time.Time `db:"date_time"`
	Total     float64   `db:"total"`
	Source    string    `db:"source"`
	CreatedAt time.Time `db:"created_at"`
}
