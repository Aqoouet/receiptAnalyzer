package main

import (
	"flag"
	"log"
)

func main() {
	saveEmails := flag.Bool("save_new_emails", false, "Скачать новые письма и сохранить их в каталог msg_html")
	importHTML := flag.Bool("import_saved_html", false, "Импортировать ранее сохранённые HTML-письма в базу данных")
	quantity := flag.Int("quantityToProcess", -1, "Количество писем для обработки (-1 = все)")
	flag.Parse()

	// Проверяем взаимную исключительность режимов
	if *saveEmails && *importHTML {
		log.Fatalln("Флаги -save_new_emails и -import_saved_html нельзя использовать одновременно")
	}

	log.Printf("Запуск анализатора чеков. save_new_emails=%v, import_saved_html=%v, quantityToProcess=%d", *saveEmails, *importHTML, *quantity)

	cfg, err := LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}

	log.Println("Конфигурация успешно загружена")

	if *saveEmails {
		// --- Режим сохранения писем ---
		imapClient, err := ConnectToEmail(cfg)
		if err != nil {
			log.Fatalf("Ошибка подключения к почте: %v", err)
		}
		defer imapClient.Close()

		log.Println("Соединение с почтовым сервером установлено")

		messages, err := FetchMessages(imapClient, *quantity)
		if err != nil {
			log.Fatalf("Ошибка получения сообщений: %v", err)
		}

		log.Printf("Получено %d писем", len(messages))
		SaveMessages(messages)
		log.Println("Сохранение писем завершено")
	} else if *importHTML {
		// --- Режим импорта HTML в базу ---
		store, err := InitializeStorage(cfg)
		if err != nil {
			log.Fatalf("Ошибка инициализации хранилища: %v", err)
		}

		log.Println("Хранилище инициализировано")

		imported, err := ImportSavedHTML(store)
		if err != nil {
			log.Fatalf("Ошибка импорта HTML: %v", err)
		}
		log.Printf("Импортировано чеков: %d", imported)
	} else {
		log.Println("Ни один режим не выбран. Запустите программу с -save_new_emails или -import_saved_html.")
	}
}
