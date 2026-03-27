CREATE TABLE IF NOT EXISTS installations (
    id                     BIGINT AUTO_INCREMENT PRIMARY KEY,
    github_installation_id BIGINT NOT NULL UNIQUE,
    owner                  VARCHAR(255) NOT NULL,
    status                 ENUM('active', 'suspended', 'removed') NOT NULL DEFAULT 'active',
    created_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at             TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS repositories (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    installation_id BIGINT NOT NULL,
    github_repo_id  BIGINT NOT NULL UNIQUE,
    full_name       VARCHAR(255) NOT NULL,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (installation_id) REFERENCES installations(id),
    INDEX idx_full_name (full_name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
