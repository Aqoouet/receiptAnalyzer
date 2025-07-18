-- Топ-10 чеков по сумме для заданного магазина
-- Использование: sqlite3 receipts.db < top_receipts_by_shop.sql
-- Или в интерактивном режиме: .read top_receipts_by_shop.sql

-- ПАРАМЕТР: ИЗМЕНИТЕ НАЗВАНИЕ МАГАЗИНА ЗДЕСЬ
-- Примеры: 'Whoosh', 'ЯНДЕКС ЕДА', 'Вкусно и Точка', 'Mr.Doors'
.parameter set :shop_name 'Mr.Doors'

-- Включаем форматированный вывод
.mode column
.headers on
.width 20 15 40

SELECT 
    '=== ТОП-10 ЧЕКОВ ПО СУММЕ ===' as info;

SELECT 
    date_time as 'Дата',
    ROUND(total, 2) as 'Сумма',
    subject as 'Тема'
FROM receipts
WHERE shop = :shop_name
  AND total > 0
ORDER BY total DESC 
LIMIT 10;

-- Ссылки на чеки (отдельно для удобства копирования)
SELECT 
    '=== ССЫЛКИ НА ЧЕКИ ===' as info;

.mode list
SELECT 
    ROW_NUMBER() OVER (ORDER BY total DESC) || '|' || link as '№|Ссылка'
FROM receipts
WHERE shop = :shop_name
  AND total > 0
ORDER BY total DESC 
LIMIT 10;

-- Возвращаем форматирование для статистики
.mode column
.headers on
-- Статистика по магазину
SELECT 
    '=== СТАТИСТИКА ПО МАГАЗИНУ ===' as info;

SELECT 
    COUNT(*) as 'Всего чеков',
    ROUND(SUM(total), 2) as 'Общая сумма',
    ROUND(AVG(total), 2) as 'Средний чек',
    ROUND(MIN(total), 2) as 'Минимальный чек',
    ROUND(MAX(total), 2) as 'Максимальный чек'
FROM receipts
WHERE shop = :shop_name
  AND total > 0; 