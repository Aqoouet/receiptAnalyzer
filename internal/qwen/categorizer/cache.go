package categorizer

import (
	"encoding/json"
	"os"
	"sync"
)

// Путь к файлу, где будет храниться кэш категорий.
const cacheFilePath = "internal/qwen/category_cache.json"

// categoryCache хранит уже определённые категории для товаров.
var (
	categoryCache map[string]string
	cacheMu       sync.RWMutex
)

// loadCategoryCache загружает кэш из файла при старте сервера.
func loadCategoryCache() {
	categoryCache = make(map[string]string)
	data, err := os.ReadFile(cacheFilePath)
	if err != nil {
		// файла нет — начинаем с пустого кэша
		return
	}
	_ = json.Unmarshal(data, &categoryCache)
}

// saveCategoryCache сохраняет кэш в файл.
func saveCategoryCache() {
	cacheMu.RLock()
	data, _ := json.MarshalIndent(categoryCache, "", "  ")
	cacheMu.RUnlock()
	_ = os.WriteFile(cacheFilePath, data, 0644)
}
