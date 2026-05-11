#!/usr/bin/env bash
# ============================================================
# Jazz Standards DB — interactive installer
# ============================================================
# What this script does:
#   1. Checks dependencies (Go, PostgreSQL)
#   2. Asks for every .env value interactively (with sensible defaults)
#   3. Creates the PostgreSQL role + database (via sudo -u postgres) if absent
#   4. Writes .env
#   5. Builds the binary
#   6. Installs a systemd service
#   7. Writes a ready-to-paste Apache ProxyPass snippet
# ============================================================

set -euo pipefail

# ── colour helpers ──────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; RESET='\033[0m'

info()    { echo -e "${CYAN}[INFO]${RESET}  $*"; }
success() { echo -e "${GREEN}[OK]${RESET}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${RESET}  $*"; }
error()   { echo -e "${RED}[ERROR]${RESET} $*" >&2; }
die()     { error "$*"; exit 1; }
bold()    { echo -e "${BOLD}$*${RESET}"; }

# ── ask helpers ─────────────────────────────────────────────
# ask VAR "Prompt" "default"
ask() {
    local var="$1" prompt="$2" default="$3"
    local hint=""
    [[ -n "$default" ]] && hint=" [${CYAN}${default}${RESET}]"
    while true; do
        echo -en "${BOLD}${prompt}${RESET}${hint}: "
        read -r value
        value="${value:-$default}"
        if [[ -z "$value" ]]; then
            warn "This field is required."
        else
            eval "$var=\"\$value\""
            return
        fi
    done
}

# ask_optional VAR "Prompt" "default"  – empty is accepted
ask_optional() {
    local var="$1" prompt="$2" default="$3"
    local hint=""
    [[ -n "$default" ]] && hint=" [${CYAN}${default}${RESET}]"
    echo -en "${BOLD}${prompt}${RESET}${hint}: "
    read -r value
    eval "$var=\"\${value:-\$default}\""
}

# ask_secret VAR "Prompt"  – hides input
ask_secret() {
    local var="$1" prompt="$2"
    while true; do
        echo -en "${BOLD}${prompt}${RESET}: "
        read -rs value
        echo
        if [[ -z "$value" ]]; then
            warn "This field is required."
        else
            eval "$var=\"\$value\""
            return
        fi
    done
}

