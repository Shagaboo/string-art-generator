#!/bin/bash

# Скрипт деплоя на сервер
SERVER="217.25.94.91"
USER="root"
APP_DIR="/var/www/string-art-generator"
BACKEND_DIR="$APP_DIR/backend"
FRONTEND_DIR="$APP_DIR/frontend"

echo "🚀 Начинаем деплой на сервер $SERVER..."

# Создаем директории на сервере
ssh $USER@$SERVER "mkdir -p $APP_DIR $BACKEND_DIR $FRONTEND_DIR"

# Загружаем backend
echo "📦 Загружаем backend..."
scp backend/string-art-backend-linux $USER@$SERVER:$BACKEND_DIR/string-art-backend
scp backend/go.mod $USER@$SERVER:$BACKEND_DIR/
scp backend/go.sum $USER@$SERVER:$BACKEND_DIR/ 2>/dev/null || true

# Загружаем frontend (production build)
echo "📦 Загружаем frontend..."
scp -r frontend/.next $USER@$SERVER:$FRONTEND_DIR/
scp -r frontend/public $USER@$SERVER:$FRONTEND_DIR/
scp frontend/package.json $USER@$SERVER:$FRONTEND_DIR/
scp frontend/next.config.mjs $USER@$SERVER:$FRONTEND_DIR/
scp frontend/tsconfig.json $USER@$SERVER:$FRONTEND_DIR/

# Загружаем конфигурационные файлы
echo "📦 Загружаем конфигурацию..."
scp deploy/nginx.conf $USER@$SERVER:/etc/nginx/sites-available/string-art-generator
scp deploy/string-art-backend.service $USER@$SERVER:/etc/systemd/system/
scp deploy/string-art-frontend.service $USER@$SERVER:/etc/systemd/system/
scp deploy/start-frontend.sh $USER@$SERVER:/var/www/string-art-generator/deploy/

# Настраиваем на сервере
echo "⚙️  Настраиваем сервер..."
ssh $USER@$SERVER << 'ENDSSH'
# Устанавливаем Node.js и npm если нужно
if ! command -v node &> /dev/null; then
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt-get install -y nodejs
fi

# Устанавливаем nginx если нужно
if ! command -v nginx &> /dev/null; then
    apt-get update
    apt-get install -y nginx
fi

# Устанавливаем зависимости frontend
cd /var/www/string-art-generator/frontend
npm install --production

# Делаем backend исполняемым
chmod +x /var/www/string-art-generator/backend/string-art-backend

# Настраиваем nginx
ln -sf /etc/nginx/sites-available/string-art-generator /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx

# Настраиваем systemd services
chmod +x /var/www/string-art-generator/deploy/start-frontend.sh
systemctl daemon-reload
systemctl enable string-art-backend
systemctl enable string-art-frontend
systemctl restart string-art-backend
systemctl restart string-art-frontend

echo "✅ Деплой завершен!"
ENDSSH

echo "🎉 Готово! Приложение доступно на http://$SERVER"

