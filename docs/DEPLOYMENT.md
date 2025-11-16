# Deployment Guide

## Interactive Setup

The easiest way to configure the application is using the appropriate interactive setup script:

### For Docker Deployment

```bash
./build_docker.sh
```

This will:
1. Create `.env` file for Docker configuration
2. Ask for external port (host port mapping) - container always uses 8000 internally
3. Ask for database credentials, JWT secret
4. Ask for base path (for subpath deployments)
5. Generate Apache proxy configuration in `proxy.txt`

### For Native Deployment (systemd)

```bash
./build.sh
```

This will:
1. Create `.env` file for native deployment
2. Ask for server port (the port the Go application listens on)
3. Ask for database host, port, and credentials
4. Ask for base path (for subpath deployments)
5. Optionally create a systemd service
6. Generate Apache proxy configuration in `proxy.txt`

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

**Important:** The Go application always runs at root internally. Apache strips the base path before forwarding requests to the app. This is the standard reverse proxy pattern and avoids routing complexity.

**Apache Configuration:**

```apache
<VirtualHost *:80>
    ServerName yourdomain.com

    # IMPORTANT: The trailing slash after /jazzdb/ and :8000/ is required
    # This tells Apache to strip /jazzdb from the URL before forwarding
    # Example: yourdomain.com/jazzdb/api/users -> localhost:8000/api/users

    ProxyPreserveHost On
    
    # Redirect /jazzdb to /jazzdb/ (required for proper path handling)
    RewriteEngine on
    RewriteRule ^/jazzdb$ /jazzdb/ [R=301,L]
    
    # Strip /jazzdb and forward to app at root
    ProxyPass /jazzdb/ http://localhost:8000/
    ProxyPassReverse /jazzdb/ http://localhost:8000/

    # WebSocket support (if needed)
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/jazzdb/?(.*) "ws://localhost:8000/$1" [P,L]
</VirtualHost>
```

**Key Points:**

1. **Trailing slashes matter:** `ProxyPass /jazzdb/ http://localhost:8000/` strips `/jazzdb` from requests
2. **Redirect rule:** Ensures `/jazzdb` (without slash) redirects to `/jazzdb/` (with slash)
3. **No app changes needed:** The Go app doesn't need to know about `/jazzdb` at all
4. **BASE_PATH in .env:** Optional - not used by the app for routing, kept for reference only

**How it works:**

```
User visits:        yourdomain.com/jazzdb/
Apache forwards:    localhost:8000/

User visits:        yourdomain.com/jazzdb/api/users
Apache forwards:    localhost:8000/api/users

User visits:        yourdomain.com/jazzdb
Apache redirects:   yourdomain.com/jazzdb/ (301)
```

The frontend uses relative URLs, so it automatically works at any path.

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
