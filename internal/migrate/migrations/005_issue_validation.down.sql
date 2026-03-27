ALTER TABLE issues DROP INDEX IF EXISTS idx_validation;
ALTER TABLE issues DROP INDEX IF EXISTS idx_github_comment_id;
ALTER TABLE issues DROP COLUMN IF EXISTS validated_at;
ALTER TABLE issues DROP COLUMN IF EXISTS validated_by;
ALTER TABLE issues DROP COLUMN IF EXISTS validation;
ALTER TABLE issues DROP COLUMN IF EXISTS github_comment_id;
