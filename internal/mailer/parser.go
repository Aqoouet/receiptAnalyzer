package mailer

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/emersion/go-message"
)

// IsReceiptEmail проверяет, является ли письмо письмом с чеком
// Анализирует тему письма на наличие слова "чек" в любом регистре
func IsReceiptEmail(e *message.Entity) bool {
	subject := e.Header.Get("Subject")
	if subject == "" {
		return false
	}
	return strings.Contains(strings.ToLower(subject), "чек")
}

// ExtractTextFromHTML извлекает чистый текст из HTML-содержимого письма
// Использует библиотеку goquery для парсинга HTML
func ExtractTextFromHTML(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}
	return doc.Text(), nil
}

// GetBodyText извлекает текстовое содержимое письма
// Поддерживает как простой текст, так и HTML-формат
func GetBodyText(e *message.Entity) (string, error) {
	if e.Body == nil {
		return "", errors.New("тело письма отсутствует")
	}

	body, err := io.ReadAll(e.Body)
	if err != nil {
		return "", err
	}

	contentType := e.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return ExtractTextFromHTML(string(body))
	}

	return string(body), nil
}

// ProcessEmail обрабатывает письмо и возвращает его содержимое
func ProcessEmail(e *message.Entity) (string, error) {
	if !IsReceiptEmail(e) {
		return "", fmt.Errorf("письмо не является чеком")
	}

	text, err := GetBodyText(e)
	if err != nil {
		return "", err
	}

	return text, nil
}
