DROP INDEX IF EXISTS idx_counters_active;

ALTER TABLE counters
  DROP COLUMN IF EXISTS staff_id,
  DROP COLUMN IF EXISTS service_type;
