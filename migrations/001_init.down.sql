-- Migration: 001_init.down.sql
-- Drop initial schema for Vert Helper

DROP TABLE IF EXISTS gohelper_worker_snapshots CASCADE;
DROP TABLE IF EXISTS gohelper_workers CASCADE;
DROP TABLE IF EXISTS gohelper_action_executions CASCADE;
DROP TABLE IF EXISTS gohelper_questions CASCADE;
DROP TABLE IF EXISTS gohelper_actions CASCADE;
DROP TABLE IF EXISTS gohelper_service_health CASCADE;
DROP TABLE IF EXISTS gohelper_services CASCADE;
