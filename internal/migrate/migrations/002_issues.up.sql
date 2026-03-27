ALTER TABLE reviews ADD COLUMN score INT DEFAULT NULL;
ALTER TABLE reviews ADD COLUMN pr_author VARCHAR(255) DEFAULT NULL;
ALTER TABLE reviews ADD COLUMN installation_id BIGINT DEFAULT NULL;

CREATE TABLE IF NOT EXISTS issues (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    review_id     BIGINT NOT NULL,
    pr_author     VARCHAR(255) NOT NULL,
    category      VARCHAR(50) NOT NULL,
    subcategory   VARCHAR(100) NOT NULL DEFAULT '',
    severity      ENUM('critical', 'attention', 'suggestion') NOT NULL,
    file_path     VARCHAR(500) NOT NULL,
    line_number   INT NOT NULL,
    description   TEXT NOT NULL,
    suggestion    TEXT,
    module_name   VARCHAR(255) DEFAULT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (review_id) REFERENCES reviews(id),
    INDEX idx_pr_author (pr_author),
    INDEX idx_category (category, subcategory),
    INDEX idx_severity (severity),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
