# Jazz Standards Database - Go PWA

A Progressive Web App (PWA) for tracking and managing your jazz standards repertoire, built with Go and standard library HTTP.

## Features

- **Progressive Web App**: Install on mobile/desktop, works offline
- **User Management**: Secure authentication with bcrypt and tokens
- **Common Standards Database**: Shared database of jazz standards
- **User Submissions**: All authenticated users can submit new standards
- **Approval Workflow**: Admin must approve user-submitted standards before they appear in the database
- **Personal Tracking**: Track which standards you know
- **Categories**: Organize your standards into custom categories
- **Search & Filter**: Find standards by title, composer, or style
- **Admin Panel**: Approve/reject/delete standards, view pending submissions
- **Responsive Design**: Works on all devices

## Technology Stack

- **Backend**: Go 1.21+ with standard library HTTP + Gorilla Mux
- **Database**: PostgreSQL 15+ with GORM ORM
- **Frontend**: Vanilla JavaScript PWA (no frameworks)
- **Authentication**: Token-based with bcrypt password hashing
- **Deployment**: Docker & Docker Compose

## Quick Start

### Option 1: Docker Deployment (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/denizsincar29/jazz_standards_db.git
   cd jazz_standards_db
   ```

2. **Run the Docker setup script**
   ```bash
   ./build_docker.sh
   ```
   
   This will:
   - Create `.env` configuration file for Docker
   - Ask for external port (host port mapping)
   - Ask for base path and database credentials
   - Generate Apache proxy configuration in `proxy.txt`

3. **Start with Docker Compose**
   ```bash
   docker-compose up --build
   ```

4. **Create the first admin account**
   
   Build and run the admin creation tool:
   ```bash
   go build -o create_admin cmd/create_admin/main.go
   ./create_admin
   ```
   
   The tool will:
   - Read database configuration from `.env` or environment variables
   - Prompt for username, display name, and password interactively
   - Create or update the admin user in the database
   - Passwords are hidden during input for security

5. **Access the application**
   - Open http://localhost:8000 in your browser (or your configured external port)
   - Login with your admin credentials
   - Start adding and approving jazz standards!

### Option 2: Native Deployment (systemd)

1. **Clone and run native setup**
   ```bash
   git clone https://github.com/denizsincar29/jazz_standards_db.git
   cd jazz_standards_db
   ./build.sh
   ```
   
   This will:
   - Create `.env` configuration file for native deployment
   - Ask for server port (the port the Go app listens on)
   - Ask for database host, port, and credentials
   - Optionally create a systemd service
   - Generate Apache proxy configuration in `proxy.txt`

2. **Build and start the application**
   ```bash
   go build -o jazz_standards_db main.go
   sudo systemctl daemon-reload
   sudo systemctl enable jazz-standards-db
   sudo systemctl start jazz-standards-db
   ```

3. **Access the application at the port you configured**

### Local Development

#### Prerequisites
- Go 1.21 or higher
- PostgreSQL 15 or higher

#### Setup

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Setup PostgreSQL**
   ```bash
   # Create database
   createdb jazz
   
   # Or use psql
   psql -U postgres -c "CREATE DATABASE jazz;"
   ```

3. **Configure environment**
   ```bash
   ./build.sh  # Interactive setup for native deployment
   # Or manually create .env file with your settings
   ```

4. **Run the application**
   ```bash
   go run main.go
   ```

5. **Access the application**
   - Open http://localhost:8000

## API Documentation

### Authentication

#### POST `/api/register`
Create a new user account (non-admin)
```json
{
  "username": "jazzfan",
  "name": "Jazz Fan",
  "password": "securepassword"
}
```

#### POST `/api/login`
Login and receive authentication token
```json
{
  "username": "jazzfan",
  "password": "securepassword"
}
```

### Jazz Standards

#### GET `/api/jazz_standards`
List all jazz standards with pagination and filters
- Regular users: Only see approved standards
- Admins: Can filter by status (pending, approved, rejected)
```
Query params:
- page (default: 1)
- limit (default: 100)
- search (optional)
- style (optional)
- status (optional, admin only): pending, approved, rejected
```

#### POST `/api/jazz_standards` (All authenticated users)
Submit a new jazz standard
- **Regular users**: Standard is created with status="pending", requires admin approval
- **Admins**: Standard is auto-approved with status="approved"
```json
{
  "title": "All the Things You Are",
  "composer": "Jerome Kern",
  "style": "swing",
  "additional_note": "From 'Very Warm for May'"
}
```

#### GET `/api/jazz_standards/pending` (Admin only)
List all pending standards awaiting approval

#### POST `/api/jazz_standards/:id/approve` (Admin only)
Approve a pending standard

#### POST `/api/jazz_standards/:id/reject` (Admin only)
Reject a pending standard

#### PUT `/api/jazz_standards/:id` (Admin only)
Update a jazz standard

#### DELETE `/api/jazz_standards/:id` (Admin only)
Delete a jazz standard

### User Standards

#### GET `/api/users/me/standards`
Get all standards the user knows

#### POST `/api/users/me/standards/:standard_id`
Add a standard to your known list
```json
{
  "category_id": 1,
  "notes": "Working on this in F major"
}
```

#### PUT `/api/users/me/standards/:standard_id`
Update category or notes for a standard

#### DELETE `/api/users/me/standards/:standard_id`
Remove a standard from your known list

### Categories

#### GET `/api/users/me/categories`
List user's categories

#### POST `/api/users/me/categories`
Create a new category
```json
{
  "name": "Working On",
  "color": "#4a90e2"
}
```

#### PUT `/api/users/me/categories/:id`
Update a category

#### DELETE `/api/users/me/categories/:id`
Delete a category

## Jazz Styles

Supported jazz styles:
- `swing`
- `bebop`
- `bossa_nova`
- `latin`
- `modal`
- `waltz`
- `fusion`
- `dixieland`
- `ragtime`
- `big_band`
- `samba`
- `latin_swing`
- `free`

## Importing Jazz Standards

### Option 1: Create Import Script

Create a file `standards.json`:
```json
[
  {
    "title": "Autumn Leaves",
    "composer": "Joseph Kosma",
    "style": "swing"
  },
  {
    "title": "All the Things You Are",
    "composer": "Jerome Kern",
    "style": "swing"
  }
]
```

Then use the import script:
```bash
go run scripts/import_standards.go standards.json
```

### Option 2: Use API Directly

```bash
# Set your admin token
TOKEN="your-admin-token"

