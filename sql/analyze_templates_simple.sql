-- Упрощенный анализ статистики по темплейтам и чекам с товарами total = 0
-- Запуск: sqlite3 output/db_dir/receipts.db < analyze_templates_simple.sql

-- Настройка форматирования вывода
.mode column
.headers on
.separator " | "
.width 30 25 15 15

-- 1. Общая статистика по темплейтам
SELECT '==========================================' as separator;
SELECT '=== ОБЩАЯ СТАТИСТИКА ПО ТЕМПЛЕЙТАМ ===' as info;
SELECT '==========================================' as separator;

SELECT 
    template,
    COUNT(*) as total_receipts,
    ROUND(SUM(total), 2) as total_amount,
    ROUND(AVG(total), 2) as avg_amount
FROM receipts 
WHERE template IS NOT NULL AND template != ''
GROUP BY template 
ORDER BY total_receipts DESC;

-- 2. Статистика по товарам с total = 0
SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== ТОВАРЫ С TOTAL = 0 ===' as info;
SELECT '==========================================' as separator;

SELECT 
    COUNT(*) as items_with_zero_total,
    COUNT(DISTINCT hash) as receipts_with_zero_items
FROM items 
WHERE total = 0 OR total IS NULL;

-- 3. Статистика по темплейтам для товаров с total = 0
SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== СТАТИСТИКА ПО ТЕМПЛЕЙТАМ ДЛЯ ТОВАРОВ С TOTAL = 0 ===' as info;
SELECT '==========================================' as separator;

SELECT 
    r.template,
    COUNT(i.id) as zero_total_items,
    COUNT(DISTINCT r.hash) as receipts_with_zero_items
FROM items i
JOIN receipts r ON i.hash = r.hash
WHERE i.total = 0 OR i.total IS NULL
GROUP BY r.template
ORDER BY zero_total_items DESC;

-- 4. Топ товаров с total = 0
SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== ТОП ТОВАРОВ С TOTAL = 0 ===' as info;
SELECT '==========================================' as separator;

SELECT 
    SUBSTR(i.name, 1, 50) as item_name,
    COUNT(*) as occurrence_count,
    COUNT(DISTINCT r.hash) as receipts_count
FROM items i
JOIN receipts r ON i.hash = r.hash
WHERE i.total = 0 OR i.total IS NULL
GROUP BY i.name
ORDER BY occurrence_count DESC
LIMIT 20;

-- 5. Анализ проблемных темплейтов
SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== АНАЛИЗ ПРОБЛЕМНЫХ ТЕМПЛЕЙТОВ ===' as info;
SELECT '==========================================' as separator;

SELECT 
    r.template,
    COUNT(r.hash) as total_receipts,
    COUNT(CASE WHEN i.total = 0 OR i.total IS NULL THEN 1 END) as receipts_with_zero_items,
    ROUND(
        (COUNT(CASE WHEN i.total = 0 OR i.total IS NULL THEN 1 END) * 100.0) / COUNT(r.hash), 
        2
    ) as percentage_with_zero_items
FROM receipts r
LEFT JOIN items i ON r.hash = i.hash AND (i.total = 0 OR i.total IS NULL)
WHERE r.template IS NOT NULL AND r.template != ''
GROUP BY r.template
HAVING receipts_with_zero_items > 0
ORDER BY percentage_with_zero_items DESC;

-- 6. Временная статистика проблем
SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== ВРЕМЕННАЯ СТАТИСТИКА ПРОБЛЕМ ===' as info;
SELECT '==========================================' as separator;

SELECT 
    strftime('%Y-%m', r.date_time) as month,
    COUNT(r.hash) as total_receipts,
    COUNT(CASE WHEN i.total = 0 OR i.total IS NULL THEN 1 END) as receipts_with_zero_items,
    ROUND(
        (COUNT(CASE WHEN i.total = 0 OR i.total IS NULL THEN 1 END) * 100.0) / COUNT(r.hash), 
        2
    ) as percentage_with_zero_items
FROM receipts r
LEFT JOIN items i ON r.hash = i.hash AND (i.total = 0 OR i.total IS NULL)
GROUP BY month
ORDER BY month DESC;

-- 7. Примеры товаров с total = 0
SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== ПРИМЕРЫ ТОВАРОВ С TOTAL = 0 ===' as info;
SELECT '==========================================' as separator;

SELECT 
    SUBSTR(i.name, 1, 40) as item_name,
    i.quantity,
    i.unit_price,
    i.total,
    SUBSTR(r.template, 1, 25) as template,
    SUBSTR(r.sender, 1, 25) as sender,
    strftime('%Y-%m-%d %H:%M', r.date_time) as date_time
FROM items i
JOIN receipts r ON i.hash = r.hash
WHERE i.total = 0 OR i.total IS NULL
ORDER BY r.date_time DESC
LIMIT 10;

SELECT '' as empty_line;
SELECT '==========================================' as separator;
SELECT '=== АНАЛИЗ ЗАВЕРШЕН ===' as info;
SELECT '==========================================' as separator; 