# Jazz Standards Database - Go PWA

A Progressive Web App (PWA) for tracking and managing your jazz standards repertoire, built with Go and standard library HTTP.

## Features

- **Progressive Web App**: Install on mobile/desktop, works offline
- **User Management**: Secure authentication with bcrypt and tokens
- **Common Standards Database**: Shared database of jazz standards (admin-only additions)
- **Personal Tracking**: Track which standards you know
- **Categories**: Organize your standards into custom categories
- **Search & Filter**: Find standards by title, composer, or style
- **Admin Panel**: Add, edit, and delete standards (admin users only)
- **Responsive Design**: Works on all devices

## Technology Stack

- **Backend**: Go 1.21+ with standard library HTTP + Gorilla Mux
- **Database**: PostgreSQL 15+ with GORM ORM
- **Frontend**: Vanilla JavaScript PWA (no frameworks)
- **Authentication**: Token-based with bcrypt password hashing
- **Deployment**: Docker & Docker Compose

## Quick Start

### Using Docker (Recommended)

1. **Clone the repository**
   ```bash
   git clone https://github.com/denizsincar29/jazz_standards_db.git
   cd jazz_standards_db
   ```

2. **(Optional) Customize the port**
   
   To use a different port:
   ```bash
   cp .env.example .env
   # Edit .env and set EXTERNAL_PORT=5251 (or your desired port)
   ```

3. **Start with Docker Compose**
   ```bash
   docker-compose up --build
   ```

4. **Access the application**
   - Open http://localhost:8000 in your browser (or your custom port from `.env`)
   - Create the first admin account via API:
     ```bash
     curl -X POST http://localhost:8000/api/admin \
       -H "Content-Type: application/json" \
       -d '{"username":"admin","name":"Admin User","password":"your-secure-password"}'
     ```

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
   cp .env.example .env
   # Edit .env with your database credentials
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

#### POST `/api/admin`
Create first admin user (only works once)
```json
{
  "username": "admin",
  "name": "Admin User",
  "password": "adminpassword"
}
```

### Jazz Standards

#### GET `/api/jazz_standards`
List all standards with pagination and filters
```
Query params:
- page (default: 1)
- limit (default: 100)
- search (optional)
- style (optional)
```

#### POST `/api/jazz_standards` (Admin only)
Create a new jazz standard
```json
{
  "title": "All the Things You Are",
  "composer": "Jerome Kern",
  "style": "swing",
  "additional_note": "From 'Very Warm for May'"
}
```

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

### Custom Port Configuration

To change the external port without modifying version-controlled files:

1. Copy the .env template:
   ```bash
   cp .env.example .env
   ```

2. Edit `.env` to set your external port:
   ```env
   EXTERNAL_PORT=5251  # Your desired port
   ```

3. The `.env` file is gitignored and won't be affected by `git pull`

The container's internal port stays at 8000, but you can access it from your custom port.

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

2. **Set BASE_PATH in .env:**
   ```env
   BASE_PATH=/jazzdb
   ```

The application automatically handles subpath routing based on the BASE_PATH environment variable.

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