# ask_yesno VAR "Prompt" "y|n"
ask_yesno() {
    local var="$1" prompt="$2" default="${3:-n}"
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

# ── locate script directory (repo root) ─────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ============================================================
bold ""
bold "╔══════════════════════════════════════════════════════╗"
bold "║         Jazz Standards DB  —  Installer              ║"
bold "╚══════════════════════════════════════════════════════╝"
echo ""

# ============================================================
# 1. Dependency check
# ============================================================
info "Checking dependencies…"

check_cmd() {
    if command -v "$1" &>/dev/null; then
        success "$1 found: $(command -v "$1")"
    else
        die "$1 not found. Please install it and re-run."
    fi
}

check_cmd go
check_cmd psql
check_cmd sudo
check_cmd systemctl

GO_VERSION=$(go version | awk '{print $3}')
info "Go version: $GO_VERSION"
echo ""

# ============================================================
# 2. Collect configuration interactively
# ============================================================
bold "── Database ────────────────────────────────────────────"
ask         DB_HOST     "PostgreSQL host"          "localhost"
ask         DB_PORT     "PostgreSQL port"          "5432"
ask         DB_NAME     "Database name"            "jazz"
ask         DB_USER     "Database role/user"       "jazz"
ask_secret  DB_PASSWORD "Database password"
echo ""

bold "── Application ─────────────────────────────────────────"
ask         APP_PORT    "HTTP listen port"         "8000"
ask         APP_ENV     "Environment (development|production)" "production"

# JWT secret: offer to generate one
ask_yesno GENJWT "Generate a random JWT secret automatically?" "y"
if [[ "$GENJWT" == "y" ]]; then
    JWT_SECRET="$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 48)"
    success "Generated JWT secret."
else
    ask_secret JWT_SECRET "JWT secret (long random string)"
fi
echo ""

bold "── Base path (reverse proxy) ───────────────────────────"
echo "  Leave empty to serve from /  (direct access, no proxy)"
echo "  Set e.g. /jazz to serve at https://example.com/jazz"
ask_optional BASE_PATH "Base path" ""

# Normalise base path: ensure leading /, no trailing /
if [[ -n "$BASE_PATH" ]]; then
    [[ "${BASE_PATH:0:1}" != "/" ]] && BASE_PATH="/$BASE_PATH"
    BASE_PATH="${BASE_PATH%/}"
fi
echo ""

bold "── ntfy push notifications ─────────────────────────────"
echo "  Admins receive a push alert when a user submits a standard."
echo "  Leave NTFY_TOPIC empty to disable."
ask_optional NTFY_URL   "ntfy server URL"   "https://ntfy.sh"
ask_optional NTFY_TOPIC "ntfy topic name"   ""
ask_optional NTFY_TOKEN "ntfy Bearer token (if protected)" ""
echo ""

bold "── Install paths ───────────────────────────────────────"
ask         INSTALL_DIR "Binary install directory" "/opt/jazz_standards_db"
ask         SERVICE_USER "Run service as user"     "www-data"
echo ""

# ============================================================
# 3. Confirm before proceeding
# ============================================================
bold "── Summary ─────────────────────────────────────────────"
echo "  DB host:port      : ${DB_HOST}:${DB_PORT}"
echo "  Database          : ${DB_NAME}"
echo "  DB user           : ${DB_USER}"
echo "  App port          : ${APP_PORT}"
echo "  Base path         : ${BASE_PATH:-/  (root)}"
echo "  Environment       : ${APP_ENV}"
echo "  ntfy topic        : ${NTFY_TOPIC:-disabled}"
echo "  Install directory : ${INSTALL_DIR}"
echo "  Service user      : ${SERVICE_USER}"
echo ""
ask_yesno CONFIRM "Proceed with installation?" "y"
[[ "$CONFIRM" != "y" ]] && { info "Aborted."; exit 0; }
echo ""

# ============================================================
# 4. PostgreSQL — create role + database if absent
# ============================================================
info "Setting up PostgreSQL…"

PG_SUPERUSER="postgres"

pg_exec() {
    sudo -u "$PG_SUPERUSER" psql -v ON_ERROR_STOP=1 -q "$@"
}

pg_exists_role() {
    sudo -u "$PG_SUPERUSER" psql -tAc \
        "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" postgres \
        2>/dev/null | grep -q 1
}

pg_exists_db() {
    sudo -u "$PG_SUPERUSER" psql -tAc \
        "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}'" postgres \
        2>/dev/null | grep -q 1
}

if pg_exists_role; then
    info "PostgreSQL role '${DB_USER}' already exists — skipping."
else
    info "Creating PostgreSQL role '${DB_USER}'…"
    pg_exec -d postgres <<-SQL
        CREATE ROLE "${DB_USER}" WITH LOGIN PASSWORD '${DB_PASSWORD}';
SQL
    success "Role '${DB_USER}' created."
fi

if pg_exists_db; then
    info "Database '${DB_NAME}' already exists — skipping."
else
    info "Creating database '${DB_NAME}'…"
    pg_exec -d postgres <<-SQL
        CREATE DATABASE "${DB_NAME}" OWNER "${DB_USER}";
SQL
    success "Database '${DB_NAME}' created."
fi

# Grant just in case role was pre-existing but didn't own the DB
pg_exec -d postgres \
    -c "GRANT ALL PRIVILEGES ON DATABASE \"${DB_NAME}\" TO \"${DB_USER}\";" \
    2>/dev/null || true

success "PostgreSQL ready."
echo ""

# ============================================================
# 5. Write .env
# ============================================================
info "Writing .env…"

ENV_FILE="${SCRIPT_DIR}/.env"

cat > "$ENV_FILE" <<EOF
# Generated by install.sh — $(date -u '+%Y-%m-%d %H:%M:%S UTC')

# ── Database ──────────────────────────────────────────────
DB_HOST=${DB_HOST}
DB_PORT=${DB_PORT}
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}

# ── Server ────────────────────────────────────────────────
PORT=${APP_PORT}
JWT_SECRET=${JWT_SECRET}
ENVIRONMENT=${APP_ENV}

# Sub-path when running behind a reverse proxy (e.g. /jazz).
# Leave empty to serve from root /.
BASE_PATH=${BASE_PATH}

# Set any non-empty value to enable /testapi debug page.
TEST_API=

# ── ntfy push notifications ───────────────────────────────
NTFY_URL=${NTFY_URL}
NTFY_TOPIC=${NTFY_TOPIC}
NTFY_TOKEN=${NTFY_TOKEN}
EOF

chmod 600 "$ENV_FILE"
success ".env written (permissions 600)."
echo ""

# ============================================================
# 6. Build the binary
# ============================================================
info "Building Jazz Standards DB binary…"

export GONOSUMDB="*"
export GOFLAGS="-mod=mod"

if ! go build -o jazz_standards_db . 2>&1; then
    die "Build failed. See errors above."
fi
success "Binary built: ${SCRIPT_DIR}/jazz_standards_db"
echo ""

# ============================================================
# 7. Install to INSTALL_DIR
# ============================================================
info "Installing to ${INSTALL_DIR}…"

sudo mkdir -p "$INSTALL_DIR"
sudo cp jazz_standards_db "$INSTALL_DIR/jazz_standards_db"
sudo chmod +x "$INSTALL_DIR/jazz_standards_db"

# Copy static assets and scripts
for d in static scripts; do
    if [[ -d "${SCRIPT_DIR}/${d}" ]]; then
        sudo cp -r "${SCRIPT_DIR}/${d}" "${INSTALL_DIR}/${d}"
    fi
done

# Copy .env to install dir
sudo cp "$ENV_FILE" "${INSTALL_DIR}/.env"
sudo chmod 600 "${INSTALL_DIR}/.env"

