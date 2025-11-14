# Jazz Standards Database - Go Implementation Specification

## Overview

This document specifies the complete reimplementation of the Jazz Standards Database from Python/FastAPI to Go, with enhanced features including PWA support, user categories, and a common shared database.

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Web Framework**: Standard library `net/http` with `gorilla/mux` for routing
- **ORM**: GORM v2
- **Database**: PostgreSQL 15+
- **Authentication**: JWT tokens + bcrypt password hashing
- **Environment**: godotenv for configuration

### Frontend (PWA)
- **Type**: Progressive Web App
- **Technologies**: HTML5, CSS3, JavaScript (vanilla or minimal framework)
- **Features**: Service Worker, Web Manifest, offline support
- **UI Framework**: Lightweight responsive CSS (or Tailwind CSS via CDN)

## Database Schema

### Tables

#### 1. users
```sql
id              SERIAL PRIMARY KEY
username        VARCHAR(255) UNIQUE NOT NULL
name            VARCHAR(255) NOT NULL
password_hash   BYTEA NOT NULL
is_admin        BOOLEAN DEFAULT FALSE
token           VARCHAR(255) UNIQUE
created_at      TIMESTAMP DEFAULT NOW()
updated_at      TIMESTAMP DEFAULT NOW()
```

#### 2. jazz_standards (Common/Shared)
```sql
id              SERIAL PRIMARY KEY
title           VARCHAR(255) UNIQUE NOT NULL
composer        VARCHAR(255) NOT NULL
additional_note TEXT
style           VARCHAR(50) NOT NULL
created_by      INTEGER REFERENCES users(id)
created_at      TIMESTAMP DEFAULT NOW()
updated_at      TIMESTAMP DEFAULT NOW()
```

**Styles**: dixieland, ragtime, big_band, bossa_nova, samba, latin, latin_swing, swing, waltz, bebop, modal, free, fusion

#### 3. user_categories
```sql
id              SERIAL PRIMARY KEY
user_id         INTEGER REFERENCES users(id) ON DELETE CASCADE
name            VARCHAR(255) NOT NULL
color           VARCHAR(7) DEFAULT '#000000'
created_at      TIMESTAMP DEFAULT NOW()
UNIQUE(user_id, name)
```

#### 4. user_jazz_standards
```sql
user_id         INTEGER REFERENCES users(id) ON DELETE CASCADE
jazz_standard_id INTEGER REFERENCES jazz_standards(id) ON DELETE CASCADE
category_id     INTEGER REFERENCES user_categories(id) ON DELETE SET NULL
notes           TEXT
created_at      TIMESTAMP DEFAULT NOW()
PRIMARY KEY(user_id, jazz_standard_id)
```

## API Endpoints

### Authentication

#### POST /api/register
Creates a new user account (non-admin)
```json
Request: {
  "username": "string",
  "name": "string",
  "password": "string"
}
Response: {
  "id": 1,
  "username": "string",
  "name": "string",
  "is_admin": false,
  "token": "string"
}
```

#### POST /api/login
Authenticates user and returns token
```json
Request: {
  "username": "string",
  "password": "string"
}
Response: {
  "id": 1,
  "username": "string",
  "name": "string",
  "is_admin": false,
  "token": "string"
}
```

#### POST /api/admin (Admin Creation)
Creates first admin user (only works if no users exist)
```json
Request: {
  "username": "string",
  "name": "string",
  "password": "string"
}
```

### User Management

#### GET /api/users/me
Returns current authenticated user
**Auth**: Required

#### GET /api/users
Lists all users (admin only)
**Auth**: Admin

#### DELETE /api/users/:id
Deletes a user (self or admin)
**Auth**: Required

### Jazz Standards (Common Database)

#### POST /api/jazz_standards
Creates a new jazz standard (admin only)
```json
Request: {
  "title": "string",
  "composer": "string",
  "additional_note": "string",
  "style": "swing"
}
```
**Auth**: Admin

#### GET /api/jazz_standards
Lists all jazz standards with pagination and search
**Query Params**: 
- `page` (default: 1)
- `limit` (default: 100)
- `search` (optional: searches title and composer)
- `style` (optional: filter by style)
**Auth**: Required

#### GET /api/jazz_standards/:id
Gets a specific jazz standard
**Auth**: Required

