package receipt

// ExpectedItem represents a single product line in the golden JSON expectations
// used by parser tests (see expected_receipts.json).
//
// The structure mirrors exactly the JSON file: numeric values are stored as
// float64 because the JSON literals are numbers, not strings.
type ExpectedItem struct {
	Name      string  `json:"name"`
	Quantity  float64 `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
	Total     float64 `json:"total"`
}

// ExpectedReceipt represents one whole receipt entry in the expectations map.
// All meta-fields (hash, sender, etc.) are preserved for future extensions,
// but the tests currently validate only totals, not the meta-information.
//
// Field order is insignificant for JSON unmarshalling, but we keep it the
// same as in the file for easier visual comparison.
type ExpectedReceipt struct {
	Hash     string         `json:"hash"`
	Sender   string         `json:"sender"`
	Subject  string         `json:"subject"`
	Date     string         `json:"date"`
	Total    float64        `json:"total"`
	Template string         `json:"template"`
	Items    []ExpectedItem `json:"items"`
}
