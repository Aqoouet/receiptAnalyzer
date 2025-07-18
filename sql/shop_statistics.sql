-- Статистика по магазинам: количество чеков и сумма покупок
-- Можно выполнить в SQLite: `.read shop_statistics.sql`

-- Включаем форматированный вывод
.mode column
.headers on
.width 30 12 15 12

-- Статистика по магазинам
SELECT 
    COALESCE(shop, 'Не указан') as 'Магазин',
    COUNT(*) as 'Чеков',
    ROUND(SUM(total), 2) as 'Сумма',
    ROUND(AVG(total), 2) as 'Средний'
FROM receipts 
WHERE shop IS NOT NULL 
GROUP BY shop 
ORDER BY SUM(total) DESC;

-- Общая статистика
SELECT 
    '=== ОБЩАЯ СТАТИСТИКА ===' as info;

SELECT 
    COUNT(*) as 'Всего чеков',
    ROUND(SUM(total), 2) as 'Общая сумма',
    ROUND(AVG(total), 2) as 'Средний чек',
    COUNT(DISTINCT shop) as 'Уникальных магазинов'
FROM receipts 
WHERE shop IS NOT NULL;

-- Топ-10 магазинов по сумме покупок
SELECT 
    '=== ТОП-10 МАГАЗИНОВ ПО СУММЕ ===' as info;

SELECT 
    shop as 'Магазин',
    COUNT(*) as 'Чеков',
    ROUND(SUM(total), 2) as 'Сумма',
    ROUND(AVG(total), 2) as 'Средний'
FROM receipts 
WHERE shop IS NOT NULL 
GROUP BY shop 
ORDER BY SUM(total) DESC 
LIMIT 10; 