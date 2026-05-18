ALTER TABLE queues
  ADD COLUMN IF NOT EXISTS respondent_name  TEXT NULL,
  ADD COLUMN IF NOT EXISTS respondent_phone TEXT NULL,
  ADD COLUMN IF NOT EXISTS issue_category   TEXT NULL;

ALTER TABLE queues
  ADD CONSTRAINT issue_category_valid CHECK (
    issue_category IS NULL OR issue_category IN
      ('TIDAK_ADA', 'PETUGAS', 'FASILITAS', 'PROSEDUR', 'WAKTU', 'LAINNYA')
  );
