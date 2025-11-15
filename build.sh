#!/bin/bash

# Jazz Standards DB - Interactive Setup Script
# This script helps you configure and deploy the Jazz Standards Database

set -e

echo "========================================="
echo "Jazz Standards DB - Interactive Setup"
echo "========================================="
echo ""

# Get current directory
INSTALL_DIR=$(pwd)
CURRENT_USER=$(whoami)
CURRENT_GROUP=$(id -gn)

echo "Installation directory: $INSTALL_DIR"
echo "User: $CURRENT_USER"
echo "Group: $CURRENT_GROUP"
echo ""

# Ask for external port
read -p "Enter external port for the application [8000]: " EXTERNAL_PORT
EXTERNAL_PORT=${EXTERNAL_PORT:-8000}

# Ask for database external port
read -p "Enter external port for PostgreSQL [5432]: " DB_EXTERNAL_PORT
DB_EXTERNAL_PORT=${DB_EXTERNAL_PORT:-5432}

# Ask for base path
echo ""
echo "Base path configuration:"
echo "  - Leave empty for root path deployment (yourdomain.com/)"
echo "  - Enter /jazzdb for subpath deployment (yourdomain.com/jazzdb/)"
read -p "Enter base path [empty for root]: " BASE_PATH

# Ask for database credentials
echo ""
read -p "Enter database user [jazz]: " DB_USER
DB_USER=${DB_USER:-jazz}

read -sp "Enter database password [jazz]: " DB_PASSWORD
echo ""
DB_PASSWORD=${DB_PASSWORD:-jazz}

read -p "Enter database name [jazz]: " DB_NAME
DB_NAME=${DB_NAME:-jazz}

# Ask for JWT secret
echo ""
read -sp "Enter JWT secret (leave empty to generate random): " JWT_SECRET
echo ""
if [ -z "$JWT_SECRET" ]; then
    JWT_SECRET=$(openssl rand -base64 32 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | fold -w 32 | head -n 1)
    echo "Generated JWT secret: $JWT_SECRET"
fi

# Ask for environment
echo ""
read -p "Enter environment (development/production) [production]: " ENVIRONMENT
ENVIRONMENT=${ENVIRONMENT:-production}

# Create .env file
echo ""
echo "Creating .env file..."
cat > .env <<EOF
# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME

# Server Configuration
PORT=8000

# External Ports (for docker-compose)
EXTERNAL_PORT=$EXTERNAL_PORT
DB_EXTERNAL_PORT=$DB_EXTERNAL_PORT

# Base Path (for reverse proxy subpath deployment)
BASE_PATH=$BASE_PATH

# Security
JWT_SECRET=$JWT_SECRET

# Environment
ENVIRONMENT=$ENVIRONMENT
EOF

echo "✓ Created .env file"

# Create systemd service
echo ""
read -p "Create systemd service? (y/n) [y]: " CREATE_SERVICE
CREATE_SERVICE=${CREATE_SERVICE:-y}

if [ "$CREATE_SERVICE" = "y" ] || [ "$CREATE_SERVICE" = "Y" ]; then
    SERVICE_NAME="jazz-standards-db"
    
    echo "Creating systemd service: $SERVICE_NAME.service"
    
    sudo tee /etc/systemd/system/$SERVICE_NAME.service > /dev/null <<EOF
[Unit]
Description=Jazz Standards Database
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=$CURRENT_USER
Group=$CURRENT_GROUP
WorkingDirectory=$INSTALL_DIR
Environment="PATH=/usr/local/bin:/usr/bin:/bin"
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=$INSTALL_DIR/jazz_standards_db
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

    echo "✓ Created systemd service"
    echo ""
    echo "To enable and start the service:"
    echo "  sudo systemctl daemon-reload"
    echo "  sudo systemctl enable $SERVICE_NAME"
    echo "  sudo systemctl start $SERVICE_NAME"
    echo "  sudo systemctl status $SERVICE_NAME"
fi

# Create Apache proxy configuration
echo ""
echo "Creating Apache proxy configuration..."

PROXY_FILE="proxy.txt"
cat > $PROXY_FILE <<EOF
# Apache Reverse Proxy Configuration for Jazz Standards DB
# Add this to your Apache VirtualHost configuration

EOF

if [ -z "$BASE_PATH" ]; then
    # Root path configuration
    cat >> $PROXY_FILE <<EOF
# Root path deployment (http://yourdomain.com/)

