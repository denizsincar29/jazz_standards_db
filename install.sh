#!/usr/bin/env bash
# ============================================================
# Jazz Standards DB — interactive installer
# ============================================================
# Steps:
#   1. Check dependencies (go, psql, sudo, systemctl, curl, jq)
#   2. Interactively collect every .env field
#   3. Create PostgreSQL role + database (sudo -u postgres) if absent
#   4. Write .env (chmod 600)
#   5. Build the binary
#   6. Install binary + assets to INSTALL_DIR
#   7. Create & enable systemd service
#   8. Start the service and wait for it to be healthy
#   9. Create the first admin user via the API
#  10. Seed the 298 jazz standards via the bulk-import API
#  11. Write apache_proxy.conf (the two ProxyPass lines)
#  12. Print a final summary
# ============================================================

set -euo pipefail

# ── colour helpers ──────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

info()    { echo -e "${CYAN}[•]${RESET} $*"; }
success() { echo -e "${GREEN}[✓]${RESET} $*"; }
warn()    { echo -e "${YELLOW}[!]${RESET} $*"; }
die()     { echo -e "${RED}[✗]${RESET} $*" >&2; exit 1; }
bold()    { echo -e "${BOLD}$*${RESET}"; }
step()    { echo ""; bold "── $* ──────────────────────────────────────────────────"; }

# ── prompt helpers ───────────────────────────────────────────
# ask VAR "Prompt" "default"  (required, loops until non-empty)
ask() {
    local var="$1" prompt="$2" default="$3"
    local hint=""
    [[ -n "$default" ]] && hint=" [${CYAN}${default}${RESET}]"
    while true; do
        echo -en "${BOLD}${prompt}${RESET}${hint}: "
        read -r value
        value="${value:-$default}"
        [[ -n "$value" ]] && { eval "$var=\"\$value\""; return; }
        warn "This field is required."
    done
}

# ask_opt VAR "Prompt" "default"  (empty accepted)
ask_opt() {
    local var="$1" prompt="$2" default="$3"
    local hint=""
    [[ -n "$default" ]] && hint=" [${CYAN}${default}${RESET}]"
    echo -en "${BOLD}${prompt}${RESET}${hint}: "
    read -r value
    eval "$var=\"\${value:-\$default}\""
}

# ask_secret VAR "Prompt"
ask_secret() {
    local var="$1" prompt="$2"
    while true; do
        echo -en "${BOLD}${prompt}${RESET}: "
        read -rs value; echo
        [[ -n "$value" ]] && { eval "$var=\"\$value\""; return; }
        warn "This field is required."
    done
}

# ask_secret_opt VAR "Prompt"  (empty accepted)
ask_secret_opt() {
    local var="$1" prompt="$2"
    echo -en "${BOLD}${prompt}${RESET} (leave blank to skip): "
    read -rs value; echo
    eval "$var=\"\$value\""
}

# ask_yn VAR "Prompt" default(y|n)
ask_yn() {
    local var="$1" prompt="$2" default="${3:-y}"
    local hint="[y/n, default: ${default}]"
    while true; do
        echo -en "${BOLD}${prompt}${RESET} ${hint}: "
        read -r ans
        ans="${ans:-$default}"
        case "${ans,,}" in
            y|yes) eval "$var=y"; return ;;
            n|no)  eval "$var=n"; return ;;
            *)     warn "Please enter y or n." ;;
        esac
    done
}

# ── locate repo root ─────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

SEED_FILE="${SCRIPT_DIR}/scripts/standards_seed.json"

# ============================================================
echo ""
bold "╔══════════════════════════════════════════════════════════╗"
bold "║        Jazz Standards DB  —  Interactive Installer       ║"
bold "╚══════════════════════════════════════════════════════════╝"
echo ""

# ============================================================
# 1. Dependency check
# ============================================================
step "Checking dependencies"

need() {
    if command -v "$1" &>/dev/null; then
        success "$1  →  $(command -v "$1")"
    else
        die "$1 not found. Please install it and re-run."
    fi
}

need go
need psql
need sudo
need systemctl
need curl
need jq

info "Go version: $(go version | awk '{print $3}')"

# ============================================================
# 2. Collect configuration
# ============================================================
step "Database settings"

ask     DB_HOST     "PostgreSQL host"     "localhost"
ask     DB_PORT     "PostgreSQL port"     "5432"
ask     DB_NAME     "Database name"       "jazz"
ask     DB_USER     "Database role/user"  "jazz"
ask_secret  DB_PASSWORD "Database password"

