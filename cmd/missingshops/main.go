package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jaytaylor/html2text"
	"github.com/olekukonko/tablewriter"
	_ "modernc.org/sqlite"
)

func main() {
	// Путь к базе можно передать первым аргументом, иначе используем дефолтный
	dbPath := "output/db_dir/receipts.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("DB open error: %v", err)
	}
	defer db.Close()

	// Получаем статистику чеков без магазина
	statsRows, err := db.Query(`
		SELECT template,
		       COUNT(*) as cnt,
		       ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM receipts WHERE template IS NOT NULL AND template != ''), 2) as percentage
		FROM receipts
		WHERE (shop IS NULL OR shop = '')
		  AND template IS NOT NULL AND template != ''
		GROUP BY template
		ORDER BY cnt DESC;
	`)
	if err != nil {
		log.Fatalf("Query error: %v", err)
	}
	defer statsRows.Close()

	// Выводим таблицу статистики
	fmt.Println("=== СТАТИСТИКА ЧЕКОВ БЕЗ МАГАЗИНА ===")
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Шаблон", "Чеков без магазина", "% от шаблонных чеков"})

	// для примеров
	type sample struct{ template, link string }
	var samples []sample

	for statsRows.Next() {
		var tpl string
		var cnt int
		var pct float64
		if err := statsRows.Scan(&tpl, &cnt, &pct); err != nil {
			log.Fatalf("Scan error: %v", err)
		}
		table.Append([]string{tpl, fmt.Sprintf("%d", cnt), fmt.Sprintf("%.2f", pct)})

		// Сохраняем пример — первый чек для шаблона
		var link string
		db.QueryRow(`SELECT link FROM receipts WHERE template = ? AND (shop IS NULL OR shop = '') ORDER BY id LIMIT 1`, tpl).Scan(&link)
		samples = append(samples, sample{tpl, link})
	}
	table.Render()

	fmt.Println()
	fmt.Println("=== ПРИМЕРЫ ЧЕКОВ (только текст) ===")

	for _, s := range samples {
		path := strings.TrimPrefix(s.link, "file://")
		data, err := ioutil.ReadFile(path)
		if err != nil {
			fmt.Printf("\n%s (read error: %v)\n", path, err)
			continue
		}
		text, err := html2text.FromString(string(data), html2text.Options{PrettyTables: false})
		if err != nil {
			fmt.Printf("\n%s (html2text error: %v)\n", path, err)
			continue
		}
		fmt.Printf("\n--- %s ---\n", s.template)
		fmt.Printf("Файл: %s\n", s.link)
		fmt.Printf("%s\n", firstNLines(text, 100))
		fmt.Printf("\n%s\n", strings.Repeat("=", 80))
	}
}

// firstNLines возвращает первые n строк из текста
func firstNLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}
