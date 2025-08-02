-- Initialize the database with some basic setup
-- This file is executed when PostgreSQL container starts for the first time

-- Create additional databases if needed
-- CREATE DATABASE easy_orders_test;

-- Enable UUID extension for primary keys
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Set timezone
SET timezone = 'UTC';

-- Create indexes that will be commonly used
-- (These will be recreated by GORM migrations, but good to have for initial setup)

-- You can add any initial data or additional setup here