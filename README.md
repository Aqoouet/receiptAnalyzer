# ReceiptAnalyzer 📊

**ReceiptAnalyzer** — это мощное Go-приложение для автоматического анализа и обработки электронных чеков из почтовых ящиков. Приложение скачивает письма с чеками, парсит их содержимое, сохраняет данные в SQLite базу и экспортирует результаты в Excel файлы.

## 🎯 Основные возможности

- **📧 Автоматическая загрузка писем** с чеками через IMAP
- **🔍 Умный парсинг чеков** с поддержкой различных форматов (Beeline OFD, Taxcom)
- **💾 Сохранение в SQLite** для быстрого поиска и анализа
- **📊 Экспорт в Excel** с детальной информацией о покупках
- **🔄 Отслеживание дубликатов** для избежания повторной обработки
- **⚙️ Гибкая конфигурация** через YAML файлы

## 🏗️ Архитектура проекта

```
receiptAnalyzer/
├── cmd/                     # CLI-утилиты и бизнес-логика
│   ├── main.go              # Точка входа, обработка флагов
│   ├── config.go            # Загрузка/валидация YAML-конфигурации
│   ├── fetcher.go           # IMAP-клиент, загрузка писем
│   ├── saveMsg.go           # Сохранение HTML-писем + индекс хэшей
│   ├── html_importer.go     # Импорт HTML-чеков в БД
│   ├── xlsx_exporter.go     # Экспорт чеков → Excel
│   └── *_test.go            # Юнит- и интеграционные тесты
├── internal/                # Пакеты, не экспортируемые наружу
│   ├── receipt/             # Парсинг HTML-чеков
│   │   ├── model.go         # Структуры Receipt / Item
│   │   ├── parser.go        # Универсальный парсер + шаблоны
│   │   └── *_test.go        # Тесты парсера
│   └── storage/             # Слой хранения данных
│       ├── sqlite.go        # Реализация Storage на SQLite
│       └── *_test.go        # Тесты хранилища
├── config.yaml.example      # Пример конфигурации
├── run.sh                   # Базовый скрипт запуска/отладки
├── output/                  # Генерируется автоматически
│   ├── receipts.db          # SQLite-база после импорта
│   ├── receipts.xlsx        # Excel-отчёт
│   ├── msg_html/            # Сохранённые HTML-письма
│   └── state/               # Последний UID и др. вспомогательные файлы
├── go.mod / go.sum          # Зависимости Go-модулей
└── README.md                # Вы читаете его 😊
```

## 🚀 Быстрый старт

### Предварительные требования

