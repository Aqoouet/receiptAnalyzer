package mailer

import (
	"encoding/base64"
	"fmt"
	"log"

	"github.com/emersion/go-imap/client" // Используем библиотеку для работы с IMAP-серверами
)

// XOAUTH2Mech реализует интерфейс sasl.Client для аутентификации через XOAUTH2.
// Это позволяет использовать OAuth2 токен вместо логина/пароля для подключения к Яндекс.Почте.
type XOAUTH2Mech struct {
	email       string // Email пользователя (например, user@yandex.ru)
	accessToken string // OAuth2 токен, полученный через Яндекс.OAuth
}

// Name возвращает имя механизма SASL (XOAUTH2).
// Это требуется для соблюдения интерфейса sasl.Client.
func (m XOAUTH2Mech) Name() string {
	log.Printf("[DEBUG] XOAUTH2Mech.Name() called")
	return "XOAUTH2"
}

// Start формирует начальный ответ клиента в формате, требуемом XOAUTH2.
func (m XOAUTH2Mech) Start() (name string, ir []byte, err error) {
	log.Printf("[DEBUG] XOAUTH2Mech.Start() called")

	// Проверка: email и accessToken не должны быть пустыми
	if m.email == "" {
		log.Printf("[ERROR] Email is empty")
		return "", nil, fmt.Errorf("email is empty")
	}
	if m.accessToken == "" {
		log.Printf("[ERROR] Access token is empty")
		return "", nil, fmt.Errorf("access token is empty")
	}

	// Формируем строку в формате: user=email\x01auth=Bearer token\x01\x01
	raw := "user=" + m.email + "\x01auth=Bearer " + m.accessToken + "\x01\x01"

	log.Printf("[DEBUG] XOAUTH2 raw string (before base64): %q", raw)

	// Кодируем в base64
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
	base64.StdEncoding.Encode(encoded, []byte(raw))
	decoded := string(encoded)

	log.Printf("[DEBUG] XOAUTH2 encoded string (base64): %s", encoded)
	log.Printf("[DEBUG] XOAUTH2 encoded string (decoded back): %s", decoded)

	return "XOAUTH2", encoded, nil
}

// Next обрабатывает вызовы SASL (не требуется для XOAUTH2).
// Всегда возвращает nil, так как XOAUTH2 использует одноразовый токен.
func (m XOAUTH2Mech) Next(challenge []byte) ([]byte, error) {
	log.Printf("[DEBUG] XOAUTH2Mech.Next() called with challenge: %q", challenge)
	return nil, nil // Дополнительные шаги не нужны
}

// ConnectToYandexOAuth подключается к Яндекс.Почте через OAuth2 токен.
// Возвращает клиент IMAP или ошибку.
func ConnectToYandexOAuth(email, accessToken string) (*client.Client, error) {
	log.Printf("[INFO] Connecting to Yandex IMAP server using XOAUTH2...")

	// Проверяем входные параметры
	if email == "" || accessToken == "" {
		log.Printf("[ERROR] Email or access token is empty")
		return nil, fmt.Errorf("email or access token is empty")
	}

	// Подключаемся к IMAP-серверу Яндекса через TLS
	c, err := client.DialTLS("imap.yandex.ru:993", nil)
	if err != nil {
		log.Printf("[ERROR] Failed to connect to IMAP server: %v", err)
		return nil, fmt.Errorf("failed to connect to IMAP server: %v", err)
	}
	log.Printf("[INFO] Successfully connected to IMAP server")

	// Создаем механизм аутентификации XOAUTH2
	mech := XOAUTH2Mech{
		email:       email,
		accessToken: accessToken,
	}

	log.Printf("[INFO] Authenticating using XOAUTH2 for email: %s", email)

	// Выполняем аутентификацию через XOAUTH2
	if err := c.Authenticate(mech); err != nil {
		log.Printf("[ERROR] XOAUTH2 authentication failed: %v", err)

		// Попробуем прочитать последний ответ сервера, если он доступен
		if responseErr, ok := err.(interface{ Response() []byte }); ok {
			log.Printf("[ERROR] Server response: %q", string(responseErr.Response()))
		}

		c.Logout() // Закрываем соединение при ошибке
		return nil, fmt.Errorf("XOAUTH2 authentication failed: %v", err)
	}

	log.Printf("[INFO] XOAUTH2 authentication successful for email: %s", email)

	// Возвращаем клиент IMAP для дальнейшей работы
	return c, nil
}