#### PUT /api/jazz_standards/:id
Updates a jazz standard (admin only)
**Auth**: Admin

#### DELETE /api/jazz_standards/:id
Deletes a jazz standard (admin only)
**Auth**: Admin

### User Standards (Personal Tracking)

#### POST /api/users/me/standards/:standard_id
Adds a standard to user's known list
```json
Request: {
  "category_id": 1,  // optional
  "notes": "string"   // optional
}
```
**Auth**: Required

#### GET /api/users/me/standards
Gets all standards user knows, grouped by category
**Auth**: Required

#### PUT /api/users/me/standards/:standard_id
Updates user's notes or category for a standard
**Auth**: Required

#### DELETE /api/users/me/standards/:standard_id
Removes standard from user's known list
**Auth**: Required

### Categories

#### POST /api/users/me/categories
Creates a new category
```json
Request: {
  "name": "string",
  "color": "#FF5733"
}
```
**Auth**: Required

#### GET /api/users/me/categories
Lists user's categories
**Auth**: Required

#### PUT /api/users/me/categories/:id
Updates a category
**Auth**: Required

#### DELETE /api/users/me/categories/:id
Deletes a category (sets standards to no category)
**Auth**: Required

## Go Project Structure

```
/
├── main.go                 # Entry point
├── go.mod
├── go.sum
├── .env.example
├── README.md
├── IMPLEMENTATION_SPEC.md
├── docker-compose.yml
├── Dockerfile
│
├── config/
│   └── config.go          # Configuration management
│
├── models/
│   ├── user.go
│   ├── jazz_standard.go
│   ├── category.go
│   └── user_standard.go
│
├── database/
│   ├── database.go        # GORM setup
│   └── migrations.go      # Auto-migrations
│
├── handlers/
│   ├── auth.go           # Login, register, admin
│   ├── users.go          # User management
│   ├── standards.go      # Jazz standards CRUD
│   ├── user_standards.go # User-standard associations
│   └── categories.go     # Category management
│
├── middleware/
│   ├── auth.go           # Authentication middleware
│   └── admin.go          # Admin-only middleware
│
├── utils/
│   ├── password.go       # Password hashing
│   ├── token.go          # Token generation
│   └── response.go       # JSON response helpers
│
├── static/              # PWA Frontend
│   ├── index.html
│   ├── manifest.json
│   ├── sw.js            # Service worker
│   ├── css/
│   │   └── styles.css
│   ├── js/
│   │   ├── app.js
│   │   └── api.js
│   └── icons/
│       └── icon-*.png
│
└── scripts/
    └── import_standards.go  # Script to import standards
```

## PWA Features

### 1. Manifest (manifest.json)
```json
{
  "name": "Jazz Standards DB",
  "short_name": "JazzDB",
  "start_url": "/",
  "display": "standalone",
  "background_color": "#ffffff",
  "theme_color": "#1a1a2e",
  "icons": [
    {
      "src": "/static/icons/icon-192.png",
      "sizes": "192x192",
      "type": "image/png"
    },
    {
      "src": "/static/icons/icon-512.png",
      "sizes": "512x512",
      "type": "image/png"
    }
  ]
}
```

### 2. Service Worker (sw.js)
- Cache static assets
- Offline fallback page
- Cache API responses with stale-while-revalidate strategy
- Background sync for adding standards offline

### 3. UI Features
- Responsive design (mobile-first)
- Search bar with live filtering
- Category filters and organization
- Add standard to personal list with search
- Create and manage categories
- Offline indicator
- Pull-to-refresh

## Authentication Flow

1. **Registration/Login**: User submits credentials → Server validates → Returns JWT token
2. **Token Storage**: Client stores token in localStorage
3. **API Requests**: Client includes token in Authorization header: `Bearer <token>`
4. **Token Validation**: Middleware validates token on protected routes
5. **Token Refresh**: Optional: implement refresh token mechanism

## Admin Workflow

1. **First Admin**: Use POST /api/admin to create first admin (only works once)
2. **Add Standards**: Admin adds standards via API or web interface
3. **Manage Users**: Admin can view all users and standards

## User Workflow

1. **Register/Login**: Create account or log in
2. **Browse Standards**: Search and filter common standards database
3. **Add to My List**: Click "I Know This" to add standard
4. **Organize**: Create categories (e.g., "Working On", "Mastered")
5. **Manage**: Assign standards to categories, add notes

