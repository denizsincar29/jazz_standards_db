#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="jazz-standards-db.service"
DEFAULT_INSTALL_DIR="/opt/jazz_standards_db"

info()    { echo "[•] $*"; }
success() { echo "[✓] $*"; }
warn()    { echo "[!] $*"; }
die()     { echo "[✗] $*" >&2; exit 1; }

require() {
    command -v "$1" >/dev/null 2>&1 || die "$1 is required but was not found"
}

detect_install_dir() {
    local unit_file="/etc/systemd/system/${SERVICE_NAME}"
    local install_dir=""

    if [[ -f "$unit_file" ]]; then
        install_dir="$(awk -F= '/^WorkingDirectory=/{print $2; exit}' "$unit_file")"
        if [[ -z "$install_dir" ]]; then
            install_dir="$(awk -F= '/^EnvironmentFile=/{print $2; exit}' "$unit_file")"
            install_dir="$(dirname "$install_dir")"
        fi
    fi

    if [[ -z "$install_dir" ]]; then
        install_dir="$DEFAULT_INSTALL_DIR"
    fi

    echo "$install_dir"
}

# ---------------------------------------------------------------------------
# prompt_field KEY DESCRIPTION DEFAULT IS_SECRET
#   Reads an existing value from the .env file (if present).
#   If the field is missing or empty, prompts the user to enter it.
#   Writes the (possibly new) value back into ENV_FILE.
#   IS_SECRET=1 → input is hidden (read -s) and value is not echoed.
# ---------------------------------------------------------------------------
ENV_FILE=""   # set in ensure_env()
declare -A ENV_VALS=()