- **Go 1.24+** — [скачать](https://golang.org/dl/)
- **IMAP-доступ** к почтовому ящику, где приходят чеки
- **OAuth-токен Яндекс**
  - 2-FA: создайте «Пароль приложения» — <https://passport.yandex.ru/profile>
  - Без 2-FA: получите токен — <https://oauth.yandex.ru/>

### Установка

1. **Клонируйте репозиторий:**
```bash
git clone https://github.com/Aqoouet/receiptAnalyzer.git
cd receiptAnalyzer
```

2. **Установите зависимости:**
```bash
go mod download
```

3. **Создайте конфигурацию:**
```bash
cp config.yaml.example config.yaml
# Отредактируйте config.yaml под ваши настройки
```

4. **Соберите приложение:**
```bash
go build -o receiptAnalyzer ./cmd
```

### Настройка конфигурации

Отредактируйте файл `config.yaml`:

```yaml
email:
  imap_server: "imap.yandex.ru:993"  # IMAP сервер
  username: "your-email@yandex.ru"   # Ваш email
  oauth_token: "your-oauth-token"    # OAuth токен

storage:
  db_path: "output/receipts.db"      # Путь к SQLite базе
  xlsx_path: "output/receipts.xlsx"  # Путь к Excel файлу

paths:
  html_dir: "output/msg_html"        # Папка для HTML писем
  state_dir: "output/state"          # Папка для состояния
```

## 📖 Использование

### Основные команды

#### 1. Скачивание новых писем с чеками
```bash
./receiptAnalyzer -config config.yaml -save_new_emails
```

**Что происходит:**
- Подключается к IMAP серверу
- Скачивает новые письма (начиная с последнего обработанного)
- Сохраняет HTML письма в `output/msg_html/`
- Обновляет состояние в `output/state/last_uid.txt`

#### 2. Импорт сохраненных HTML чеков в базу данных
```bash
./receiptAnalyzer -config config.yaml -import_saved_html
```

**Что происходит:**
- Читает HTML файлы из `output/msg_html/`
- Парсит чеки с помощью шаблонов
- Сохраняет данные в SQLite базу
- Создает Excel файл с результатами

#### 3. Ограничение количества обрабатываемых писем
```bash
./receiptAnalyzer -config config.yaml -save_new_emails -quantityToProcess 10
```

#### 4. Пересоздание базы данных
```bash
./receiptAnalyzer -config config.yaml -import_saved_html -rebuild_db
```

### Флаги командной строки

| Флаг | Описание | Пример |
|------|----------|--------|
| `-config` | Путь к YAML-конфигу (необязателен, по умолчанию `config.yaml`) | `-config custom.yaml` |
| `-save_new_emails` | Скачать новые письма | `-save_new_emails` |
| `-import_saved_html` | Импортировать HTML в базу | `-import_saved_html` |
| `-quantityToProcess` | Сколько писем обработать за один запуск.  
По умолчанию `-1` — обрабатываются **все** новые письма.  
Полезен при разработке/отладке, чтобы ускорить цикл и не скачивать большой архив:  
`-quantityToProcess 20` — обработать только первые 20 писем. | `-quantityToProcess 20` |
| `-rebuild_db` | Пересоздать базу данных | `-rebuild_db` |

## 📊 Структура данных

### База данных SQLite

#### Таблица `receipts`
| Поле | Тип | Описание |
|------|-----|----------|
| `id` | TEXT PRIMARY KEY | Уникальный ID чека |
| `shop` | TEXT | Название магазина |
| `date_time` | DATETIME | Дата и время покупки |
| `total` | REAL | Общая сумма чека |
| `source` | TEXT | Источник данных |
| `created_at` | DATETIME | Дата создания записи |

#### Таблица `items`
| Поле | Тип | Описание |
|------|-----|----------|
| `id` | INTEGER PRIMARY KEY | Уникальный ID позиции |
| `receipt_id` | TEXT | Ссылка на чек |
| `name` | TEXT | Название товара |
| `quantity` | REAL | Количество |
| `unit_price` | REAL | Цена за единицу |
| `total` | REAL | Сумма по позиции |

### Excel файл

Создается файл `output/receipts.xlsx` с двумя листами:

#### Лист "Чеки"
- ID чека
- Магазин
- Дата и время
- Сумма
- Источник
- Дата создания

#### Лист "Позиции"
- ID чека
- Название товара
- Количество
- Цена за единицу
- Сумма по позиции

## 🔧 Поддерживаемые форматы чеков

### Beeline OFD (Перекресток)
- **Отправитель:** `ofdreceipt@beeline.ru`
- **Формат:** HTML таблицы с CSS стилями
- **Поля:** Название товара, количество, цена

### Taxcom
- **Отправитель:** `noreply@taxcom.ru`
- **Формат:** HTML с CSS классами
- **Поля:** Название товара, количество, цена

### Добавление новых форматов

Для поддержки новых форматов чеков:

1. **Создайте новый шаблон** в `internal/receipt/parser.go`:
```go
var NewTemplate = Template{
    ItemSelector:       "div.new-item",
    NameSelector:       "span.item-name",
    PriceSelector:      "span.item-price",
    QtySelector:        "span.item-quantity",
    PriceCleanupRegexp: regexp.MustCompile(`[^0-9.,]`),
    QtyCleanupRegexp:   regexp.MustCompile(`[^0-9.,]`),
}
```

2. **Добавьте шаблон** в список используемых:
```go
[]receipt.Template{
    receipt.DefaultBeelineTemplate,
    receipt.DefaultTaxcomTemplate,
    receipt.NewTemplate,  // Новый шаблон
}
```

## 🔍 Анализ данных

### SQL запросы для анализа

#### Общая статистика
```sql
SELECT 
    COUNT(*) as total_receipts,
    SUM(total) as total_spent,
    AVG(total) as avg_receipt,
    MIN(date_time) as first_purchase,
    MAX(date_time) as last_purchase
FROM receipts;
```

#### Топ магазинов по тратам
```sql
SELECT 
    shop,
    COUNT(*) as receipts_count,
    SUM(total) as total_spent,
    AVG(total) as avg_receipt
FROM receipts 
GROUP BY shop 
ORDER BY total_spent DESC 
LIMIT 10;
```

#### Топ товаров
```sql
SELECT 
    name,
    COUNT(*) as purchase_count,
    SUM(quantity) as total_quantity,
    SUM(total) as total_spent
FROM items 
GROUP BY name 
ORDER BY total_spent DESC 
LIMIT 20;
```

#### Ежемесячная статистика
```sql
SELECT 
    strftime('%Y-%m', date_time) as month,
    COUNT(*) as receipts_count,
    SUM(total) as total_spent
FROM receipts 
GROUP BY month 
ORDER BY month;
```

## 🧪 Тестирование

В проекте реализовано более 200 модульных и интеграционных тестов ( `go test ./...` ).  Часть сценариев, требующих доработки или нестабильных из-за особенностей SQLite/файловой системы, помечена директивой `t.Skip()`.

Запуск всех тестов:
```bash
go test ./... -v
```

### Быстрый запуск отдельных пакетов
```bash
go test ./internal/storage      # только хранилище
go test ./cmd -run ImportSaved   # любое регулярное выражение для выбора тестов
```

## 🛠️  Отладка и логирование

Приложение пишет подробные логи в stdout. Для наглядности вы можете перенаправить вывод в файл:
```bash
./receiptAnalyzer -config config.yaml > run.log 2>&1
```

При необходимости повышайте уровень логирования в коде (например, добавьте `log.SetFlags(log.Lshortfile)` в `main.go`).

## 🔒 Безопасность

### Рекомендации по безопасности

1. **Храните OAuth токены в безопасном месте**
2. **Не коммитьте `config.yaml` с реальными данными**
3. **Используйте `.gitignore` для исключения:**
   ```
   config.yaml
   output/
   *.db
   *.xlsx
   ```

### Пример `.gitignore`
```gitignore
# Конфигурация
config.yaml

# Результаты работы
output/
*.db
*.xlsx

# Временные файлы
*.tmp
*.log

# IDE
.vscode/
.idea/
```

## 🐛 Устранение неполадок

### Частые проблемы

#### 1. Ошибка аутентификации IMAP
```
Ошибка подключения к почте: authentication failed
```
**Решение:** Проверьте OAuth токен в `config.yaml`

#### 2. Файл конфигурации не найден
```
Файл config.yaml не найден — используем значения по умолчанию
```
**Решение:** Создайте `config.yaml` в корне проекта

#### 3. Нет новых писем
```
Новых писем нет — выходим
```
**Решение:** Проверьте, есть ли новые письма в почтовом ящике

#### 4. Ошибка парсинга чеков
```
Пропуск — не чек или неизвестный шаблон
```
**Решение:** Проверьте, поддерживается ли формат чека

### Логи и отладка

Приложение выводит подробные логи:
```bash
./receiptAnalyzer -config config.yaml -import_saved_html 2>&1 | tee log.txt
```

## 📈 Производительность

### Оптимизации

1. **Индексы базы данных** создаются автоматически
2. **Пакетная обработка** писем для экономии памяти
3. **Отслеживание состояния** для избежания повторной обработки
4. **Асинхронная загрузка** писем через IMAP

### Рекомендации по производительности

1. **Ограничивайте количество писем** при первом запуске
2. **Используйте SSD** для базы данных
3. **Регулярно архивируйте** старые данные
4. **Мониторьте размер** базы данных

## 🤝 Вклад в проект

### Как внести вклад

1. **Форкните репозиторий**
2. **Создайте ветку** для новой функции
3. **Напишите код** с тестами
4. **Создайте Pull Request**

### Стандарты кода

- **Go fmt** для форматирования
- **Go vet** для проверки кода
- **Тесты** для новой функциональности
- **Документация** для публичных функций

## 📄 Лицензия

Этот проект распространяется под лицензией MIT. См. файл `LICENSE` для подробностей.

## 📞 Поддержка

### Способы связи

- **Issues** на GitHub для багов и предложений
- **Discussions** для общих вопросов
- **Wiki** для дополнительной документации

### Полезные ссылки

- [Go Documentation](https://golang.org/doc/)
- [SQLite Documentation](https://www.sqlite.org/docs.html)
- [Excelize Documentation](https://xuri.me/excelize/)

---

**ReceiptAnalyzer** — ваш надежный помощник в анализе покупок! 🛒📊 