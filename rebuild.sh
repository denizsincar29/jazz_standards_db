#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SERVICE_NAME="jazz-standards-db.service"
DEFAULT_INSTALL_DIR="/opt/jazz_standards_db"

info() { echo "[•] $*"; }
success() { echo "[✓] $*"; }
warn() { echo "[!] $*"; }
die() { echo "[✗] $*" >&2; exit 1; }

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

main() {
    require go
    require sudo
    require systemctl

    cd "$SCRIPT_DIR"

    local install_dir
    install_dir="$(detect_install_dir)"

    info "Building binary"
    export GOFLAGS="-mod=mod"
    go build -o jazz_standards_db .

    info "Stopping ${SERVICE_NAME}"
    sudo systemctl stop "$SERVICE_NAME"

    info "Installing binary to ${install_dir}"
    sudo mkdir -p "$install_dir"
    sudo cp jazz_standards_db "$install_dir/jazz_standards_db"
    sudo chmod +x "$install_dir/jazz_standards_db"

    if [[ -f "$install_dir/.env" ]]; then
        sudo chmod 600 "$install_dir/.env"
    else
        warn "No .env file found in ${install_dir}; service restart will still be attempted"
    fi

    info "Reloading systemd and restarting ${SERVICE_NAME}"
    sudo systemctl daemon-reload
    sudo systemctl restart "$SERVICE_NAME"

    success "Rebuild complete"
    echo "Service: sudo systemctl status ${SERVICE_NAME}"
}

main "$@"