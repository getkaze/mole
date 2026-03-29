ALTER TABLE module_metrics
    ADD COLUMN repo VARCHAR(255) NOT NULL DEFAULT '' AFTER id;

-- Backfill repo from issues → reviews join
UPDATE module_metrics mm
JOIN (
    SELECT DISTINCT i.module_name, r.repo
    FROM issues i
    JOIN reviews r ON r.id = i.review_id
    WHERE i.module_name IS NOT NULL AND i.module_name != ''
) src ON src.module_name = mm.module_name
SET mm.repo = src.repo;

-- Replace old unique index with one that includes repo
ALTER TABLE module_metrics
    DROP INDEX idx_module_period,
    ADD UNIQUE INDEX idx_repo_module_period (repo, module_name, period_type, period_start);
