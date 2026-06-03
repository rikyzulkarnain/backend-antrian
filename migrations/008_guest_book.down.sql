DROP INDEX IF EXISTS idx_queues_guest_token;
ALTER TABLE queues DROP COLUMN IF EXISTS guest_token;
ALTER TABLE queues DROP COLUMN IF EXISTS guest_purpose;
ALTER TABLE queues DROP COLUMN IF EXISTS guest_name;