load_env_file() {
    # Parse KEY=VALUE lines, ignoring comments and blanks.
    if [[ -f "$ENV_FILE" ]]; then
        while IFS= read -r line; do
            [[ "$line" =~ ^[[:space:]]*# ]] && continue
            [[ -z "${line//[[:space:]]/}" ]] && continue
            local key="${line%%=*}"
            local val="${line#*=}"
            # Strip surrounding quotes if present
            val="${val#\"}" ; val="${val%\"}"
            val="${val#\'}" ; val="${val%\'}"
            ENV_VALS["$key"]="$val"
        done < "$ENV_FILE"
    fi
}

write_env_file() {
    # Write all collected key=value pairs to ENV_FILE.
    # Preserve order: required fields first, then optionals.
    {
        echo "# Jazz Standards DB – environment configuration"
        echo "# Generated/updated by rebuild.sh on $(date)"
        echo ""
        echo "# ── Database ─────────────────────────────────────────────"
        _write_key DB_HOST
        _write_key DB_PORT
        _write_key DB_USER
        _write_key DB_PASSWORD
        _write_key DB_NAME
        echo ""
        echo "# ── Application ──────────────────────────────────────────"
        _write_key PORT
        _write_key JWT_SECRET
        _write_key ENVIRONMENT
        _write_key BASE_PATH
        _write_key TEST_API
        echo ""
        echo "# ── WebAuthn (passkeys / biometric login) ────────────────"
        _write_key WEBAUTHN_RPID
        _write_key WEBAUTHN_ORIGINS
        echo ""
        echo "# ── ntfy.sh push notifications (optional) ───────────────"
        _write_key NTFY_URL
        _write_key NTFY_TOPIC
        _write_key NTFY_TOKEN
    } > "$ENV_FILE"
    sudo chmod 600 "$ENV_FILE"
}

_write_key() {
    local key="$1"
    local val="${ENV_VALS[$key]:-}"
    echo "${key}=${val}"
}

prompt_field() {
    local key="$1"
    local description="$2"
    local default="$3"
    local is_secret="${4:-0}"

    local current="${ENV_VALS[$key]:-}"

    if [[ -n "$current" ]]; then
        # Already set – show a hint and keep the existing value.
        if [[ "$is_secret" == "1" ]]; then
            info "  ${key}: [already set, keeping]"
        else
            info "  ${key}: ${current}"
        fi
        return
    fi

    # Field is missing – prompt the user.
    local prompt_str
    if [[ -n "$default" ]]; then
        prompt_str="  ${description} [${key}] (default: ${default}): "
    else
        prompt_str="  ${description} [${key}] (required): "
    fi

    local value=""
    while true; do
        if [[ "$is_secret" == "1" ]]; then
            read -rsp "$prompt_str" value
            echo ""  # newline after hidden input
        else
            read -rp "$prompt_str" value
        fi

        # Use default if user pressed Enter on an optional field.
        if [[ -z "$value" && -n "$default" ]]; then
            value="$default"
        fi

        if [[ -n "$value" ]]; then
            break
        fi

        warn "  This field is required. Please enter a value."
    done

    ENV_VALS["$key"]="$value"
    info "  ${key} set."
}

ensure_env() {
    local install_dir="$1"
    ENV_FILE="${install_dir}/.env"

    sudo mkdir -p "$install_dir"

    # If the .env is owned by root we need sudo to read it; copy to a temp file.
    local tmp_env
    tmp_env="$(mktemp)"
    if [[ -f "$ENV_FILE" ]]; then
        sudo cp "$ENV_FILE" "$tmp_env"
        sudo chmod 644 "$tmp_env"
        ENV_FILE="$tmp_env"   # work on the copy
    else
        ENV_FILE="$tmp_env"
        info "No .env found – will create one at ${install_dir}/.env"
    fi

    load_env_file

    echo ""
    info "Checking environment configuration…"
    echo "  (Press Enter to accept defaults; existing values are kept as-is)"
    echo ""

    # ── Database ──────────────────────────────────────────────────────────
    prompt_field DB_HOST      "PostgreSQL host"        "localhost"  0
    prompt_field DB_PORT      "PostgreSQL port"        "5432"       0
    prompt_field DB_USER      "PostgreSQL user"        "jazz"       0
    prompt_field DB_PASSWORD  "PostgreSQL password"    ""           1
    prompt_field DB_NAME      "PostgreSQL database"    "jazz"       0

    # ── Application ───────────────────────────────────────────────────────
    prompt_field PORT         "HTTP listen port"       "8000"       0
    prompt_field JWT_SECRET   "JWT signing secret"     ""           1
    prompt_field ENVIRONMENT  "Environment (development|production)" "production" 0
    prompt_field BASE_PATH    "URL base path (e.g. /jazz, or leave blank)" "" 0
    prompt_field TEST_API     "Enable test-API page? (1=yes, blank=no)"   "" 0

    # ── WebAuthn ──────────────────────────────────────────────────────────
    # Derive a sensible RPID default from BASE_PATH or just "localhost".
    local rpid_default="localhost"
    prompt_field WEBAUTHN_RPID     "WebAuthn Relying Party ID (plain domain, e.g. example.com)" "$rpid_default" 0
    # Default origin from RPID
    local rpid_val="${ENV_VALS[WEBAUTHN_RPID]:-localhost}"
    local origins_default="https://${rpid_val}"
    [[ "$rpid_val" == "localhost" ]] && origins_default="http://localhost:${ENV_VALS[PORT]:-8000}"
    prompt_field WEBAUTHN_ORIGINS  "WebAuthn allowed origins, comma-separated (e.g. https://example.com)" "$origins_default" 0

    # ── ntfy (optional) ───────────────────────────────────────────────────
    prompt_field NTFY_URL    "ntfy server URL"         "https://ntfy.sh" 0
    prompt_field NTFY_TOPIC  "ntfy topic (leave blank to disable)" "" 0
    prompt_field NTFY_TOKEN  "ntfy Bearer token (if topic is protected)" "" 1

    echo ""

    # Write merged values back.
    write_env_file

    # Copy the updated file to the real install location (needs sudo).
    local real_env="${install_dir}/.env"
    sudo cp "$ENV_FILE" "$real_env"
    sudo chmod 600 "$real_env"
    rm -f "$tmp_env"

    # Reset ENV_FILE to the real path for the rest of the script.
    ENV_FILE="$real_env"

    success ".env is up to date at ${ENV_FILE}"
    echo ""
}

main() {
    require go
    require sudo
    require systemctl

    cd "$SCRIPT_DIR"

    local install_dir
    install_dir="$(detect_install_dir)"

    # ── Ensure .env exists and all fields are filled ───────────────────────
    ensure_env "$install_dir"

    # ── Build ──────────────────────────────────────────────────────────────
    info "Tidying Go modules (updates go.sum for new dependencies)"
    go mod tidy

    info "Building binary"
    export GOFLAGS="-mod=mod"
    go build -o jazz_standards_db .

    # ── Deploy ────────────────────────────────────────────────────────────
    info "Stopping ${SERVICE_NAME}"
    sudo systemctl stop "$SERVICE_NAME"

    info "Installing binary to ${install_dir}"
    sudo mkdir -p "$install_dir"
    sudo cp jazz_standards_db "$install_dir/jazz_standards_db"
    sudo chmod +x "$install_dir/jazz_standards_db"

    # Copy static assets, scripts, tests, and cmd tools
    for d in static scripts tests cmd; do
        if [[ -d "${SCRIPT_DIR}/${d}" ]]; then
            sudo rm -rf "${install_dir}/${d}"
            sudo cp -r "${SCRIPT_DIR}/${d}" "${install_dir}/${d}"
        fi
    done

    info "Reloading systemd and restarting ${SERVICE_NAME}"
    sudo systemctl daemon-reload
    sudo systemctl restart "$SERVICE_NAME"

    success "Rebuild complete"
    echo "Service: sudo systemctl status ${SERVICE_NAME}"
}

main "$@"
