#!/bin/bash
# this is a simple script for my docker container
# it will print the environment variables and check if the database exists
# use DB_NAME, DB_USER, DB_PASSWORD from environment, hostname db
# and port 5432
# print all variables
echo "DB_NAME: $DB_NAME"
echo "DB_USER: $DB_USER"
echo "DB_PASSWORD: $DB_PASSWORD"
psql postgresql://$DB_USER:$DB_PASSWORD@db:5432/$DB_NAME -c "SELECT * FROM pg_database WHERE datname = '$DB_NAME';"
