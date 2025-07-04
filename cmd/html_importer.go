package main

import (
    "fmt"
    "io/ioutil"
    "log"
    "strings"
    "time"

    "receiptAnalyzer/internal/receipt"
    "receiptAnalyzer/internal/storage"
)

// ImportSavedHTML проходит по каталогу msg_html и добавляет сведения о каждом
// HTML-файле в базу. Возвращает количество успешно импортированных чеков.
// Файл считается уже импортированным, если в таблице существует запись с таким id.
func ImportSavedHTML(store storage.Storage) (int, error) {
    files, err := ioutil.ReadDir("msg_html")
    if err != nil {
        return 0, fmt.Errorf("не удалось прочитать каталог msg_html: %w", err)
    }

    imported := 0
    for _, f := range files {
        if f.IsDir() || !strings.HasSuffix(f.Name(), ".html") {
            continue
        }

        id := f.Name() // используем имя файла как первичный ключ

        r := receipt.Receipt{
            ID:        id,
            Shop:      "Неизвестно",
            DateTime:  f.ModTime(),
            Total:     0,
            Source:    "saved_html",
            CreatedAt: time.Now(),
        }

        if err := store.SaveReceipt(r); err != nil {
            // скорее всего это дубликат по PRIMARY KEY – игнорируем
            if strings.Contains(err.Error(), "UNIQUE") {
                log.Printf("Чек %s уже существует, пропускаем", id)
                continue
            }
            return imported, fmt.Errorf("ошибка сохранения чека %s: %w", id, err)
        }
        imported++
        log.Printf("Чек из файла %s добавлен в базу", id)
    }
    return imported, nil
} 