## Importing 2000+ Jazz Standards

### Option 1: Manual Admin Entry via API
Use provided import script with CSV/JSON data file

### Option 2: Web Scraping (Ethical)
- Wikipedia's List of Jazz Standards
- Jazz Standards.com (check robots.txt)
- Real Book indices (public domain only)

### Option 3: Manual Database Entry
Create SQL script with INSERT statements

### Sample Import Script (scripts/import_standards.go)
```go
// Reads from standards.json and inserts via API
// Format: [{"title": "...", "composer": "...", "style": "..."}]
```

### Data Sources Guide (in README)
```markdown
## Importing Jazz Standards Data

### Recommended Sources:
1. **Wikipedia**: "List of jazz standards" - public domain
2. **Jazz Standards Index**: Compile from public sources
3. **Real Book**: Public domain standards only
4. **User Contributions**: Community-sourced

### Import Process:
1. Prepare CSV/JSON file with: title, composer, style
2. Set admin token in .env
3. Run: `go run scripts/import_standards.go standards.json`
4. Monitor for duplicates and errors
```

## Environment Configuration

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=jazz
DB_PASSWORD=jazz
DB_NAME=jazz

# Server
PORT=8000
JWT_SECRET=your-secret-key-here

# Admin (for imports)
ADMIN_TOKEN=your-admin-token
```

## Docker Configuration

### docker-compose.yml
```yaml
version: '3.8'

services:
  db:
    image: postgres:15
    environment:
      POSTGRES_USER: jazz
      POSTGRES_PASSWORD: jazz
      POSTGRES_DB: jazz
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  app:
    build: .
    ports:
      - "8000:8000"
    environment:
      DB_HOST: db
      DB_PORT: 5432
      DB_USER: jazz
      DB_PASSWORD: jazz
      DB_NAME: jazz
      JWT_SECRET: change-me-in-production
    depends_on:
      - db

volumes:
  pgdata:
```

## Migration from Python

1. **Export existing data** (if any exists)
2. **Set up Go environment**
3. **Run migrations** (GORM auto-migrate)
4. **Import data** (if exported)
5. **Delete Python files**
6. **Update documentation**

## Testing Strategy

1. **Unit Tests**: Test individual functions (password hashing, token generation)
2. **Integration Tests**: Test API endpoints with test database
3. **E2E Tests**: Test PWA functionality
4. **Load Tests**: Ensure 2000+ standards perform well

## Security Considerations

1. **Password Storage**: bcrypt with cost factor 10+
2. **SQL Injection**: Use GORM parameterized queries
3. **XSS Protection**: Sanitize user input
4. **CSRF**: Implement CSRF tokens for forms
5. **Rate Limiting**: Prevent brute force attacks
6. **HTTPS**: Enforce in production

## Performance Optimizations

1. **Database Indexing**: Index on title, composer, style
2. **Pagination**: Limit query results
3. **Caching**: Cache common queries (standards list)
4. **Connection Pooling**: GORM connection pool
5. **Static Asset Caching**: Service worker caching

## Future Enhancements

1. **Social Features**: Share playlists/repertoire
2. **Practice Tracking**: Log practice sessions
3. **Chord Charts**: Integrate with iReal Pro format
4. **Set Lists**: Create and manage gig set lists
5. **Export**: Export personal list to PDF/CSV
6. **Mobile Apps**: React Native wrapper

## Success Criteria

- ✅ All Python functionality replicated in Go
- ✅ PWA passes Lighthouse audit (90+ PWA score)
- ✅ Supports 2000+ standards with <100ms query time
- ✅ User categories fully functional
- ✅ Admin can add/edit/delete standards
- ✅ Users can search and track standards
- ✅ Offline support works
- ✅ Docker deployment successful
- ✅ Security audit passes (CodeQL)

## Timeline Estimate

- **Phase 1**: Setup & Models (2-3 hours)
- **Phase 2**: API Implementation (4-5 hours)
- **Phase 3**: PWA Frontend (3-4 hours)
- **Phase 4**: Testing & Polish (2-3 hours)
- **Total**: ~12-15 hours

## Questions for Clarification

1. Should we preserve existing user data or start fresh?
2. Preferred JWT expiration time?
3. Maximum category limit per user?
4. Should standards have additional fields (key, tempo, difficulty)?
5. Image/audio file support for standards?