# Add standards via API
curl -X POST http://localhost:8000/api/jazz_standards \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Blue Bossa","composer":"Kenny Dorham","style":"bossa_nova"}'
```

### Data Sources

**Recommended sources for jazz standards data:**

1. **Wikipedia**: "List of jazz standards" - public domain information
2. **Real Book**: Public domain standards only
3. **Jazz Standards Index**: Compile from various public sources
4. **Community Contributions**: User-sourced standards

**Important**: Respect copyright laws. Only include titles, composers, and basic metadata. Do not include copyrighted chord charts or lyrics.

## PWA Features

### Installation
- On mobile: Tap "Add to Home Screen" when prompted
- On desktop: Look for the install icon in the address bar

### Offline Support
- Service worker caches static assets
- Works offline after first visit
- API requests cached for offline viewing

### Performance
- Fast load times with caching
- Responsive design for all screen sizes
- Optimized for 2000+ standards

## Project Structure

```
/
├── main.go              # Application entry point
├── go.mod               # Go module dependencies
├── config/              # Configuration management
├── models/              # Database models (User, JazzStandard, etc.)
├── database/            # Database connection and migrations
├── handlers/            # HTTP request handlers
├── middleware/          # Authentication middleware
├── utils/               # Helper functions
├── static/              # PWA frontend
│   ├── index.html
│   ├── manifest.json
│   ├── sw.js           # Service worker
│   ├── css/
│   ├── js/
│   └── icons/
└── scripts/            # Import scripts

