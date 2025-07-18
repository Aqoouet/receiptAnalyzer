-- Статистика по шаблонам (templates) чеков, у которых не проставлено поле «shop»
-- Можно выполнить в SQLite: `.read top_missing_shop_receipts.sql`

-- Подсчет чеков по шаблонам, где поле shop не заполнено
SELECT 
    'Шаблон' as template_header,
    'Чеков без магазина' as count_header,
    'Процент' as percentage_header;

SELECT 
    template,
    COUNT(*) as receipts_without_shop,
    ROUND(COUNT(*) * 100.0 / (SELECT COUNT(*) FROM receipts WHERE template IS NOT NULL AND template != ''), 2) as percentage
FROM receipts
WHERE (shop IS NULL OR shop = '') 
    AND template IS NOT NULL 
    AND template != ''
GROUP BY template
ORDER BY receipts_without_shop DESC;

-- Общая статистика
SELECT 
    '=== ОБЩАЯ СТАТИСТИКА ===' as info;

SELECT 
    COUNT(*) as total_receipts_without_shop,
    COUNT(*) * 100.0 / (SELECT COUNT(*) FROM receipts) as percentage_of_total
FROM receipts
WHERE shop IS NULL OR shop = '';

-- Примеры чеков без магазина по каждому шаблону
SELECT 
    '=== ПРИМЕРЫ ЧЕКОВ БЕЗ МАГАЗИНА ===' as info;

SELECT 
    template,
    link
FROM (
    SELECT 
        template,
        link,
        ROW_NUMBER() OVER (PARTITION BY template ORDER BY id) as rn
    FROM receipts
    WHERE (shop IS NULL OR shop = '') 
        AND template IS NOT NULL 
        AND template != ''
)
WHERE rn = 1
ORDER BY template;

-- ПРИМЕЧАНИЕ: Для просмотра HTML-содержимого чеков выполните:
-- sqlite3 output/db_dir/receipts.db < sql/top_missing_shop_receipts.sql > results.txt
-- Затем для каждого файла из результатов выполните:
-- cat "file:///path/to/html/file" | head -100 