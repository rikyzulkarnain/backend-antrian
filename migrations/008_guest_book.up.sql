-- Buku Tamu: visitors register name + purpose (via QR form on their phone)
-- before a queue number is issued. guest_token links the kiosk session to the
-- mobile form submission; UNIQUE makes the guest create idempotent per token.
ALTER TABLE queues ADD COLUMN guest_name    TEXT;
ALTER TABLE queues ADD COLUMN guest_purpose TEXT;
ALTER TABLE queues ADD COLUMN guest_token   TEXT UNIQUE;

CREATE INDEX idx_queues_guest_token ON queues(guest_token);