```

## Development

### Build
```bash
go build -o jazz_standards_db
```

### Run Tests
```bash
go test ./...
```

### Docker Build
```bash
docker build -t jazz-standards-db .
docker run -p 8000:8000 \
  -e DB_HOST=host.docker.internal \
  -e DB_PASSWORD=jazz \
  jazz-standards-db
```

## Environment Variables

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=jazz
DB_PASSWORD=jazz
DB_NAME=jazz

# Server
PORT=8000

# Security
JWT_SECRET=change-me-in-production

# Environment
ENVIRONMENT=development
```

## Deployment

### Configuration

Use the appropriate interactive setup script:

**For Docker deployment:**
```bash
./build_docker.sh
```

**For native deployment (systemd):**
```bash
./build.sh
```

Both scripts guide you through:
- Port configuration (external port for Docker, server port for native)
- Base path for subpath deployments
- Database credentials
- JWT secret generation (auto-generated if not provided)
- Apache proxy configuration

Native deployment also offers:
- Systemd service creation
- Auto-detects user, group, and working directory

The `.env` file is gitignored and won't be affected by `git pull`.

### Apache Reverse Proxy

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for detailed Apache configuration including:
- Root path setup
- Subpath setup (e.g., /jazzdb)
- HTTPS configuration
- WebSocket support

Quick example for subpath deployment:

1. **Apache config:**
   ```apache
   ProxyPass /jazzdb http://localhost:8000/
   ProxyPassReverse /jazzdb http://localhost:8000/
   ```

2. **Configure Apache (Important - read the comments!):**
   ```apache
   # The trailing slash after /jazzdb/ is REQUIRED
   # This tells Apache to strip /jazzdb from URLs before forwarding
   
   RewriteEngine on
   RewriteRule ^/jazzdb$ /jazzdb/ [R=301,L]
   
   ProxyPass /jazzdb/ http://localhost:8000/
   ProxyPassReverse /jazzdb/ http://localhost:8000/
   ```

The Go application doesn't need to know about the subpath - Apache strips it automatically. See `docs/DEPLOYMENT.md` for complete configuration examples.

### Production Checklist

- [ ] Use HTTPS with valid SSL certificate
- [ ] Set strong JWT_SECRET
- [ ] Configure firewall to block direct port access
- [ ] Set up database backups
- [ ] Configure log rotation
- [ ] Monitor resource usage
- [ ] Document your configuration in `changes.md`

## Security Considerations

- Passwords hashed with bcrypt (cost factor 10)
- Token-based authentication
- SQL injection protection via GORM
- CORS configuration for production
- HTTPS recommended for production

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

MIT License - See LICENSE file for details

## Acknowledgments

- Jazz community for standards information
- Real Book for jazz standard references
- Open source community for tools and libraries

## Support

For issues, questions, or contributions:
- Open an issue on GitHub
- Submit a pull request
- Contact the maintainer

## Migration from Python

If you're migrating from the Python version:

1. Export your data (if needed)
2. Set up the Go version
3. Import data using the import script
4. Test all functionality
5. Remove Python files

The database schema is compatible, but the API may have minor differences.

## Future Enhancements

- [ ] Mobile apps (React Native)
- [ ] Set list management
- [ ] Practice session tracking
- [ ] Chord chart integration
- [ ] Social features (share repertoire)
- [ ] Export to PDF/CSV
- [ ] Integration with iReal Pro
- [ ] Audio recordings/references

## Version History

### v2.0.0 (Go Rewrite)
- Complete rewrite in Go
- Progressive Web App frontend
- User categories feature
- Improved performance and scalability
- Docker support

### v1.0.0 (Python)
- Initial Python/FastAPI implementation
- Basic user and standards management
- Simple web interface
