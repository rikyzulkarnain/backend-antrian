-- Counter→Service assignment + currently-assigned staff.
-- service_type is nullable so a "flex" counter can serve any service.
ALTER TABLE counters
  ADD COLUMN service_type TEXT,
  ADD COLUMN staff_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_counters_active ON counters(is_active);
