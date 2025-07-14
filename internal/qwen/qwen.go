package qwen

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
)

type QwenMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type QwenRequest struct {
	Model    string        `json:"model"`
	Messages []QwenMessage `json:"messages"`
}

type QwenChoice struct {
	Message QwenMessage `json:"message"`
}

type QwenResponse struct {
	Choices []QwenChoice `json:"choices"`
}

// CategorizeItems отправляет список товаров в Qwen и возвращает map[товар]категория
func CategorizeItems(items []string) (map[string]string, error) {
	prompt := buildPrompt(items)

	qwenReq := QwenRequest{
		Model: "qwen/qwen3-235b-a22b",
		Messages: []QwenMessage{
			{Role: "system", Content: "Ты — ассистент, который категоризирует товары по категориям."},
			{Role: "user", Content: prompt},
		},
	}
	data, _ := json.Marshal(qwenReq)

	req, _ := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(data))
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		return nil, errors.New("OPENROUTER_API_KEY is not set")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://yourdomain.com") // можно указать свой домен
	req.Header.Set("X-Title", "Receipt Categorizer")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var qwenResp QwenResponse
	if err := json.Unmarshal(body, &qwenResp); err != nil {
		return nil, err
	}
	if len(qwenResp.Choices) == 0 {
		return nil, errors.New("empty response from Qwen")
	}

	var categories map[string]string
	if err := json.Unmarshal([]byte(qwenResp.Choices[0].Message.Content), &categories); err != nil {
		return nil, err
	}
	return categories, nil
}

func buildPrompt(items []string) string {
	prompt := "Категоризируй товары по категориям. Верни результат в формате JSON: {\"товар1\": \"категория1\", ...}\n\nСписок товаров:\n"
	for _, item := range items {
		prompt += "- " + item + "\n"
	}
	return prompt
}
