-- Пример SQL скрипта для добавления поля shop в таблицу receipts
-- Выполните этот скрипт, если хотите добавить поле shop

-- Добавляем поле shop в таблицу receipts
ALTER TABLE receipts ADD COLUMN shop TEXT;

-- Создаем индекс для быстрого поиска по магазину
CREATE INDEX IF NOT EXISTS idx_receipts_shop ON receipts(shop);

-- Пример обновления данных для существующих записей
-- UPDATE receipts SET shop = 'Перекресток' WHERE sender LIKE '%beeline%';
-- UPDATE receipts SET shop = 'Спар' WHERE sender LIKE '%taxcom%';

-- После добавления поля shop, вы можете использовать его в исправлениях:
/*
{
  "hash": "example_hash",
  "fields": [
    {
      "field": "shop",
      "value": "Магнит"
    },
    {
      "field": "sender",
      "value": "corrected@example.com"
    }
  ]
}
*/ 