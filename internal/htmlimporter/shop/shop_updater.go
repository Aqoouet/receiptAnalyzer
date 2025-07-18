package shop

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// ShopMapping represents the mapping from search keys to shop names
type ShopMapping map[string]string

// LoadShopMapping loads shop names mapping from JSON file
func LoadShopMapping() (ShopMapping, error) {
	// Try to find shop_names.json in the same directory as this file
	jsonPath := filepath.Join("internal", "htmlimporter", "shop", "shop_names.json")

	// If not found, try current directory
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		jsonPath = "shop_names.json"
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}

	var mapping ShopMapping
	if err := json.Unmarshal(data, &mapping); err != nil {
		return nil, err
	}

	return mapping, nil
}

// UpdateShops updates shop field in receipts table based on HTML content
func UpdateShops(db *sql.DB) (int, int, error) {
	// Add shop column if it doesn't exist
	if _, err := db.Exec("ALTER TABLE receipts ADD COLUMN shop TEXT"); err != nil {
		// Ignore error if column already exists
		if !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
			return 0, 0, err
		}
	}

	// Load shop mapping
	mapping, err := LoadShopMapping()
	if err != nil {
		return 0, 0, err
	}

	// Select receipts without shop
	rows, err := db.Query("SELECT id, link FROM receipts WHERE shop IS NULL OR shop = ''")
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	updated := 0
	skipped := 0

	for rows.Next() {
		var id int
		var link string
		if err := rows.Scan(&id, &link); err != nil {
			skipped++
			continue
		}

		// Convert file:// URI to file path
		path := strings.TrimPrefix(link, "file://")
		content, err := os.ReadFile(path)
		if err != nil {
			skipped++
			continue
		}

		lower := strings.ToLower(string(content))
		shopName := ""

		// Find matching shop name
		for k, v := range mapping {
			if strings.Contains(lower, strings.ToLower(k)) {
				shopName = v
				break
			}
		}

		if shopName == "" {
			shopName = "Неизвестный магазин"
		}

		if _, err := db.Exec("UPDATE receipts SET shop = ? WHERE id = ?", shopName, id); err != nil {
			skipped++
			continue
		}

		updated++
		log.Printf("Updated receipt %d with shop: %s", id, shopName)
	}

	return updated, skipped, nil
}

// ClearShops clears shop field for all receipts
func ClearShops(db *sql.DB) (int, error) {
	result, err := db.Exec("UPDATE receipts SET shop = NULL")
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	log.Printf("Cleared shop field for %d receipts", rowsAffected)
	return int(rowsAffected), nil
}