step "Application settings"

ask     APP_PORT "HTTP listen port"                              "8000"
ask     APP_ENV  "Environment (development|production)"          "production"

ask_yn GENJWT "Auto-generate a secure JWT secret?" "y"
if [[ "$GENJWT" == "y" ]]; then
    JWT_SECRET="$(openssl rand -hex 32 2>/dev/null \
        || head -c 48 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 48)"
    success "JWT secret generated."
else
    ask_secret JWT_SECRET "JWT secret (long random string)"
fi

step "Reverse proxy / base path"
echo "  Leave empty to serve from /  (direct, no proxy sub-path)"
echo "  Enter e.g.  /jazz  to serve at  https://example.com/jazz"
ask_opt BASE_PATH "Base path" ""

# Normalise: leading slash, no trailing slash
if [[ -n "$BASE_PATH" && "$BASE_PATH" != "/" ]]; then
    [[ "${BASE_PATH:0:1}" != "/" ]] && BASE_PATH="/$BASE_PATH"
    BASE_PATH="${BASE_PATH%/}"
fi

step "ntfy push notifications"
echo "  Admins receive a push when a user submits a standard for review."
echo "  Leave NTFY_TOPIC empty to disable."
ask_opt     NTFY_URL   "ntfy server URL"    "https://ntfy.sh"
ask_opt     NTFY_TOPIC "ntfy topic name"    ""
ask_secret_opt NTFY_TOKEN "ntfy Bearer token (if your topic is protected)"

step "Install paths"
ask  INSTALL_DIR  "Binary install directory"  "/opt/jazz_standards_db"
ask  SERVICE_USER "Run service as OS user"     "www-data"

step "First admin account"
echo "  This account will be created automatically via the API."
ask        ADMIN_USERNAME "Admin username"     "admin"
ask        ADMIN_NAME     "Admin display name" "Administrator"
ask_secret ADMIN_PASSWORD "Admin password"

step "Seed the jazz standards database?"
echo "  The repo ships with 298 standards (standards_seed.json)."
echo "  They will be imported automatically after the service starts."
ask_yn SEED_DB "Import the 298 jazz standards now?" "y"

# ============================================================
# 3. Confirmation
# ============================================================
step "Summary — please review"
echo "  DB host:port      : ${DB_HOST}:${DB_PORT}"
echo "  Database          : ${DB_NAME}"
echo "  DB user           : ${DB_USER}"
echo "  App port          : ${APP_PORT}"
echo "  Base path         : ${BASE_PATH:-/  (root)}"
echo "  Environment       : ${APP_ENV}"
echo "  ntfy topic        : ${NTFY_TOPIC:-disabled}"
echo "  Install directory : ${INSTALL_DIR}"
echo "  Service user      : ${SERVICE_USER}"
echo "  Admin username    : ${ADMIN_USERNAME}"
echo "  Seed standards    : ${SEED_DB}"
echo ""
ask_yn CONFIRM "Proceed with installation?" "y"
[[ "$CONFIRM" != "y" ]] && { info "Aborted."; exit 0; }

# ============================================================
# 4. PostgreSQL — create role + database if absent
# ============================================================
step "Setting up PostgreSQL"

PG_SUPER="postgres"

pg_exec() { sudo -u "$PG_SUPER" psql -v ON_ERROR_STOP=1 -q "$@"; }

role_exists() {
    sudo -u "$PG_SUPER" psql -tAc \
        "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" postgres \
        2>/dev/null | grep -q 1
}

db_exists() {
    sudo -u "$PG_SUPER" psql -tAc \
        "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" postgres \
        2>/dev/null | grep -q 1
}

if role_exists; then
    info "Role '${DB_USER}' already exists — skipping."
else
    info "Creating role '${DB_USER}'…"
    pg_exec -d postgres -c \
        "CREATE ROLE \"${DB_USER}\" WITH LOGIN PASSWORD '${DB_PASSWORD}';"
    success "Role '${DB_USER}' created."
fi

if db_exists; then
    info "Database '${DB_NAME}' already exists — skipping."
else
    info "Creating database '${DB_NAME}'…"
    pg_exec -d postgres -c \
        "CREATE DATABASE \"${DB_NAME}\" OWNER \"${DB_USER}\";"
    success "Database '${DB_NAME}' created."
fi

