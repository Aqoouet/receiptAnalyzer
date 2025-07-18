-- Топ-10 чеков по сумме для заданного магазина
-- Использование: sqlite3 receipts.db < top_receipts_by_shop.sql
-- Или в интерактивном режиме: .read top_receipts_by_shop.sql

-- Включаем форматированный вывод
.mode column
.headers on
.width 12 20 15 50

-- Параметр магазина (можно изменить)
-- Примеры: 'Whoosh', 'ЯНДЕКС ЕДА', 'Вкусно и Точка'
-- Для всех магазинов: закомментируйте WHERE shop = '...'

SELECT 
    '=== ТОП-10 ЧЕКОВ ПО СУММЕ ===' as info;

SELECT 
    date_time as 'Дата',
    ROUND(total, 2) as 'Сумма',
    subject as 'Тема',
    link as 'Ссылка на чек'
FROM receipts 
WHERE shop = 'Неизвестный магазин'  -- ИЗМЕНИТЕ НА НУЖНЫЙ МАГАЗИН
  AND total > 0
ORDER BY total DESC 
LIMIT 10;

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
WHERE shop = 'Whoosh'  -- ИЗМЕНИТЕ НА НУЖНЫЙ МАГАЗИН
  AND total > 0; 