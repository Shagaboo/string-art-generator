#!/bin/bash

# Скрипт деплоя на сервер (ручной запуск с паролем)
SERVER="217.25.94.91"
USER="root"
PASSWORD="nFEo?VLSv4@1d#"
APP_DIR="/var/www/string-art-generator"
BACKEND_DIR="$APP_DIR/backend"
FRONTEND_DIR="$APP_DIR/frontend"

echo "🚀 Начинаем деплой на сервер $SERVER..."

# Проверяем наличие sshpass
if ! command -v sshpass &> /dev/null; then
    echo "⚠️  sshpass не установлен. Устанавливаем..."
    if [[ "$OSTYPE" == "darwin"* ]]; then
        brew install hudochenkov/sshpass/sshpass 2>/dev/null || echo "Установите sshpass: brew install hudochenkov/sshpass/sshpass"
    else
        echo "Установите sshpass: sudo apt-get install sshpass"
    fi
fi

# Функция для выполнения команд через SSH с паролем
ssh_cmd() {
    sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USER@$SERVER "$1"
}

# Функция для загрузки файлов через SCP с паролем
scp_cmd() {
    sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no "$@"
}

# Создаем директории на сервере
echo "📁 Создаем директории..."
ssh_cmd "mkdir -p $APP_DIR $BACKEND_DIR $FRONTEND_DIR /var/www/string-art-generator/deploy"

# Загружаем backend
echo "📦 Загружаем backend..."
scp_cmd backend/string-art-backend-linux $USER@$SERVER:$BACKEND_DIR/string-art-backend
scp_cmd backend/go.mod $USER@$SERVER:$BACKEND_DIR/ 2>/dev/null || true
scp_cmd backend/go.sum $USER@$SERVER:$BACKEND_DIR/ 2>/dev/null || true

# Загружаем frontend (production build) - используем tar для сжатия
echo "📦 Загружаем frontend (сжимаем и загружаем)..."
cd frontend
tar czf /tmp/frontend-next.tar.gz .next 2>/dev/null
tar czf /tmp/frontend-public.tar.gz public 2>/dev/null
cd ..
sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no /tmp/frontend-next.tar.gz $USER@$SERVER:/tmp/
sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no /tmp/frontend-public.tar.gz $USER@$SERVER:/tmp/
ssh_cmd "cd $FRONTEND_DIR && tar xzf /tmp/frontend-next.tar.gz && tar xzf /tmp/frontend-public.tar.gz && rm /tmp/frontend-*.tar.gz"
rm /tmp/frontend-*.tar.gz

scp_cmd frontend/package.json $USER@$SERVER:$FRONTEND_DIR/
scp_cmd frontend/next.config.mjs $USER@$SERVER:$FRONTEND_DIR/
scp_cmd frontend/tsconfig.json $USER@$SERVER:$FRONTEND_DIR/
echo "✅ Frontend загружен"

# Загружаем конфигурационные файлы
echo "📦 Загружаем конфигурацию..."
scp_cmd deploy/nginx.conf $USER@$SERVER:/etc/nginx/sites-available/string-art-generator
scp_cmd deploy/string-art-backend.service $USER@$SERVER:/etc/systemd/system/
scp_cmd deploy/string-art-frontend.service $USER@$SERVER:/etc/systemd/system/
scp_cmd deploy/start-frontend.sh $USER@$SERVER:/var/www/string-art-generator/deploy/

# Настраиваем на сервере
echo "⚙️  Настраиваем сервер..."
ssh_cmd "bash -s" << 'ENDSSH'
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
chmod +x /var/www/string-art-generator/deploy/start-frontend.sh

# Настраиваем nginx
rm -f /etc/nginx/sites-enabled/default
ln -sf /etc/nginx/sites-available/string-art-generator /etc/nginx/sites-enabled/
nginx -t && systemctl reload nginx

# Настраиваем systemd services
systemctl daemon-reload
systemctl enable string-art-backend
systemctl enable string-art-frontend
systemctl restart string-art-backend
systemctl restart string-art-frontend

echo "✅ Деплой завершен!"
systemctl status string-art-backend --no-pager
systemctl status string-art-frontend --no-pager
ENDSSH

echo "🎉 Готово! Приложение доступно на http://$SERVER"
echo "📊 Проверка статуса:"
ssh_cmd "systemctl status string-art-backend --no-pager -l"
ssh_cmd "systemctl status string-art-frontend --no-pager -l"