<VirtualHost *:80>
    ServerName yourdomain.com

    ProxyPreserveHost On
    ProxyPass / http://localhost:$EXTERNAL_PORT/
    ProxyPassReverse / http://localhost:$EXTERNAL_PORT/

    # WebSocket support
    RewriteEngine on
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/?(.*) "ws://localhost:$EXTERNAL_PORT/\$1" [P,L]

    # Optional: Enable compression
    <IfModule mod_deflate.c>
        AddOutputFilterByType DEFLATE text/html text/plain text/xml text/css text/javascript application/javascript application/json
    </IfModule>

    # Security headers
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-XSS-Protection "1; mode=block"
</VirtualHost>

# For HTTPS (recommended for production):
# <VirtualHost *:443>
#     ServerName yourdomain.com
#     
#     SSLEngine on
#     SSLCertificateFile /path/to/cert.pem
#     SSLCertificateKeyFile /path/to/key.pem
#     
#     ProxyPreserveHost On
#     ProxyPass / http://localhost:$EXTERNAL_PORT/
#     ProxyPassReverse / http://localhost:$EXTERNAL_PORT/
#     
#     # Add security headers
#     Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
# </VirtualHost>

# Enable required modules:
# sudo a2enmod proxy proxy_http rewrite headers ssl
# sudo systemctl restart apache2
EOF
else
    # Subpath configuration
    cat >> $PROXY_FILE <<EOF
# Subpath deployment (http://yourdomain.com$BASE_PATH/)

<VirtualHost *:80>
    ServerName yourdomain.com

    # Proxy to application
    ProxyPreserveHost On
    ProxyPass $BASE_PATH http://localhost:$EXTERNAL_PORT/
    ProxyPassReverse $BASE_PATH http://localhost:$EXTERNAL_PORT/

    # WebSocket support
    RewriteEngine on
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^$BASE_PATH/?(.*) "ws://localhost:$EXTERNAL_PORT/\$1" [P,L]

    # Security headers
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-XSS-Protection "1; mode=block"
</VirtualHost>

# Note: You've configured BASE_PATH=$BASE_PATH
# Make sure this matches in your .env file

# For HTTPS (recommended for production):
# <VirtualHost *:443>
#     ServerName yourdomain.com
#     
#     SSLEngine on
#     SSLCertificateFile /path/to/cert.pem
#     SSLCertificateKeyFile /path/to/key.pem
#     
#     ProxyPreserveHost On
#     ProxyPass $BASE_PATH http://localhost:$EXTERNAL_PORT/
#     ProxyPassReverse $BASE_PATH http://localhost:$EXTERNAL_PORT/
#     
#     # Add security headers
#     Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
# </VirtualHost>

# Enable required modules:
# sudo a2enmod proxy proxy_http rewrite headers ssl
# sudo systemctl restart apache2
EOF
fi

echo "✓ Created $PROXY_FILE"

# Summary
echo ""
echo "========================================="
echo "Setup Complete!"
echo "========================================="
echo ""
echo "Configuration Summary:"
echo "  - External Port: $EXTERNAL_PORT"
echo "  - Database Port: $DB_EXTERNAL_PORT"
echo "  - Base Path: ${BASE_PATH:-/ (root)}"
echo "  - Environment: $ENVIRONMENT"
echo "  - Install Dir: $INSTALL_DIR"
echo ""
echo "Files created:"
echo "  ✓ .env - Environment configuration"
if [ "$CREATE_SERVICE" = "y" ] || [ "$CREATE_SERVICE" = "Y" ]; then
    echo "  ✓ /etc/systemd/system/$SERVICE_NAME.service - Systemd service"
fi
echo "  ✓ $PROXY_FILE - Apache configuration"
echo ""
echo "Next steps:"
echo ""
echo "1. Build the application:"
echo "   go build -o jazz_standards_db main.go"
echo ""
echo "2. Or use Docker Compose:"
echo "   docker-compose up -d"
echo ""
if [ "$CREATE_SERVICE" = "y" ] || [ "$CREATE_SERVICE" = "Y" ]; then
    echo "3. Start the systemd service:"
    echo "   sudo systemctl daemon-reload"
    echo "   sudo systemctl enable $SERVICE_NAME"
    echo "   sudo systemctl start $SERVICE_NAME"
    echo ""
fi
echo "4. Configure Apache (paste contents of $PROXY_FILE):"
echo "   sudo nano /etc/apache2/sites-available/yourdomain.conf"
echo "   sudo a2ensite yourdomain"
echo "   sudo systemctl restart apache2"
echo ""
echo "5. Create first admin user:"
echo "   curl -X POST http://localhost:$EXTERNAL_PORT/api/admin \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"username\":\"admin\",\"name\":\"Admin\",\"password\":\"your-password\"}'"
echo ""
echo "========================================="
