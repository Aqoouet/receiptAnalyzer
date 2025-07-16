package mailfetcher

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"receiptAnalyzer/internal/config"
	"strings"
	"time"
)

// Глобальная переменная для конфигурации
var globalConfig *config.Config

func cleanUpHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[cleanUpHandler] Старт очистки писем")
	htmlDir := globalConfig.Paths.HTMLDirPath
	files, err := os.ReadDir(htmlDir)
	if err != nil {
		log.Printf("[cleanUpHandler] Ошибка чтения директории: %v", err)
		http.Error(w, "Ошибка чтения директории: "+err.Error(), 500)
		return
	}
	deleted := 0
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".html") {
			continue
		}
		err := os.Remove(filepath.Join(htmlDir, f.Name()))
		if err == nil {
			deleted++
		}
	}
	msg := fmt.Sprintf("<html><body><h2>Удалено писем: %d</h2></body></html>", deleted)
	log.Printf("[cleanUpHandler] Завершено, удалено писем: %d", deleted)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(msg))
}

func StartServer() {
	log.Println("[StartServer] Запуск сервера mailfetcher")

	// Загружаем конфигурацию один раз при старте
	var err error
	globalConfig, err = config.LoadConfig("")
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}
	log.Printf("[StartServer] Конфигурация загружена для mailfetcher")

	http.HandleFunc("/fetch-emails", fetchEmailsHandler)
	http.HandleFunc("/clean_up", cleanUpHandler)

	port := os.Getenv("MAILFETCHER_PORT")
	if port == "" {
		port = fmt.Sprintf("%d", globalConfig.Ports.Mailfetcher)
	}
	log.Printf("[StartServer] Mailfetcher listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// isVPNActive проверяет наличие активного VPN-интерфейса или процесса
func isVPNActive() bool {
	// Проверяем наличие интерфейсов tun0 или wg0
	ifaces := []string{"tun0", "wg0"}
	for _, iface := range ifaces {
		if _, err := os.Stat("/sys/class/net/" + iface); err == nil {
			return true
		}
	}
	// Проверяем наличие процессов openvpn, wireguard, wg-quick
	vpnProcs := []string{"openvpn", "wireguard", "wg-quick"}
	for _, proc := range vpnProcs {
		cmd := exec.Command("pgrep", proc)
		if err := cmd.Run(); err == nil {
			return true
		}
	}
	return false
}

// isIPFromRussia определяет, российский ли внешний IP (через api.myip.com)
func isIPFromRussia() bool {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.myip.com")
	if err != nil {
		log.Printf("Ошибка получения внешнего IP: %v", err)
		return false // если не удалось проверить — считаем, что не из РФ
	}
	defer resp.Body.Close()
	var data struct {
		Country string `json:"country"`
		CC      string `json:"cc"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Printf("Ошибка декодирования ответа IP API: %v", err)
		return false
	}
	log.Printf("Внешний IP: страна=%s, код=%s", data.Country, data.CC)
	return data.CC == "RU"
}

func fetchEmailsHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("[fetchEmailsHandler] Старт обработки запроса /fetch-emails")
	if !isIPFromRussia() {
		msg := "<html><body><h2>Внешний IP не из России. Пожалуйста, выключите VPN для работы с почтой.</h2></body></html>"
		log.Println("[fetchEmailsHandler] Внешний IP не из России. Пожалуйста, выключите VPN для работы с почтой.")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(503)
		w.Write([]byte(msg))
		return
	}
	if isVPNActive() {
		msg := "<html><body><h2>Обнаружен VPN. Пожалуйста, выключите VPN для работы с почтой.</h2></body></html>"
		log.Println("[fetchEmailsHandler] Обнаружен VPN. Пожалуйста, выключите VPN для работы с почтой.")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(503)
		w.Write([]byte(msg))
		return
	}

	added, err := FetchAndSave(globalConfig)
	if err != nil {
		log.Printf("[fetchEmailsHandler] Ошибка FetchAndSave: %v", err)
		http.Error(w, err.Error(), 500)
		return
	}
	log.Println("[fetchEmailsHandler] Завершено успешно")
	msg := fmt.Sprintf("<html><body><h2>Добавлено писем: %d</h2></body></html>", added)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(msg))
}
