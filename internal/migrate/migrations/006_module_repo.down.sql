ALTER TABLE module_metrics
    DROP INDEX idx_repo_module_period,
    ADD UNIQUE INDEX idx_module_period (module_name, period_type, period_start);

ALTER TABLE module_metrics DROP COLUMN repo;