# Ensure privileges (idempotent)
pg_exec -d postgres -c \
    "GRANT ALL PRIVILEGES ON DATABASE \"${DB_NAME}\" TO \"${DB_USER}\";" \
    2>/dev/null || true

success "PostgreSQL ready."

# ============================================================
# 5. Write .env
# ============================================================
step "Writing .env"

ENV_FILE="${SCRIPT_DIR}/.env"

cat > "$ENV_FILE" <<EOF
# Generated by install.sh — $(date -u '+%Y-%m-%d %H:%M:%S UTC')

# ── Database ──────────────────────────────────────────────────────
DB_HOST=${DB_HOST}
DB_PORT=${DB_PORT}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}

# ── Server ────────────────────────────────────────────────────────
PORT=${APP_PORT}
JWT_SECRET=${JWT_SECRET}
ENVIRONMENT=${APP_ENV}

# Sub-path when running behind a reverse proxy, e.g. /jazz
# Leave empty to serve from root /
BASE_PATH=${BASE_PATH}

# Set any non-empty value to enable the /testapi debug page
TEST_API=

# ── ntfy push notifications ───────────────────────────────────────
NTFY_URL=${NTFY_URL}
NTFY_TOPIC=${NTFY_TOPIC}
NTFY_TOKEN=${NTFY_TOKEN}
EOF

chmod 600 "$ENV_FILE"
success ".env written (permissions 600)."

# ============================================================
# 6. Build the binary
# ============================================================
step "Building binary"

export GONOSUMDB="*"
export GOFLAGS="-mod=mod"

info "Running go build…"
go build -o jazz_standards_db . || die "Build failed. See errors above."
success "Binary built: ${SCRIPT_DIR}/jazz_standards_db"

# ============================================================
# 7. Install to INSTALL_DIR
# ============================================================
step "Installing to ${INSTALL_DIR}"

sudo mkdir -p "$INSTALL_DIR"
sudo cp jazz_standards_db "$INSTALL_DIR/jazz_standards_db"
sudo chmod +x "$INSTALL_DIR/jazz_standards_db"

for d in static scripts; do
    [[ -d "${SCRIPT_DIR}/${d}" ]] && sudo cp -r "${SCRIPT_DIR}/${d}" "${INSTALL_DIR}/${d}"
done

sudo cp "$ENV_FILE" "${INSTALL_DIR}/.env"
sudo chmod 600 "${INSTALL_DIR}/.env"

# Ownership — tolerate missing user gracefully
if id "$SERVICE_USER" &>/dev/null; then
    sudo chown -R "${SERVICE_USER}:${SERVICE_USER}" "$INSTALL_DIR" 2>/dev/null \
        || sudo chown -R "$SERVICE_USER" "$INSTALL_DIR" 2>/dev/null \
        || warn "Could not chown ${INSTALL_DIR} to ${SERVICE_USER} — check manually."
else
    warn "OS user '${SERVICE_USER}' does not exist. Binary installed but ownership unchanged."
fi

success "Installed to ${INSTALL_DIR}."

# ============================================================
# 8. Systemd service
# ============================================================
step "Creating systemd service"

SERVICE_FILE="/etc/systemd/system/jazz-standards-db.service"

sudo tee "$SERVICE_FILE" > /dev/null <<EOF
# Jazz Standards DB — generated by install.sh on $(date -u '+%Y-%m-%d %H:%M UTC')
[Unit]
Description=Jazz Standards Database API
After=network.target postgresql.service
Wants=postgresql.service

[Service]
Type=simple
User=${SERVICE_USER}
WorkingDirectory=${INSTALL_DIR}
EnvironmentFile=${INSTALL_DIR}/.env
ExecStart=${INSTALL_DIR}/jazz_standards_db
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=jazz-standards-db

# Basic hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=${INSTALL_DIR}

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable jazz-standards-db.service
success "Service installed and enabled: jazz-standards-db.service"

# ── start the service ────────────────────────────────────────
info "Starting service…"
sudo systemctl restart jazz-standards-db.service

# Wait up to 30 s for the HTTP port to open
BASE_URL="http://localhost:${APP_PORT}${BASE_PATH}"
HEALTH_URL="${BASE_URL}/api/jazz_standards"

info "Waiting for app to be ready at ${HEALTH_URL}…"
READY=n
for i in $(seq 1 30); do
    if curl -sf "${HEALTH_URL}" -o /dev/null 2>/dev/null; then
        READY=y; break
    fi
    sleep 1
done

