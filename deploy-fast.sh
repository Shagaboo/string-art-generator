#!/bin/bash

# Быстрый деплой с прогрессом
SERVER="217.25.94.91"
USER="root"
PASSWORD="oNPQx1NFTu3h*V"
APP_DIR="/var/www/string-art-generator"

echo "🚀 Быстрый деплой на $SERVER..."

# Функции
ssh_cmd() {
    sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no -o ConnectTimeout=10 $USER@$SERVER "$1"
}

# Этап 1: Подготовка сервера
echo "📁 [1/6] Создаем директории..."
ssh_cmd "mkdir -p $APP_DIR/{backend,frontend,deploy}" || exit 1

# Этап 2: Backend
echo "📦 [2/6] Загружаем backend..."
sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no -o ConnectTimeout=10 \
    backend/string-art-backend-linux $USER@$SERVER:$APP_DIR/backend/string-art-backend || exit 1

# Этап 3: Frontend - только необходимые файлы
echo "📦 [3/6] Загружаем frontend (сжатие)..."
cd frontend
tar czf /tmp/next.tar.gz .next 2>/dev/null
tar czf /tmp/public.tar.gz public 2>/dev/null
cd ..

echo "📤 [4/6] Отправляем frontend на сервер..."
sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no -o ConnectTimeout=30 \
    /tmp/next.tar.gz /tmp/public.tar.gz \
    frontend/package.json frontend/next.config.mjs frontend/tsconfig.json \
    $USER@$SERVER:/tmp/ || exit 1

# Этап 4: Распаковка на сервере
echo "📦 [5/6] Распаковываем на сервере..."
ssh_cmd "cd $APP_DIR/frontend && \
    tar xzf /tmp/next.tar.gz && \
    tar xzf /tmp/public.tar.gz && \
    mv /tmp/package.json /tmp/next.config.mjs /tmp/tsconfig.json . && \
    rm /tmp/*.tar.gz" || exit 1

# Этап 5: Конфигурация
echo "⚙️  [6/6] Настраиваем сервер..."
sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no \
    deploy/nginx.conf deploy/string-art-backend.service \
    deploy/string-art-frontend.service deploy/start-frontend.sh \
    $USER@$SERVER:/tmp/ || exit 1

ssh_cmd "bash" << 'ENDSSH'
# Установка зависимостей
if ! command -v node &> /dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash - >/dev/null 2>&1
    apt-get install -y nodejs >/dev/null 2>&1
fi

if ! command -v nginx &> /dev/null; then
    apt-get update >/dev/null 2>&1
    apt-get install -y nginx >/dev/null 2>&1
fi

# Frontend зависимости
cd /var/www/string-art-generator/frontend
npm install --production --silent >/dev/null 2>&1

# Права
chmod +x /var/www/string-art-generator/backend/string-art-backend
chmod +x /tmp/start-frontend.sh
mv /tmp/start-frontend.sh /var/www/string-art-generator/deploy/

# Nginx
mv /tmp/nginx.conf /etc/nginx/sites-available/string-art-generator
rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/string-art-generator /etc/nginx/sites-enabled/
nginx -t >/dev/null 2>&1 && systemctl reload nginx

# Systemd
mv /tmp/string-art-backend.service /etc/systemd/system/
mv /tmp/string-art-frontend.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable string-art-backend string-art-frontend
systemctl restart string-art-backend string-art-frontend

echo "✅ Готово!"
ENDSSH

# Очистка
rm -f /tmp/next.tar.gz /tmp/public.tar.gz

echo ""
echo "🎉 Деплой завершен!"
echo "🌐 Приложение: http://$SERVER"
echo ""
echo "Проверка статуса:"
ssh_cmd "systemctl is-active string-art-backend string-art-frontend"

