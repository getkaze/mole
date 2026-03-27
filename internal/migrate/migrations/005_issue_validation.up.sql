ALTER TABLE issues ADD COLUMN github_comment_id BIGINT DEFAULT NULL;
ALTER TABLE issues ADD COLUMN validation ENUM('pending', 'confirmed', 'false_positive') NOT NULL DEFAULT 'pending';
ALTER TABLE issues ADD COLUMN validated_by VARCHAR(255) DEFAULT NULL;
ALTER TABLE issues ADD COLUMN validated_at TIMESTAMP NULL DEFAULT NULL;

ALTER TABLE issues ADD INDEX idx_github_comment_id (github_comment_id);
ALTER TABLE issues ADD INDEX idx_validation (validation);
