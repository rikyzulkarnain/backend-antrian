-- Realign counters to a 2-loket "flex" layout: both loket aktif, tanpa
-- spesialisasi service (service_type = NULL) sehingga setiap loket bisa
-- melayani semua kategori (UMUM / LAB / AMP / UTIL / SEWA).
--
-- Idempotent: aman dijalankan ulang.
UPDATE counters SET is_active = false WHERE id NOT IN (1, 2);

INSERT INTO counters (id, name, service_type, is_active)
VALUES
  (1, 'Loket 1', NULL, true),
  (2, 'Loket 2', NULL, true)
ON CONFLICT (id) DO UPDATE
  SET name = EXCLUDED.name,
      service_type = NULL,
      is_active = true;

-- Realign SERIAL sequence agar INSERT berikutnya tidak bentrok.
SELECT setval(
  pg_get_serial_sequence('counters', 'id'),
  GREATEST((SELECT COALESCE(MAX(id), 0) FROM counters), 2)
);