if [[ "$READY" != "y" ]]; then
    warn "App did not respond within 30 s."
    warn "Check: sudo journalctl -u jazz-standards-db -n 50"
    warn "Skipping admin creation and standard seeding — run manually after fixing."
    # Still write the Apache snippet and finish
fi

# ============================================================
# 9. Create admin user via the API
# ============================================================
ADMIN_TOKEN=""

if [[ "$READY" == "y" ]]; then
    step "Creating admin user '${ADMIN_USERNAME}'"

    # Register the admin account
    REG_RESP=$(curl -sf -X POST "${BASE_URL}/api/register" \
        -H "Content-Type: application/json" \
        -d "{\"username\":\"${ADMIN_USERNAME}\",\"name\":\"${ADMIN_NAME}\",\"password\":\"${ADMIN_PASSWORD}\"}" \
        2>/dev/null) || REG_RESP=""

    if [[ -z "$REG_RESP" ]]; then
        warn "Registration endpoint failed — admin may already exist. Trying login…"
    else
        ADMIN_TOKEN=$(echo "$REG_RESP" | jq -r '.token // empty' 2>/dev/null || true)
        [[ -n "$ADMIN_TOKEN" ]] && success "Admin account registered."
    fi

    # If we don't have a token yet (already exists), log in
    if [[ -z "$ADMIN_TOKEN" ]]; then
        LOGIN_RESP=$(curl -sf -X POST "${BASE_URL}/api/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"${ADMIN_USERNAME}\",\"password\":\"${ADMIN_PASSWORD}\"}" \
            2>/dev/null) || LOGIN_RESP=""
        ADMIN_TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token // empty' 2>/dev/null || true)
    fi

    if [[ -z "$ADMIN_TOKEN" ]]; then
        warn "Could not obtain an admin token. Skipping admin promotion and seeding."
        warn "Create the admin manually: go run cmd/create_admin/main.go"
    else
        # Promote to admin in the DB (the register endpoint creates a regular user;
        # we flip is_admin directly since there is no self-promotion endpoint)
        info "Promoting '${ADMIN_USERNAME}' to admin in the database…"
        sudo -u "$PG_SUPER" psql -q -d "$DB_NAME" \
            -c "UPDATE users SET is_admin = true WHERE username = '${ADMIN_USERNAME}';" \
            2>/dev/null || \
        PGPASSWORD="$DB_PASSWORD" psql -q -h "$DB_HOST" -p "$DB_PORT" \
            -U "$DB_USER" -d "$DB_NAME" \
            -c "UPDATE users SET is_admin = true WHERE username = '${ADMIN_USERNAME}';" \
            2>/dev/null || \
            warn "Could not promote via psql — promote manually (see below)."

        # Re-login so the token reflects the admin role (gorm reads is_admin on auth)
        LOGIN_RESP=$(curl -sf -X POST "${BASE_URL}/api/login" \
            -H "Content-Type: application/json" \
            -d "{\"username\":\"${ADMIN_USERNAME}\",\"password\":\"${ADMIN_PASSWORD}\"}" \
            2>/dev/null) || LOGIN_RESP=""
        FRESH_TOKEN=$(echo "$LOGIN_RESP" | jq -r '.token // empty' 2>/dev/null || true)
        [[ -n "$FRESH_TOKEN" ]] && ADMIN_TOKEN="$FRESH_TOKEN"

        success "Admin user '${ADMIN_USERNAME}' is ready."
        info "Admin token: ${ADMIN_TOKEN}"
    fi
fi

# ============================================================
# 10. Seed jazz standards via bulk-import API
# ============================================================
if [[ "$SEED_DB" == "y" && "$READY" == "y" && -n "$ADMIN_TOKEN" ]]; then
    step "Seeding jazz standards database"

    if [[ ! -f "$SEED_FILE" ]]; then
        warn "Seed file not found: ${SEED_FILE} — skipping."
    else
        SEED_COUNT=$(jq 'length' "$SEED_FILE" 2>/dev/null || echo "?")
        info "Importing ${SEED_COUNT} standards from $(basename "$SEED_FILE")…"

        IMPORT_RESP=$(curl -sf -X POST "${BASE_URL}/api/jazz_standards/bulk_import" \
            -H "Authorization: Bearer ${ADMIN_TOKEN}" \
            -H "Content-Type: application/json" \
            -d @"$SEED_FILE" 2>/dev/null) || IMPORT_RESP=""

        if [[ -z "$IMPORT_RESP" ]]; then
            warn "Bulk import request failed. Import manually after install:"
            warn "  ADMIN_TOKEN=<token> API_URL=${BASE_URL} go run scripts/import_standards.go scripts/standards_seed.json"
        else
            IMPORTED=$(echo "$IMPORT_RESP" | jq -r '.imported // 0' 2>/dev/null || echo 0)
            SKIPPED=$(echo  "$IMPORT_RESP" | jq -r '.skipped  // 0' 2>/dev/null || echo 0)
            FAILED=$(echo   "$IMPORT_RESP" | jq -r '.failed   // 0' 2>/dev/null || echo 0)
            success "Import done — imported: ${IMPORTED}, skipped: ${SKIPPED}, failed: ${FAILED}"
        fi
    fi
