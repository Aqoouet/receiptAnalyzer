#!/bin/bash

# Скрипт для быстрого запуска ReceiptAnalyzer
# Использование: ./run.sh [команда] [опции]

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Функция для вывода сообщений
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Проверяем, что мы в правильной директории
if [ ! -f "go.mod" ]; then
    print_error "Файл go.mod не найден. Запустите скрипт из корня проекта."
    exit 1
fi

# Проверяем наличие исполняемого файла
if [ ! -f "receiptAnalyzer" ]; then
    print_info "Собираем приложение..."
    go build -o receiptAnalyzer ./cmd
    if [ $? -eq 0 ]; then
        print_success "Приложение собрано успешно"
    else
        print_error "Ошибка сборки приложения"
        exit 1
    fi
fi

# Проверяем наличие конфигурации
if [ ! -f "config.yaml" ]; then
    print_warning "Файл config.yaml не найден"
    if [ -f "config.yaml.example" ]; then
        print_info "Копируем пример конфигурации..."
        cp config.yaml.example config.yaml
        print_warning "Отредактируйте config.yaml перед использованием"
    else
        print_error "Файл config.yaml.example не найден"
        exit 1
    fi
fi

# Функция показа справки
show_help() {
    echo "ReceiptAnalyzer - Анализатор чеков"
    echo ""
    echo "Использование: $0 [команда] [опции]"
    echo ""
    echo "Команды:"
    echo "  fetch [N]     - Скачать новые письма (N - количество, по умолчанию все)"
    echo "  import        - Импортировать HTML в базу данных"
    echo "  rebuild       - Пересоздать базу и импортировать"
    echo "  help          - Показать эту справку"
    echo ""
    echo "Примеры:"
    echo "  $0 fetch      - Скачать все новые письма"
    echo "  $0 fetch 10   - Скачать 10 новых писем"
    echo "  $0 import     - Импортировать сохраненные HTML"
    echo "  $0 rebuild    - Пересоздать базу и импортировать"
}

# Обработка команд
case "${1:-help}" in
    "fetch")
        if [ -n "$2" ]; then
            print_info "Скачиваем $2 новых писем..."
            ./receiptAnalyzer -config config.yaml -save_new_emails -quantityToProcess "$2"
        else
            print_info "Скачиваем все новые письма..."
            ./receiptAnalyzer -config config.yaml -save_new_emails
        fi
        ;;
    "import")
        print_info "Импортируем HTML в базу данных..."
        ./receiptAnalyzer -config config.yaml -import_saved_html
        ;;
    "rebuild")
        print_warning "Пересоздаем базу данных..."
        ./receiptAnalyzer -config config.yaml -import_saved_html -rebuild_db
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    *)
        print_error "Неизвестная команда: $1"
        echo ""
        show_help
        exit 1
        ;;
esac

print_success "Операция завершена" 