#!/bin/bash

# Jazz Standards DB - Interactive Setup Script for Docker Deployment
# This script helps you configure the Jazz Standards Database for Docker Compose

set -e

echo "========================================="
echo "Jazz Standards DB - Docker Setup"
echo "========================================="
echo ""
echo "This script configures for Docker deployment"
echo "For native deployment (systemd), use build.sh instead"
echo ""

# Get current directory
INSTALL_DIR=$(pwd)

echo "Installation directory: $INSTALL_DIR"
echo ""

# Ask for external port
read -p "Enter external port for the application (host port) [8000]: " EXTERNAL_PORT
EXTERNAL_PORT=${EXTERNAL_PORT:-8000}

# Ask for database external port
read -p "Enter external port for PostgreSQL (host port) [5432]: " DB_EXTERNAL_PORT
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
DB_HOST=db
DB_PORT=5432
DB_USER=$DB_USER
DB_PASSWORD=$DB_PASSWORD
DB_NAME=$DB_NAME

# Server Configuration (internal container port)
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
    # Apache strips the base path before forwarding to the app
    cat >> $PROXY_FILE <<EOF
# Subpath deployment (http://yourdomain.com$BASE_PATH/)
# Apache strips "$BASE_PATH" from the URL before forwarding to the app

<VirtualHost *:80>
    ServerName yourdomain.com

    # IMPORTANT: ProxyPass with trailing slash strips the base path
    # Example: yourdomain.com$BASE_PATH/api/users -> localhost:$EXTERNAL_PORT/api/users
    
    ProxyPreserveHost On
    
    # Redirect /jazz to /jazz/ (required for proper path handling)
    RewriteEngine on
    RewriteRule ^$BASE_PATH\$ $BASE_PATH/ [R=301,L]
    
    # Strip base path and forward to app at root
    ProxyPass $BASE_PATH/ http://localhost:$EXTERNAL_PORT/
    ProxyPassReverse $BASE_PATH/ http://localhost:$EXTERNAL_PORT/

    # WebSocket support
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^$BASE_PATH/?(.*) "ws://localhost:$EXTERNAL_PORT/\$1" [P,L]

    # Security headers
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-XSS-Protection "1; mode=block"
</VirtualHost>

# Note: You've configured BASE_PATH=$BASE_PATH in .env
# The Go app doesn't need to know about this path - Apache handles the stripping.
# Access your app at: http://yourdomain.com$BASE_PATH/

# For HTTPS (recommended for production):
# <VirtualHost *:443>
#     ServerName yourdomain.com
#     
#     SSLEngine on
#     SSLCertificateFile /path/to/cert.pem
#     SSLCertificateKeyFile /path/to/key.pem
#     
#     ProxyPreserveHost On
#     
#     # Redirect /jazz to /jazz/
#     RewriteEngine on
#     RewriteRule ^$BASE_PATH\$ $BASE_PATH/ [R=301,L]
#     
#     ProxyPass $BASE_PATH/ http://localhost:$EXTERNAL_PORT/
#     ProxyPassReverse $BASE_PATH/ http://localhost:$EXTERNAL_PORT/
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
echo "  - External Port: $EXTERNAL_PORT (host) -> 8000 (container)"
echo "  - Database Port: $DB_EXTERNAL_PORT (host) -> 5432 (container)"
echo "  - Base Path: ${BASE_PATH:-/ (root)}"
echo "  - Environment: $ENVIRONMENT"
echo "  - Install Dir: $INSTALL_DIR"
echo ""
echo "Files created:"
echo "  ✓ .env - Environment configuration"
echo "  ✓ $PROXY_FILE - Apache configuration"
echo ""
echo "Next steps:"
echo ""
echo "1. Start with Docker Compose:"
echo "   docker-compose up -d"
echo ""
echo "2. Configure Apache (paste contents of $PROXY_FILE):"
echo "   sudo nano /etc/apache2/sites-available/yourdomain.conf"
echo "   sudo a2ensite yourdomain"
echo "   sudo systemctl restart apache2"
echo ""
echo "3. Create first admin user:"
echo "   curl -X POST http://localhost:$EXTERNAL_PORT/api/admin \\"
echo "     -H 'Content-Type: application/json' \\"
echo "     -d '{\"username\":\"admin\",\"name\":\"Admin\",\"password\":\"your-password\"}'"
echo ""
echo "========================================="