elif [[ "$SEED_DB" == "y" && ( "$READY" != "y" || -z "$ADMIN_TOKEN" ) ]]; then
    warn "Skipped seeding (service not ready / no admin token)."
    warn "Run manually once the service is up:"
    warn "  ADMIN_TOKEN=<token> API_URL=${BASE_URL} go run scripts/import_standards.go scripts/standards_seed.json"
fi

# ============================================================
# 11. Apache ProxyPass snippet
# ============================================================
step "Generating Apache config snippet"

APACHE_FILE="${SCRIPT_DIR}/apache_proxy.conf"

if [[ -n "$BASE_PATH" ]]; then
    PROXY_PATH="${BASE_PATH}/"
    PROXY_TARGET="http://localhost:${APP_PORT}${BASE_PATH}/"
    cat > "$APACHE_FILE" <<EOF
# ── Jazz Standards DB — Apache ProxyPass ──────────────────────────────────────
# Add these two lines inside your <VirtualHost *:80> or <VirtualHost *:443>.
#
#   Required modules:
#     sudo a2enmod proxy proxy_http && sudo systemctl reload apache2
#
#   Base path : ${BASE_PATH}
#   App port  : ${APP_PORT}

ProxyPass        ${PROXY_PATH}  ${PROXY_TARGET}
ProxyPassReverse ${PROXY_PATH}  ${PROXY_TARGET}
EOF
else
    cat > "$APACHE_FILE" <<EOF
# ── Jazz Standards DB — Apache ProxyPass ──────────────────────────────────────
# App serves from root /. Add these lines inside your <VirtualHost>.
#
#   Required modules:
#     sudo a2enmod proxy proxy_http && sudo systemctl reload apache2
#
#   App port : ${APP_PORT}

ProxyPass        /  http://localhost:${APP_PORT}/
ProxyPassReverse /  http://localhost:${APP_PORT}/
EOF
fi

success "Apache snippet written: ${APACHE_FILE}"

# ============================================================
# 12. Final summary
# ============================================================
echo ""
bold "╔══════════════════════════════════════════════════════════╗"
bold "║                 Installation complete ✓                  ║"
bold "╚══════════════════════════════════════════════════════════╝"
echo ""
echo -e "  Binary      : ${GREEN}${INSTALL_DIR}/jazz_standards_db${RESET}"
echo -e "  Config      : ${GREEN}${INSTALL_DIR}/.env${RESET}"
echo -e "  Service     : ${GREEN}jazz-standards-db.service${RESET}"
[[ -n "$ADMIN_TOKEN" ]] && \
echo -e "  Admin token : ${GREEN}${ADMIN_TOKEN}${RESET}"
echo ""
echo -e "  ${BOLD}Apache ProxyPass (paste into your <VirtualHost>):${RESET}"
echo "  ┌─────────────────────────────────────────────────────┐"
grep -v '^#' "$APACHE_FILE" | grep -v '^$' | while IFS= read -r line; do
    echo "  │  $line"
done
echo "  └─────────────────────────────────────────────────────┘"
echo -e "  Full file: ${GREEN}${APACHE_FILE}${RESET}"
echo ""
bold "  Service management:"
echo "   sudo systemctl status  jazz-standards-db"
echo "   sudo systemctl restart jazz-standards-db"
echo "   sudo journalctl -u     jazz-standards-db -f"
echo ""
if [[ -z "$ADMIN_TOKEN" || "$READY" != "y" ]]; then
    bold "  ⚠  Manual steps still needed:"
    echo "   1. Create admin:  go run cmd/create_admin/main.go"
    echo "   2. Seed DB:       ADMIN_TOKEN=<token> API_URL=${BASE_URL} \\"
    echo "                     go run scripts/import_standards.go scripts/standards_seed.json"
    echo ""
fi
