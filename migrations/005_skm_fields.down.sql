ALTER TABLE queues DROP CONSTRAINT IF EXISTS issue_category_valid;
ALTER TABLE queues
  DROP COLUMN IF EXISTS issue_category,
  DROP COLUMN IF EXISTS respondent_phone,
  DROP COLUMN IF EXISTS respondent_name;
