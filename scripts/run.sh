#!/usr/bin/env bash

set -e

# Запуск всех микросервисов в фоне с логами

echo "Запуск mailfetcher..."
go run ./cmd/mailfetcher/ &
MAILFETCHER_PID=$!

echo "Запуск htmlimporter..."
go run ./cmd/htmlimporter/ &
HTMLIMPORTER_PID=$!

echo "Запуск xlsxexporter..."
go run ./cmd/xlsxexporter/ &
XLSXEXPORTER_PID=$!

echo "Запуск qwencategorizer..."
go run ./cmd/qwencategorizer/ &
QWENCATEGORIZER_PID=$!

echo "Запуск orchestrator..."
go run ./cmd/orchestrator/ &
ORCHESTRATOR_PID=$!

# Функция для остановки всех сервисов
function cleanup {
  echo "\nОстанавливаю все сервисы..."
  kill $MAILFETCHER_PID $HTMLIMPORTER_PID $XLSXEXPORTER_PID $QWENCATEGORIZER_PID $ORCHESTRATOR_PID 2>/dev/null || true
  wait
}

trap cleanup EXIT

echo "Все сервисы запущены. Для остановки нажмите Ctrl+C."
wait 