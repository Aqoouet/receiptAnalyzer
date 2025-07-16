package receipt

import "time"

type Receipt struct {
	Folder    string    `db:"folder"`
	ReceiptID string    `db:"receipt_id"`
	Hash      string    `db:"hash"`
	Sender    string    `db:"sender"`
	DateTime  time.Time `db:"date_time"`
	Subject   string    `db:"subject"`
	IsReceipt bool      `db:"is_receipt"`
	Link      string    `db:"link"`
	Template  string    `db:"template"`
	Total     float64   `db:"total"`     // заявленная сумма чека, если есть
	DeltaSum  float64   `db:"delta_sum"` // разница между фактической и заявленной суммой
}
