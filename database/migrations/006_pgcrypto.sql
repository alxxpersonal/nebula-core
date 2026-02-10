-- ---
-- ADD PGCRYPTO EXTENSION
-- ---
-- gen_random_bytes() used by generate_job_id() in 000_init.sql requires
-- pgcrypto on PostgreSQL setups where it's not available as a core function.

CREATE EXTENSION IF NOT EXISTS pgcrypto;
