-- Migration: 002_auth_users.up.sql
-- Create auth users table for JWT authentication

CREATE TABLE IF NOT EXISTS gohelper_auth_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    is_superuser BOOLEAN DEFAULT false,
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_auth_users_email ON gohelper_auth_users(email);
CREATE INDEX IF NOT EXISTS idx_auth_users_is_superuser ON gohelper_auth_users(is_superuser);
CREATE INDEX IF NOT EXISTS idx_auth_users_active ON gohelper_auth_users(active);
