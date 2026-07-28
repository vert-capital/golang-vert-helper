-- Migration: 002_auth_users.down.sql
-- Drop auth users table for JWT authentication

DROP TABLE IF EXISTS gohelper_auth_users CASCADE;
