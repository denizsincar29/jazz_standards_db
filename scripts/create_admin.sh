#!/bin/bash

# Interactive script to create admin users in the Jazz Standards DB
# This script connects directly to the PostgreSQL database

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Jazz Standards DB - Create Admin User ===${NC}"
echo ""

# Load environment variables from .env if it exists
if [ -f .env ]; then
    echo -e "${GREEN}Loading configuration from .env...${NC}"
    export $(cat .env | grep -v '^#' | xargs)
else
    echo -e "${RED}Warning: .env file not found. Using defaults.${NC}"
fi

# Get database connection details
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-jazz}
DB_PASSWORD=${DB_PASSWORD:-jazz}
DB_NAME=${DB_NAME:-jazz}

# Prompt for admin details
echo ""
echo -e "${BLUE}Enter admin user details:${NC}"
read -p "Username: " USERNAME
read -p "Full Name: " NAME
read -sp "Password: " PASSWORD
echo ""

# Validate inputs
if [ -z "$USERNAME" ] || [ -z "$NAME" ] || [ -z "$PASSWORD" ]; then
    echo -e "${RED}Error: All fields are required${NC}"
    exit 1
fi

# Hash the password using Python (bcrypt)
echo ""
echo -e "${GREEN}Hashing password...${NC}"

# Check if Python 3 is available
if ! command -v python3 &> /dev/null; then
    echo -e "${RED}Error: python3 is required but not installed${NC}"
    exit 1
fi

# Install bcrypt if needed
python3 -c "import bcrypt" 2>/dev/null || {
    echo -e "${BLUE}Installing bcrypt...${NC}"
    pip3 install bcrypt --quiet || {
        echo -e "${RED}Error: Failed to install bcrypt. Please install it manually: pip3 install bcrypt${NC}"
        exit 1
    }
}

# Hash the password
PASSWORD_HASH=$(python3 -c "import bcrypt; print(bcrypt.hashpw('$PASSWORD'.encode(), bcrypt.gensalt()).decode())")

# Generate a random token
TOKEN=$(openssl rand -hex 32)

echo -e "${GREEN}Creating admin user in database...${NC}"

# Insert into database
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME << EOSQL
INSERT INTO users (username, name, password_hash, is_admin, token)
VALUES ('$USERNAME', '$NAME', '$PASSWORD_HASH', true, '$TOKEN')
ON CONFLICT (username) DO UPDATE
SET is_admin = true,
    password_hash = EXCLUDED.password_hash,
    token = EXCLUDED.token;
EOSQL

if [ $? -eq 0 ]; then
    echo ""
    echo -e "${GREEN}=== Admin user created successfully! ===${NC}"
    echo ""
    echo -e "${BLUE}Username:${NC} $USERNAME"
    echo -e "${BLUE}Token:${NC} $TOKEN"
    echo ""
    echo -e "${GREEN}You can now login with your username and password.${NC}"
else
    echo -e "${RED}Error: Failed to create admin user${NC}"
    exit 1
fi
