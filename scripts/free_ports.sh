#!/bin/bash

# Скрипт для освобождения портов, указанных в config.yaml
# Использование: ./free_ports.sh [config_file]

set -e

# Определяем путь к конфигурационному файлу
CONFIG_FILE="${1:-config.yaml}"

# Проверяем существование файла
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Ошибка: Файл конфигурации '$CONFIG_FILE' не найден"
    echo "Использование: $0 [config_file]"
    exit 1
fi

echo "🔍 Анализируем порты в файле: $CONFIG_FILE"

# Функция для освобождения порта
free_port() {
    local port=$1
    local service_name=$2
    
    echo "🔌 Проверяем порт $port ($service_name)..."
    
    # Ищем процессы, использующие порт
    local pids=$(lsof -ti:$port 2>/dev/null || true)
    
    if [ -z "$pids" ]; then
        echo "   ✅ Порт $port свободен"
        return 0
    fi
    
    echo "   ⚠️  Порт $port занят процессами: $pids"
    
    # Показываем информацию о процессах
    echo "   📋 Информация о процессах:"
    lsof -i:$port 2>/dev/null || echo "   Не удалось получить информацию"
    
    # Спрашиваем подтверждение на завершение
    read -p "   ❓ Завершить процессы на порту $port? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "   🗑️  Завершаем процессы..."
        kill -TERM $pids 2>/dev/null || true
        
        # Ждем немного и проверяем
        sleep 2
        
        # Если процессы все еще живы, принудительно завершаем
        local remaining_pids=$(lsof -ti:$port 2>/dev/null || true)
        if [ -n "$remaining_pids" ]; then
            echo "   💀 Принудительно завершаем процессы..."
            kill -KILL $remaining_pids 2>/dev/null || true
        fi
        
        # Финальная проверка
        sleep 1
        local final_check=$(lsof -ti:$port 2>/dev/null || true)
        if [ -z "$final_check" ]; then
            echo "   ✅ Порт $port успешно освобожден"
        else
            echo "   ❌ Не удалось освободить порт $port"
            return 1
        fi
    else
        echo "   ⏭️  Пропускаем порт $port"
    fi
}

# Извлекаем порты из YAML файла
echo "📖 Извлекаем порты из конфигурации..."

# Используем grep и sed для извлечения портов
PORTS=$(grep -E "^[[:space:]]*[a-zA-Z_]+:[[:space:]]*[0-9]+" "$CONFIG_FILE" | \
        grep -E "(mailfetcher|htmlimporter|xlsxexporter|qwencategorizer)" | \
        sed 's/.*:[[:space:]]*\([0-9]*\).*/\1/')

if [ -z "$PORTS" ]; then
    echo "❌ Не найдены порты в секции 'ports' файла $CONFIG_FILE"
    exit 1
fi

echo "🎯 Найдены порты: $PORTS"

# Создаем ассоциативный массив порт -> сервис
declare -A PORT_SERVICES

# Парсим порты и их сервисы
while IFS= read -r line; do
    if [[ $line =~ ^[[:space:]]*([a-zA-Z_]+):[[:space:]]*([0-9]+) ]]; then
        service="${BASH_REMATCH[1]}"
        port="${BASH_REMATCH[2]}"
        if [[ $service =~ (mailfetcher|htmlimporter|xlsxexporter|qwencategorizer) ]]; then
            PORT_SERVICES["$port"]="$service"
        fi
    fi
done < "$CONFIG_FILE"

# Освобождаем каждый порт
echo ""
echo "🚀 Начинаем освобождение портов..."
echo ""

for port in "${!PORT_SERVICES[@]}"; do
    service="${PORT_SERVICES[$port]}"
    free_port "$port" "$service"
    echo ""
done

echo "✅ Завершено освобождение портов"
echo ""
echo "📊 Статус портов:"
for port in "${!PORT_SERVICES[@]}"; do
    service="${PORT_SERVICES[$port]}"
    if lsof -ti:$port >/dev/null 2>&1; then
        echo "   ❌ Порт $port ($service) - занят"
    else
        echo "   ✅ Порт $port ($service) - свободен"
    fi
done 