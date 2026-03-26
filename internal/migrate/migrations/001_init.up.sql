CREATE TABLE IF NOT EXISTS reviews (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    repo          VARCHAR(255) NOT NULL,
    pr_number     INT NOT NULL,
    review_type   ENUM('standard', 'deep') NOT NULL,
    model         VARCHAR(100) NOT NULL,
    input_tokens  INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    status        ENUM('success', 'failed') NOT NULL,
    summary       TEXT,
    error_message TEXT,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_repo_pr (repo, pr_number),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ignored_prs (
    id         BIGINT AUTO_INCREMENT PRIMARY KEY,
    repo       VARCHAR(255) NOT NULL,
    pr_number  INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE INDEX idx_repo_pr (repo, pr_number)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
