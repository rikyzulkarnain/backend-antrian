-- Add Buku Tamu service
INSERT INTO services (key, code, name, description, glyph, color_bg, color_fg, color_border, sop_steps, qr_url, avg_wait_min, display_order) VALUES
('TAMU', 'F', 'Buku Tamu',
 'Pendaftaran kunjungan tamu dan pengunjung ke fasilitas balai.',
 'F',
 'oklch(0.96 0.05 40)', 'oklch(0.5 0.15 40)', 'oklch(0.9 0.05 40)',
 '[
    "Siapkan identitas diri (KTP/SIM) dan tujuan kunjungan.",
    "Isi form pendaftaran tamu dengan lengkap dan jelas.",
    "Cetak nomor antrian di kiosk.",
    "Tunggu panggilan dan lapor ke meja receptionist.",
    "Receptionist akan mengarahkan ke departemen tujuan.",
    "Estimasi waktu: 5–10 menit per tamu."
  ]'::jsonb,
 'https://binamarga.pu.go.id/balai-sumsel/gomusi_app', 7, 6);
