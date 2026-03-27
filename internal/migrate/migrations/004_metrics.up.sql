CREATE TABLE IF NOT EXISTS developer_metrics (
    id                 BIGINT AUTO_INCREMENT PRIMARY KEY,
    developer          VARCHAR(255) NOT NULL,
    period_type        ENUM('weekly', 'monthly') NOT NULL,
    period_start       DATE NOT NULL,
    period_end         DATE NOT NULL,
    total_reviews      INT NOT NULL DEFAULT 0,
    avg_score          DECIMAL(5,2) NOT NULL DEFAULT 0,
    issues_by_category JSON,
    issues_by_severity JSON,
    streak_clean_prs   INT NOT NULL DEFAULT 0,
    badges             JSON,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_dev_period (developer, period_type, period_start),
    INDEX idx_period (period_start, period_end)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS module_metrics (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    module_name  VARCHAR(255) NOT NULL,
    period_type  ENUM('weekly', 'monthly') NOT NULL,
    period_start DATE NOT NULL,
    period_end   DATE NOT NULL,
    avg_score    DECIMAL(5,2) NOT NULL DEFAULT 0,
    health_score DECIMAL(5,2) NOT NULL DEFAULT 0,
    total_issues INT NOT NULL DEFAULT 0,
    debt_items   INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_module_period (module_name, period_type, period_start)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS dashboard_access (
    id                    BIGINT AUTO_INCREMENT PRIMARY KEY,
    github_user           VARCHAR(255) NOT NULL UNIQUE,
    role                  ENUM('dev', 'tech_lead', 'architect', 'manager', 'admin') NOT NULL DEFAULT 'dev',
    individual_visibility BOOLEAN NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
