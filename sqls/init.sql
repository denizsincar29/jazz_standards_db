-- this file should be run with postgres user
-- this variables need to be changed to your own values
\set db_name 'jazz'
\set db_user 'jazz'
\set db_password 'jazz'

-- Create user if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_user WHERE usename = :db_user) THEN
        CREATE USER :db_user WITH PASSWORD :'db_password';
    END IF;
END $$;

-- Create the database
CREATE DATABASE :db_name WITH OWNER :db_user;

