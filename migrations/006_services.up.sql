CREATE TABLE services (
  key            TEXT PRIMARY KEY,
  code           TEXT NOT NULL UNIQUE,
  name           TEXT NOT NULL,
  description    TEXT NOT NULL DEFAULT '',
  glyph          TEXT NOT NULL,
  color_bg       TEXT NOT NULL,
  color_fg       TEXT NOT NULL,
  color_border   TEXT NOT NULL,
  sop_steps      JSONB NOT NULL DEFAULT '[]'::jsonb,
  sop_pdf_url    TEXT NULL,
  qr_url         TEXT NULL,
  avg_wait_min   INTEGER NOT NULL DEFAULT 0,
  is_active      BOOLEAN NOT NULL DEFAULT true,
  display_order  INTEGER NOT NULL DEFAULT 0,
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_services_active_order ON services(is_active, display_order);

-- Seed 5 service dari frontend/lib/constants.ts
INSERT INTO services (key, code, name, description, glyph, color_bg, color_fg, color_border, sop_steps, qr_url, avg_wait_min, display_order) VALUES
('UMUM', 'A', 'Pelayanan Umum',
 'Pendaftaran, informasi, dan layanan administrasi umum.',
 'A',
 'oklch(0.96 0.04 250)', 'oklch(0.4 0.16 250)', 'oklch(0.9 0.05 250)',
 '[
    "Siapkan kartu identitas (KTP/SIM) dan dokumen pendukung.",
    "Pastikan formulir permohonan sudah diisi lengkap.",
    "Cetak nomor antrian, lalu tunggu di area yang disediakan.",
    "Saat nomor Anda dipanggil, segera menuju loket sesuai layar TV.",
    "Estimasi waktu pelayanan: 5–10 menit per pengunjung."
  ]'::jsonb,
 'https://binamarga.pu.go.id/balai-sumsel/gomusi_app', 8, 1),

('LAB', 'B', 'Layanan Lab',
 'Pemeriksaan, kalibrasi, dan pengujian sampel laboratorium.',
 'B',
 'oklch(0.96 0.04 155)', 'oklch(0.4 0.13 155)', 'oklch(0.9 0.05 155)',
 '[
    "Bawa formulir permintaan uji dan sampel dalam wadah steril.",
    "Petugas akan memverifikasi data sampel sebelum diregistrasi.",
    "Pembayaran dilakukan di loket setelah verifikasi sampel.",
    "Estimasi waktu serah-terima: 10–15 menit per sampel."
  ]'::jsonb,
 'https://binamarga.pu.go.id/balai-sumsel/gomusi_app', 14, 2),

('AMP', 'C', 'Layanan AMP',
 'Pengaduan, mediasi, dan permohonan AMP.',
 'C',
 'oklch(0.96 0.05 35)', 'oklch(0.5 0.16 35)', 'oklch(0.9 0.05 35)',
 '[
    "Sertakan dokumen pendukung sesuai jenis permohonan.",
    "Petugas akan mengarahkan ke loket yang sesuai.",
    "Beberapa permohonan memerlukan janji temu lanjutan."
  ]'::jsonb,
 'https://binamarga.pu.go.id/balai-sumsel/gomusi_app', 12, 3),

('UTIL', 'D', 'Layanan Utilitas',
 'Permintaan layanan teknis utilitas dan fasilitas.',
 'D',
 'oklch(0.96 0.05 85)', 'oklch(0.45 0.13 85)', 'oklch(0.9 0.06 85)',
 '[
    "Isi formulir permintaan utilitas dengan jelas.",
    "Cantumkan lokasi dan kontak teknisi penanggung jawab.",
    "Konfirmasi jadwal pelaksanaan dengan petugas loket."
  ]'::jsonb,
 'https://binamarga.pu.go.id/balai-sumsel/gomusi_app', 6, 4),

('SEWA', 'E', 'Sewa Alat',
 'Peminjaman dan pengembalian peralatan operasional.',
 'E',
 'oklch(0.96 0.04 305)', 'oklch(0.42 0.16 305)', 'oklch(0.9 0.05 305)',
 '[
    "Sertakan surat permohonan dan jaminan identitas asli.",
    "Periksa kondisi alat bersama petugas sebelum keluar gudang.",
    "Pengembalian wajib dalam kondisi sesuai serah-terima awal."
  ]'::jsonb,
 'https://binamarga.pu.go.id/balai-sumsel/gomusi_app', 9, 5);
