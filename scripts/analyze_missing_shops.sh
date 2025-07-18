#!/bin/bash

# Скрипт для анализа чеков без магазина с показом HTML-содержимого

echo "=== АНАЛИЗ ЧЕКОВ БЕЗ МАГАЗИНА ==="
echo

# Выполняем SQL-запрос и сохраняем результаты
sqlite3 output/db_dir/receipts.db < sql/top_missing_shop_receipts.sql > temp_results.txt

# Показываем статистику
head -20 temp_results.txt

echo
echo "=== HTML-СОДЕРЖИМОЕ ПРИМЕРОВ ЧЕКОВ ==="
echo

# Извлекаем ссылки на файлы из результатов
grep "file://" temp_results.txt | while read -r line; do
    # Извлекаем путь к файлу
    file_path=$(echo "$line" | sed 's/.*file:\/\///')
    
    if [ -f "$file_path" ]; then
        echo "=== ФАЙЛ: $file_path ==="
        echo
        # Показываем первые 200 строк текста без HTML-тегов
        sed -e 's/<[^>]*>//g' "$file_path" | head -200
        echo
        echo "=== КОНЕЦ ФАЙЛА ==="
        echo
        echo "Нажмите Enter для продолжения или Ctrl+C для выхода..."
        read -r
    else
        echo "Файл не найден: $file_path"
    fi
done

# Удаляем временный файл
rm temp_results.txt

echo "Анализ завершен." 