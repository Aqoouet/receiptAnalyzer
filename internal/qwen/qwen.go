package qwen

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type QwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type QwenRequest struct {
	Model       string        `json:"model"`
	Messages    []QwenMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type QwenChoice struct {
	Message QwenMessage `json:"message"`
}

type QwenResponse struct {
	Choices []QwenChoice `json:"choices"`
}

// CategorizeItems отправляет список товаров в Qwen через OpenRouter и возвращает map[товар]категория
func CategorizeItems(items []string) (map[string]string, error) {
	prompt := buildPrompt(items)

	// Используем бесплатную модель Qwen через OpenRouter
	qwenReq := QwenRequest{
		Model: "qwen/qwen2.5-vl-32b-instruct:free", // Бесплатная модель для категоризации товаров
		Messages: []QwenMessage{
			{Role: "system", Content: "Ты — ассистент, который категоризирует товары по категориям."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   1500, // Увеличено до 1500 для больших batch size
		Temperature: 0.1,
	}
	data, _ := json.Marshal(qwenReq)

	// Логируем весь запрос
	log.Printf("🔍 Отправляем запрос к Qwen через OpenRouter:")
	log.Printf("📝 Промпт: %s", prompt)
	log.Printf("📦 JSON запрос: %s", string(data))

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(data))
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY is not set")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/receiptAnalyzer")
	req.Header.Set("X-Title", "ReceiptAnalyzer")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ Ошибка при отправке запроса: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Логируем ответ
	log.Printf("📥 Получен ответ от Qwen (статус: %d)", resp.StatusCode)

	// Проверяем на код 429 (Too Many Requests)
	if resp.StatusCode == 429 {
		log.Printf("🚫 Получен код 429 (Too Many Requests). Прекращаем отправку запросов.")
		return nil, errors.New("rate limit exceeded (429)")
	}

	var qwenResp QwenResponse
	if err := json.Unmarshal(body, &qwenResp); err != nil {
		log.Printf("❌ Ошибка парсинга ответа: %v", err)
		return nil, err
	}
	if len(qwenResp.Choices) == 0 {
		log.Printf("❌ Пустой ответ от Qwen")
		return nil, errors.New("empty response from Qwen")
	}

	// Очищаем ответ от markdown-разметки
	content := qwenResp.Choices[0].Message.Content
	content = cleanJsonResponse(content)

	var categories map[string]string
	if err := json.Unmarshal([]byte(content), &categories); err != nil {
		log.Printf("❌ Ошибка парсинга категорий: %v", err)
		log.Printf("📝 Содержимое ответа: %s", content)
		return nil, err
	}

	log.Printf("✅ Успешно получены категории: %+v", categories)

	// Детальное логирование пар товар-категория
	log.Printf("📋 Пары товар-категория:")
	for item, category := range categories {
		log.Printf("   %s → %s", item, category)
	}

	return categories, nil
}

func buildPrompt(items []string) string {
	prompt := "Категоризируй товары по категориям. Верни результат в формате JSON: {\"товар1\": \"категория1\", ...}\n\nПримеры:\n- Молоко Простоквашино → \"Молочные продукты\"\n- Макароны Макфа → \"Бакалея\"\n- Квас Добрый → \"Безалкогольные напитки\"\n- Хлеб Бородинский → \"Хлебобулочные изделия\"\n\nСписок товаров:\n"
	for _, item := range items {
		prompt += "- " + item + "\n"
	}
	return prompt
}

// cleanJsonResponse очищает JSON от markdown-разметки
func cleanJsonResponse(content string) string {
	// Убираем ```json в начале
	if len(content) > 7 && content[:7] == "```json" {
		content = content[7:]
	}
	// Убираем ``` в конце
	if len(content) > 3 && content[len(content)-3:] == "```" {
		content = content[:len(content)-3]
	}
	// Убираем лишние пробелы и переносы строк
	content = strings.TrimSpace(content)
	return content
}
