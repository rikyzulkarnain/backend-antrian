-- Seed initial admin user and the six default counters.
-- Admin credentials are environment-specific. The hash committed here is
-- the production hash (rotated 2026-05-18); local dev environments that
-- need a different password should edit this file in their own branch
-- and reset their local DB.
-- Idempotent: ON CONFLICT (email) DO NOTHING so re-running is safe.

INSERT INTO users (name, email, password_hash, role, is_active) VALUES
  ('Administrator', 'admin@local',
   '$2a$10$GgvSt61.f6v94.Mu43Ng4uF7rkidTk1xt/rPxj0COwK54Bq3oyNc2',
   'admin', true)
ON CONFLICT (email) DO NOTHING;

-- Counter seed. Match the mock layout in frontend/lib/constants.ts so the
-- staff UI has familiar IDs to choose from.
INSERT INTO counters (id, name, service_type, is_active) VALUES
  (1, 'Loket 1', 'UMUM', true),
  (2, 'Loket 2', 'UMUM', true),
  (3, 'Loket 3', 'LAB',  true),
  (4, 'Loket 4', 'AMP',  true),
  (5, 'Loket 5', 'UTIL', true),
  (6, 'Loket 6', 'SEWA', false)
ON CONFLICT (id) DO NOTHING;

-- Realign the SERIAL sequence so future INSERTs without an explicit id
-- don't collide with the seeded rows.
SELECT setval(
  pg_get_serial_sequence('counters', 'id'),
  COALESCE((SELECT MAX(id) FROM counters), 0)
);
