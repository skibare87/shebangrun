-- Migration: 002_add_wrapped_key.sql

ALTER TABLE script_content ADD COLUMN wrapped_key BLOB AFTER encryption_key_id;
