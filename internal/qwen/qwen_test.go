package qwen

import (
	"os"
	"testing"
)

func TestCategorizeItems_Integration(t *testing.T) {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		t.Skip("OPENROUTER_API_KEY not set; skipping integration test")
	}
	items := []string{"Молоко Простоквашино 1л", "Яблоки Гала 1кг", "Шоколад Alpen Gold 90г"}
	cats, err := CategorizeItems(items)
	if err != nil {
		t.Fatalf("Integration request failed: %v", err)
	}
	if len(cats) != len(items) {
		t.Errorf("Expected %d categories, got %d: %v", len(items), len(cats), cats)
	}
	for _, item := range items {
		if cats[item] == "" {
			t.Errorf("No category for item: %s", item)
		}
	}
}
