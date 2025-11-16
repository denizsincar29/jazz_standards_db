#!/bin/bash

# Interactive script to create an admin user in the Jazz Standards DB
# This script reads the database configuration from .env and creates an admin user

set -e

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Load .env file
if [ -f "$PROJECT_DIR/.env" ]; then
    export $(grep -v '^#' "$PROJECT_DIR/.env" | xargs)
else
    echo "Error: .env file not found in $PROJECT_DIR"
    echo "Please run ./build.sh or ./build_docker.sh first to create the configuration."
    exit 1
fi

echo "================================================"
echo "  Jazz Standards DB - Create Admin User"
echo "================================================"
echo ""

# Get database connection details
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-jazz}"
DB_USER="${DB_USER:-jazz}"
DB_PASSWORD="${DB_PASSWORD:-jazz}"

# Prompt for admin details
read -p "Enter admin username: " ADMIN_USERNAME
read -p "Enter admin display name: " ADMIN_NAME
read -s -p "Enter admin password: " ADMIN_PASSWORD
echo ""
read -s -p "Confirm admin password: " ADMIN_PASSWORD_CONFIRM
echo ""

if [ "$ADMIN_PASSWORD" != "$ADMIN_PASSWORD_CONFIRM" ]; then
    echo "Error: Passwords do not match"
    exit 1
fi

if [ -z "$ADMIN_USERNAME" ] || [ -z "$ADMIN_NAME" ] || [ -z "$ADMIN_PASSWORD" ]; then
    echo "Error: All fields are required"
    exit 1
fi

echo ""
echo "Creating admin user..."

# Generate bcrypt hash using Go
HASH=$(go run << 'GOCODE'
package main
import (
    "fmt"
    "os"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    password := os.Args[1]
    hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: %v\n", err)
        os.Exit(1)
    }
    fmt.Print(string(hash))
}
GOCODE
"$ADMIN_PASSWORD" 2>/dev/null)

if [ $? -ne 0 ]; then
    echo "Error: Failed to generate password hash"
    echo "Make sure Go is installed and golang.org/x/crypto/bcrypt is available"
    exit 1
fi

# Insert admin into database
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" << SQL
INSERT INTO users (username, name, password_hash, is_admin, created_at)
VALUES ('$ADMIN_USERNAME', '$ADMIN_NAME', '$HASH', true, NOW())
ON CONFLICT (username) DO UPDATE
SET password_hash = EXCLUDED.password_hash,
    is_admin = true;
SQL

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Admin user '$ADMIN_USERNAME' created successfully!"
    echo ""
    echo "You can now log in at:"
    if [ -n "$BASE_PATH" ] && [ "$BASE_PATH" != "/" ]; then
        echo "  http://yourdomain.com${BASE_PATH}"
    else
        echo "  http://yourdomain.com/"
    fi
else
    echo "Error: Failed to create admin user"
    exit 1
fi
