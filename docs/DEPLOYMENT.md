# Deployment Guide

## Custom Port Configuration

The default `docker-compose.yml` uses port 8000. To use a different port without modifying the version-controlled file:

### Using .env File (Recommended)

1. **Copy the example file:**
   ```bash
   cp .env.example .env
   ```

2. **Edit .env to set your external port:**
   ```env
   EXTERNAL_PORT=5251  # Your desired port
   DB_EXTERNAL_PORT=5432  # PostgreSQL port (optional)
   ```

3. **Start the services:**
   ```bash
   docker-compose up -d
   ```

The `.env` file is gitignored, so your configuration persists across `git pull` and doesn't affect the repository.

**How it works:**
- Container internal port stays at 8000 (unchanged)
- `EXTERNAL_PORT` in `.env` controls what port on your host maps to container's 8000
- Example: `EXTERNAL_PORT=5251` means access at `http://localhost:5251`

### Alternative: Command Line

You can also override the port when starting:
```bash
EXTERNAL_PORT=5251 docker-compose up -d
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

For subpath deployment, you need to update the Go application to handle the base path. Add to `main.go`:

```go
// Add before router setup
basePath := os.Getenv("BASE_PATH")
if basePath == "" {
    basePath = "/"
}

// Then wrap your router
if basePath != "/" {
    mainRouter := mux.NewRouter()
    mainRouter.PathPrefix(basePath).Handler(http.StripPrefix(basePath, r))
    r = mainRouter
}
```

**3. Update docker-compose.yml or docker-compose.override.yml:**

```yaml
services:
  app:
    environment:
      - BASE_PATH=/jazzdb
```

**4. Update PWA manifest paths:**

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
