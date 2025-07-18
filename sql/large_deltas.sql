-- Анализ чеков с большой дельтой (|delta_sum| > 50)
-- Можно выполнить в SQLite: `.read large_deltas.sql

-- Включаем форматированный вывод
.mode column
.headers on
.width 30 15 15 15 50

-- Общая статистика по большим дельтам
SELECT 
    '=== ОБЩАЯ СТАТИСТИКА ПО БОЛЬШИМ ДЕЛЬТАМ ===' as info;

SELECT 
    COUNT(*) as 'Всего чеков с |delta| > 50',
    ROUND(SUM(ABS(delta_sum)), 2) as 'Сумма абсолютных дельт',
    ROUND(AVG(ABS(delta_sum)), 2) as 'Средняя абсолютная дельта',
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM receipts), 2) as '% от всех чеков'
FROM receipts 
WHERE ABS(delta_sum) > 50;

-- Топ-20 чеков с наибольшей дельтой
SELECT 
    '=== ТОП-20 ЧЕКОВ С НАИБОЛЬШЕЙ ДЕЛЬТОЙ ===' as info;

SELECT 
    COALESCE(shop, 'Не указан') as 'Магазин',
    ROUND(total, 2) as 'Заявленная сумма',
    ROUND(delta_sum, 2) as 'Дельта',
    ROUND(ABS(delta_sum), 2) as '|Дельта|',
    subject as 'Тема'
FROM receipts 
WHERE ABS(delta_sum) > 50
ORDER BY ABS(delta_sum) DESC 
LIMIT 20;

-- Статистика по магазинам с большими дельтами
SELECT 
    '=== СТАТИСТИКА ПО МАГАЗИНАМ ===' as info;

SELECT 
    COALESCE(shop, 'Не указан') as 'Магазин',
    COUNT(*) as 'Чеков с |delta| > 50',
    ROUND(SUM(ABS(delta_sum)), 2) as 'Сумма |delta|',
    ROUND(AVG(ABS(delta_sum)), 2) as 'Средняя |delta|',
    ROUND(MAX(ABS(delta_sum)), 2) as 'Максимальная |delta|'
FROM receipts 
WHERE ABS(delta_sum) > 50
GROUP BY shop 
ORDER BY SUM(ABS(delta_sum)) DESC;

-- Ссылки на чеки с наибольшей дельтой
SELECT 
    '=== ССЫЛКИ НА ЧЕКИ С НАИБОЛЬШЕЙ ДЕЛЬТОЙ ===' as info;

.mode list
SELECT 
    ROW_NUMBER() OVER (ORDER BY ABS(delta_sum) DESC) || '|' || 
    ROUND(ABS(delta_sum), 2) || '|' || 
    COALESCE(shop, 'Не указан') || '|' || 
    link as '№|Дельта|Магазин|Ссылка'
FROM receipts 
WHERE ABS(delta_sum) > 50
ORDER BY ABS(delta_sum) DESC 
LIMIT 10;

-- Возвращаем форматирование
.mode column
.headers on 