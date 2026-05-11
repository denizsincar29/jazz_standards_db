# Jazz Standards Database

A self-hosted REST API + PWA for tracking jazz standards, personal pieces, and original compositions. Built with Go, PostgreSQL, and Gorilla Mux.

---

## Table of Contents

1. [Features](#features)
2. [Quick Start](#quick-start)
3. [Configuration](#configuration)
4. [Apache Reverse Proxy](#apache-reverse-proxy)
5. [Importing Standards](#importing-standards)
6. [API Reference](#api-reference)
7. [Pass Keys](#pass-keys)
8. [ntfy Notifications](#ntfy-notifications)
9. [Running Tests](#running-tests)

---

## Features

| Feature | Description |
|---|---|
| **300+ Jazz Standards** | Pre-seeded database with composers, keys, styles, and iReal Pro links |
| **User Lists** | Add standards to your personal list with proficiency level, notes, category |
| **Proficiency Tracking** | `beginner → learning → know_it → master` per standard |
| **Practice Logs** | Log practice sessions with duration and notes |
| **Categories** | Colour-coded categories to organise your list (Ballads, Gig Tunes, etc.) |
| **Personal Pieces** | Private list of rare/local tunes not in the global database |
| **Composed Tunes** | Your own compositions, shareable via a unique URL |
| **Shared Tune Links** | Send a composition to another user; they click Accept to add it |
| **Pass Keys** | Named API tokens for scripted / programmatic access |
| **Public Profiles** | Opt-in: let others browse your standard list |
| **Admin Approval** | User-submitted standards require admin sign-off |
| **ntfy Notifications** | Push alerts to admins when a standard awaits approval |
| **JSON + CSV Export** | Export your list in either format |
| **Random Picker** | Get a random standard, optionally filtered by style/key |
| **Base-path Support** | Run behind an Apache `ProxyPass` at any sub-path |
| **PWA** | Installable progressive web app with service worker |

---

## Quick Start

### Docker Compose (recommended)

```bash
cp .env.example .env          # edit DB credentials, JWT_SECRET, etc.
docker-compose up -d
# Create first admin
docker-compose exec app ./create_admin
# Seed the 300-standard database
ADMIN_TOKEN=<token> API_URL=http://localhost:8000 \
  go run scripts/import_standards.go scripts/standards_seed.json
```

### Local development

```bash
# Requires Go 1.22+ and PostgreSQL
cp .env.example .env
go run . &
./create_admin          # or: go run cmd/create_admin/main.go
```

---

## Configuration

All settings are read from environment variables (or `.env` in the project root).

| Variable | Default | Description |
|---|---|---|
| `DB_HOST` | `localhost` | PostgreSQL host |
| `DB_PORT` | `5432` | PostgreSQL port |
| `DB_USER` | `jazz` | DB user |
| `DB_PASSWORD` | `jazz` | DB password |
| `DB_NAME` | `jazz` | DB name |
| `PORT` | `8000` | HTTP listen port |
| `JWT_SECRET` | *(change me)* | Token signing secret |
| `BASE_PATH` | *(empty)* | URL prefix for reverse proxy, e.g. `/jazz` |
| `NTFY_URL` | `https://ntfy.sh` | ntfy server base URL |
| `NTFY_TOPIC` | *(empty)* | ntfy topic name (leave empty to disable) |
| `NTFY_TOKEN` | *(empty)* | Bearer token if topic is access-controlled |
| `ENVIRONMENT` | `development` | Set to `production` for quieter DB logging |
| `TEST_API` | *(empty)* | Any non-empty value enables `/testapi` debug page |

Copy `.env.example` to `.env` and edit before starting.

---

## Apache Reverse Proxy

Set `BASE_PATH=/jazz` in your `.env`, then add to your Apache vhost:

```apache
<VirtualHost *:443>
    ServerName example.com

    # Proxy the app
    ProxyPass        /jazz/ http://localhost:8000/jazz/
    ProxyPassReverse /jazz/ http://localhost:8000/jazz/

    # WebSocket support (optional)
    RewriteEngine On
    RewriteCond %{HTTP:Upgrade} websocket [NC]
    RewriteCond %{HTTP:Connection} upgrade [NC]
    RewriteRule ^/jazz/(.*) ws://localhost:8000/jazz/$1 [P,L]

    # Required modules: mod_proxy mod_proxy_http mod_rewrite
</VirtualHost>
```

The app normalises `BASE_PATH` automatically — leading/trailing slashes are handled, so `/jazz`, `jazz`, and `/jazz/` all work.

---

## Importing Standards

### Option 1 – Bulk import API (recommended)

```bash
# 1. Create an admin user and log in to get a token
curl -s -X POST http://localhost:8000/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"yourpassword"}' | jq .token

# 2. Import the bundled 300-standard seed file
ADMIN_TOKEN=<token from step 1>
go run scripts/import_standards.go scripts/standards_seed.json
```

### Option 2 – One standard at a time

```bash
curl -X POST http://localhost:8000/api/jazz_standards \
  -H "Authorization: Bearer $ADMIN_TOKEN" \
  -H 'Content-Type: application/json' \
  -d '{
    "title": "Autumn Leaves",
    "composer": "Joseph Kosma",
    "style": "swing",
    "key": "G-",
    "ireal_pro_link": "irealbook://Autumn%20Leaves=Joseph%20Kosma=Medium%20Swing=..."
  }'
```

### Getting iReal Pro URLs for existing standards

iReal Pro stores its library as URL-encoded links. There are three ways to get them:

1. **From the app** – open a chart in iReal Pro → Share → Copy iReal Pro Link.
2. **irealb.com forums** – the community at [https://www.irealb.com/forums/](https://www.irealb.com/forums/) maintains playlists with hundreds of standards. Download a playlist, open it in iReal Pro, then export individual links.
3. **iRealb community playlists** – playlists like *"The Jazz 1350"* or *"Jazz Standards"* are shared on the forums as single `irealbook://` URLs containing dozens of songs. Parse them with a script and import the key/style data.

> **Tip**: The `ireal_pro_link` field accepts any `irealbook://` URL. Paste it straight from the iReal Pro share sheet.

### Bulk JSON format

The seed file and import script use this schema:

```json
[
  {
    "title": "All the Things You Are",
    "composer": "Jerome Kern",
    "style": "swing",
    "key": "Ab",
    "additional_note": "From the musical Very Warm for May",
    "ireal_pro_link": "irealbook://..."
  }
]
```

Valid styles: `dixieland`, `ragtime`, `big_band`, `bossa_nova`, `samba`, `latin`, `latin_swing`, `swing`, `waltz`, `bebop`, `modal`, `free`, `fusion`.

---

## API Reference

All endpoints are under `/api` (or `/<BASE_PATH>/api` if a base path is set).

### Authentication

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/register` | Register: `{username, name, password}` → `{token, user}` |
| `POST` | `/api/login` | Login: `{username, password}` → `{token, user}` |
| `POST` | `/api/logout` | Invalidates the current session token |

Use the returned token as `Authorization: Bearer <token>` on all subsequent requests, or as the `token` cookie.

### My Profile

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/users/me` | Get current user |
| `PATCH` | `/api/users/me` | Update `{name?, public_profile?}` |

### Pass Keys

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/users/me/passkeys` | Create pass key `{name}` — returns raw token **once** |
| `GET` | `/api/users/me/passkeys` | List pass keys (tokens hidden, hint shown) |
| `DELETE` | `/api/users/me/passkeys/{id}` | Revoke a pass key |

### Jazz Standards (global database)

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/jazz_standards` | User | List/search. Params: `search`, `style`, `key`, `page`, `limit` |
| `GET` | `/api/jazz_standards/{id}` | User | Get one standard |
| `GET` | `/api/jazz_standards/random` | User | Random standard. Params: `style`, `key` |
| `POST` | `/api/jazz_standards` | User | Submit a standard (admin → auto-approved; user → pending) |
| `PUT` | `/api/jazz_standards/{id}` | Admin | Update standard |
| `DELETE` | `/api/jazz_standards/{id}` | Admin | Delete standard |
| `POST` | `/api/jazz_standards/{id}/approve` | Admin | Approve a pending standard |
| `POST` | `/api/jazz_standards/{id}/reject` | Admin | Reject a pending standard |
| `GET` | `/api/jazz_standards/pending` | Admin | List pending submissions |
| `POST` | `/api/jazz_standards/bulk_import` | Admin | Import array of standards (JSON body) |

### My Standard List

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/users/me/standards` | List my standards (grouped by category). Param: `proficiency` |
| `POST` | `/api/users/me/standards/{id}` | Add a standard: `{proficiency?, notes?, category_id?}` |
| `PUT` | `/api/users/me/standards/{id}` | Update entry: `{proficiency?, notes?, category_id?}` |
| `DELETE` | `/api/users/me/standards/{id}` | Remove from my list |
| `GET` | `/api/users/me/standards/export` | Export list. Param: `format=json\|csv` |
| `POST` | `/api/users/me/standards/{id}/practice` | Log practice: `{duration_min, notes?, practiced_at?}` |
| `GET` | `/api/users/me/practice` | List practice logs. Param: `standard_id` |

### Categories

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/users/me/categories` | List my categories |
| `POST` | `/api/users/me/categories` | Create: `{name, color?}` |
| `PUT` | `/api/users/me/categories/{id}` | Update: `{name?, color?}` |
| `DELETE` | `/api/users/me/categories/{id}` | Delete (standards move to Uncategorized) |

### Personal Pieces

Personal pieces are rare/local tunes that aren't jazz standards — local compositions your band plays, obscure originals, etc. They live in your personal space and are never submitted for admin review.

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/users/me/pieces` | List my personal pieces |
| `POST` | `/api/users/me/pieces` | Create: `{title, composer?, style?, key?, notes?, ireal_pro_link?, is_public?}` |
| `PUT` | `/api/users/me/pieces/{id}` | Update |
| `DELETE` | `/api/users/me/pieces/{id}` | Delete |

### Composed Tunes & Sharing

Composed tunes are your own original compositions. Each gets a unique share URL that you can send to another musician.

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/users/me/compositions` | User | List my compositions |
| `POST` | `/api/users/me/compositions` | User | Create: `{title, composer?, style?, key?, description?, ireal_pro_link?, is_public?}` |
| `PUT` | `/api/users/me/compositions/{id}` | User | Update |
| `DELETE` | `/api/users/me/compositions/{id}` | User | Delete |
| `GET` | `/api/shared_tune?id=<shareID>` | None | View a public composition by share ID |
| `POST` | `/api/shared_tune/accept?id=<shareID>` | User | Accept into personal pieces |

**Sharing workflow:**
1. You create a composition with `is_public: true`.
2. The response contains `share_id` — build the share URL:
   `https://yourdomain.com/api/shared_tune?id=<share_id>`
3. Send the URL to a colleague.
4. They `POST /api/shared_tune/accept?id=<share_id>` — the tune is copied into their personal pieces.

### Public Profiles

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/users/{username}/standards` | View another user's list — only if `public_profile: true` |

### Admin

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/admin/stats` | Totals: users, standards, pending, practice minutes, top 10 |
| `GET` | `/api/users` | List all users |
| `DELETE` | `/api/users/{id}` | Delete a user |

---

## Pass Keys

Pass keys are named long-lived API tokens for scripts and integrations. Unlike the session token (which is overwritten on each login), you can create multiple pass keys and revoke them individually.

```bash
# Create
curl -X POST http://localhost:8000/api/users/me/passkeys \
  -H "Authorization: Bearer $SESSION_TOKEN" \
  -d '{"name": "My Import Script"}'
# Response: {"token": "abc...xyz", "token_hint": "xyz", "message": "Save this token – it will not be shown again"}

# Use exactly like a session token
curl http://localhost:8000/api/jazz_standards \
  -H "Authorization: Bearer abc...xyz"

# List (tokens hidden)
curl http://localhost:8000/api/users/me/passkeys \
  -H "Authorization: Bearer $SESSION_TOKEN"

# Revoke
curl -X DELETE http://localhost:8000/api/users/me/passkeys/1 \
  -H "Authorization: Bearer $SESSION_TOKEN"
```

---

## ntfy Notifications

[ntfy.sh](https://ntfy.sh) delivers push notifications to your phone or desktop when a user submits a standard for review.

**Setup:**
1. Install the ntfy app on your phone and subscribe to a private topic (e.g. `jazz_db_admin_abc123`).
2. Add to `.env`:

```env
NTFY_URL=https://ntfy.sh
NTFY_TOPIC=jazz_db_admin_abc123
# Optional: protect the topic with an access token
NTFY_TOKEN=tk_mytoken
```

3. Restart the app.

You can also self-host ntfy — just point `NTFY_URL` at your instance.

---

## Running Tests

Tests require a PostgreSQL instance (separate from production).

```bash
# Start a test DB
docker run -d --name jazz_test_db \
  -e POSTGRES_USER=jazz -e POSTGRES_PASSWORD=jazz -e POSTGRES_DB=jazz_test \
  -p 5433:5432 postgres:15-alpine

# Run all tests
export TEST_DB_HOST=localhost TEST_DB_PORT=5433 \
       TEST_DB_USER=jazz TEST_DB_PASSWORD=jazz TEST_DB_NAME=jazz_test
go test ./tests/... -v

# Run a specific test file
go test ./tests/... -run TestCreateAndUsePassKey -v
```

If `TEST_DB_HOST` is not set or the DB is unreachable, database tests are **skipped** automatically (not failed), so `go test ./...` always succeeds in CI environments without a DB.

---

## Docker Compose

```yaml
# .env.example values used:
EXTERNAL_PORT=8000
DB_USER=jazz
DB_PASSWORD=changeme
DB_NAME=jazz
JWT_SECRET=change-me-in-production
ENVIRONMENT=production
BASE_PATH=          # e.g. /jazz for reverse proxy
NTFY_TOPIC=         # leave empty to disable
NTFY_TOKEN=
```

```bash
docker-compose up -d
docker-compose logs -f app
```
