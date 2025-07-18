#!/usr/bin/env bash

set -e

echo "Экспортируем данные в XLSX..."

# Переходим в корневую директорию проекта
cd "$(dirname "$0")/.."

# Запускаем экспорт
go run ./cmd/xlsxexporter/main.go -export-only

echo "Экспорт завершен. Файл: output/db_dir/receipts.xlsx" 