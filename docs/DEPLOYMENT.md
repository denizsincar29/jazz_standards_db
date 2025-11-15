# Deployment Guide

## Interactive Setup

The easiest way to configure the application is using the interactive setup script:

```bash
./build.sh
```

This will:
1. Create `.env` file with your configuration
2. Ask for external port, database credentials, JWT secret
3. Ask for base path (for subpath deployments)
4. Optionally create a systemd service
5. Generate Apache proxy configuration in `proxy.txt`

The script automatically detects:
- Current directory for installation
- Current user and group for systemd service
- Generates secure JWT secret if not provided

### Manual Configuration

If you prefer manual setup, create a `.env` file:

```env
# External port
EXTERNAL_PORT=8000

# Database
DB_USER=jazz
DB_PASSWORD=jazz
DB_NAME=jazz

# Base path (empty for root, /jazzdb for subpath)
BASE_PATH=

# JWT secret (generate with: openssl rand -base64 32)
JWT_SECRET=your-secret-here

# Environment
ENVIRONMENT=production
```

## Apache Reverse Proxy Configuration

### Setup for Root Path

If you want to access the app at `http://yourdomain.com/`:

```apache
<VirtualHost *:80>
    ServerName yourdomain.com

    ProxyPreserveHost On
    ProxyPass / http://localhost:8000/
    ProxyPassReverse / http://localhost:8000/

    # WebSocket support (if needed in future)
    RewriteEngine on
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/?(.*) "ws://localhost:8000/$1" [P,L]

    # Optional: Enable compression
    <IfModule mod_deflate.c>
        AddOutputFilterByType DEFLATE text/html text/plain text/xml text/css text/javascript application/javascript application/json
    </IfModule>
</VirtualHost>
```

### Setup for Subpath (e.g., /jazzdb)

If you want to access the app at `http://yourdomain.com/jazzdb/`:

**1. Apache Configuration:**

```apache
<VirtualHost *:80>
    ServerName yourdomain.com

    # Serve static files directly (optional, for better performance)
    Alias /jazzdb/static /path/to/jazz_standards_db/static
    <Directory /path/to/jazz_standards_db/static>
        Require all granted
    </Directory>

    # Proxy API and app requests
    ProxyPreserveHost On
    ProxyPass /jazzdb/static !
    ProxyPass /jazzdb http://localhost:8000/
    ProxyPassReverse /jazzdb http://localhost:8000/

    # WebSocket support (if needed)
    RewriteEngine on
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/jazzdb/?(.*) "ws://localhost:8000/$1" [P,L]
</VirtualHost>
```

**2. Application Configuration:**

Set the base path via environment variable in `.env`:

```env
BASE_PATH=/jazzdb
```

Or in `docker-compose.yml`:

```yaml
services:
  app:
    environment:
      - BASE_PATH=/jazzdb
```

The application will automatically handle the base path routing. No code changes needed.

**3. Update PWA manifest paths:**

In `static/manifest.json`:
```json
{
  "start_url": "/jazzdb/",
  "scope": "/jazzdb/"
}
```

In `static/sw.js`, update cache paths:
```javascript
const ASSETS_TO_CACHE = [
    '/jazzdb/',
    '/jazzdb/static/css/styles.css',
    // ... etc
];
```

In `static/js/api.js`, update base URL:
```javascript
const API = {
    baseURL: '/jazzdb/api',
    // ... rest
};
```

### HTTPS Configuration (Recommended for Production)

```apache
<VirtualHost *:443>
    ServerName yourdomain.com

    SSLEngine on
    SSLCertificateFile /path/to/cert.pem
    SSLCertificateKeyFile /path/to/key.pem

    ProxyPreserveHost On
    ProxyPass / http://localhost:8000/
    ProxyPassReverse / http://localhost:8000/

    # Security headers
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "DENY"
    Header always set X-XSS-Protection "1; mode=block"
</VirtualHost>

# Redirect HTTP to HTTPS
<VirtualHost *:80>
    ServerName yourdomain.com
    Redirect permanent / https://yourdomain.com/
</VirtualHost>
```

## Enable Required Apache Modules

```bash
sudo a2enmod proxy
sudo a2enmod proxy_http
sudo a2enmod rewrite
sudo a2enmod headers
sudo a2enmod ssl  # for HTTPS
sudo systemctl restart apache2
```

## Testing Your Configuration

1. **Test Apache configuration:**
   ```bash
   sudo apache2ctl configtest
   ```

2. **Restart Apache:**
   ```bash
   sudo systemctl restart apache2
   ```

3. **Verify the app is accessible:**
   ```bash
   curl http://yourdomain.com/
   # or for subpath:
   curl http://yourdomain.com/jazzdb/
   ```

## Troubleshooting

### Port Already in Use
```bash
# Check what's using the port
sudo lsof -i :8000
# or
sudo netstat -tlnp | grep 8000
```

### Apache Proxy Not Working
- Check Apache error logs: `sudo tail -f /var/log/apache2/error.log`
- Verify app is running: `docker-compose ps`
- Test direct access: `curl http://localhost:8000/`

### PWA Not Loading After Proxy Setup
- Clear browser cache and service worker
- Check browser console for CORS or mixed content errors
- Verify all asset paths are correct in the network tab

## Production Checklist

- [ ] Use HTTPS with valid SSL certificate
- [ ] Set strong `JWT_SECRET` in environment
- [ ] Configure firewall to block direct access to port 8000
- [ ] Set up database backups
- [ ] Configure log rotation
- [ ] Monitor resource usage
- [ ] Set up uptime monitoring
- [ ] Document your specific configuration in `changes.md`