# Ensure service user can read everything
sudo chown -R "${SERVICE_USER}:${SERVICE_USER}" "$INSTALL_DIR" 2>/dev/null || \
    sudo chown -R "${SERVICE_USER}" "$INSTALL_DIR" 2>/dev/null || \
    warn "Could not chown ${INSTALL_DIR} to ${SERVICE_USER} — check manually."

success "Installed to ${INSTALL_DIR}."
echo ""

# ============================================================
# 8. Systemd service
# ============================================================
info "Creating systemd service…"

SERVICE_FILE="/etc/systemd/system/jazz-standards-db.service"

sudo tee "$SERVICE_FILE" > /dev/null <<EOF
# Jazz Standards DB — generated by install.sh
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

# Hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=${INSTALL_DIR}

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable jazz-standards-db.service
success "Service installed: jazz-standards-db.service"
echo ""

# Ask whether to start the service now
ask_yesno START_NOW "Start the service now?" "y"
if [[ "$START_NOW" == "y" ]]; then
    sudo systemctl restart jazz-standards-db.service
    sleep 2
    if sudo systemctl is-active --quiet jazz-standards-db.service; then
        success "Service is running."
    else
        warn "Service did not start cleanly. Check: sudo journalctl -u jazz-standards-db -n 50"
    fi
fi
echo ""

# ============================================================
# 9. Apache ProxyPass snippet
# ============================================================
APACHE_SNIPPET_FILE="${SCRIPT_DIR}/apache_proxy.conf"

if [[ -n "$BASE_PATH" ]]; then
    # Reverse-proxy at a sub-path
    PROXY_PATH="${BASE_PATH}/"
    PROXY_TARGET="http://localhost:${APP_PORT}${BASE_PATH}/"

    cat > "$APACHE_SNIPPET_FILE" <<EOF
# ── Jazz Standards DB — Apache ProxyPass config ───────────────────────────────
# Add these two lines inside your <VirtualHost> block.
# Required Apache modules: mod_proxy  mod_proxy_http
#   sudo a2enmod proxy proxy_http && sudo systemctl reload apache2
#
# Base path : ${BASE_PATH}
# App port  : ${APP_PORT}

ProxyPass        ${PROXY_PATH}  ${PROXY_TARGET}
ProxyPassReverse ${PROXY_PATH}  ${PROXY_TARGET}
EOF

else
    # Serving from root — proxy everything
    cat > "$APACHE_SNIPPET_FILE" <<EOF
# ── Jazz Standards DB — Apache ProxyPass config ───────────────────────────────
# App is served from root (/). Add these lines inside your <VirtualHost>.
# Required Apache modules: mod_proxy  mod_proxy_http
#   sudo a2enmod proxy proxy_http && sudo systemctl reload apache2
#
# App port : ${APP_PORT}

ProxyPass        /  http://localhost:${APP_PORT}/
ProxyPassReverse /  http://localhost:${APP_PORT}/
EOF

fi

success "Apache config snippet written: ${APACHE_SNIPPET_FILE}"
echo ""

# ============================================================
# 10. Done — print summary
# ============================================================
bold "╔══════════════════════════════════════════════════════╗"
bold "║                  Installation complete               ║"
bold "╚══════════════════════════════════════════════════════╝"
echo ""
echo -e "  Binary      : ${GREEN}${INSTALL_DIR}/jazz_standards_db${RESET}"
echo -e "  Config      : ${GREEN}${INSTALL_DIR}/.env${RESET}"
echo -e "  Service     : ${GREEN}jazz-standards-db.service${RESET}"
echo ""
echo -e "  ${BOLD}Apache proxy snippet:${RESET}"
echo "  ┌──────────────────────────────────────────────────┐"
while IFS= read -r line; do
    # Skip comment lines in the display
    [[ "$line" =~ ^# ]] && continue
    [[ -z "$line" ]]    && continue
    echo "  │  $line"
done < "$APACHE_SNIPPET_FILE"
echo "  └──────────────────────────────────────────────────┘"
echo ""
echo -e "  Full snippet file: ${GREEN}${APACHE_SNIPPET_FILE}${RESET}"
echo ""
bold "  Next steps:"
echo "   1. Create the first admin user:"
echo "      sudo -u ${SERVICE_USER} ${INSTALL_DIR}/jazz_standards_db --create-admin"
echo "      (or run:  go run cmd/create_admin/main.go  from the source tree)"
echo ""
echo "   2. Seed the 298-standard database:"
echo "      Read the token from step 1, then:"
echo "      ADMIN_TOKEN=<token> API_URL=http://localhost:${APP_PORT}${BASE_PATH} \\"
echo "        go run scripts/import_standards.go scripts/standards_seed.json"
echo ""
echo "   3. Paste the Apache snippet into your <VirtualHost> and reload:"
echo "      sudo a2enmod proxy proxy_http"
echo "      sudo systemctl reload apache2"
echo ""
echo "   4. Service management:"
echo "      sudo systemctl status  jazz-standards-db"
echo "      sudo systemctl restart jazz-standards-db"
echo "      sudo journalctl -u     jazz-standards-db -f"
echo ""
