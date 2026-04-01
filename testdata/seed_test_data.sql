-- Seed: reviews + issues for 13 weeks (90 days) for 5 developers
-- Score = 100 - (critical*8) - (attention*5)
-- Run: docker exec -i $(docker ps -qf name=mysql) mysql -umole -pmole mole < seed_test_data.sql

-- Clean previous test data
DELETE FROM issues WHERE review_id IN (SELECT id FROM reviews WHERE repo = 'kaze/test-repo');
DELETE FROM reviews WHERE repo = 'kaze/test-repo';
DELETE FROM developer_metrics WHERE developer IN ('testuser','anasouza','lucasmendes','pedroalves','rafaelcosta') AND period_type = 'weekly';

-- ============================================================
-- REVIEWS (1 per developer per week, 13 weeks)
-- Week Mondays: Jan 5 -> Mar 30, 2026
-- ============================================================

-- testuser: 55 -> 62 (slow climb)
INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score, status, summary, created_at) VALUES
('kaze/test-repo', 1001, 'testuser', 'standard', 'claude-sonnet-4-6', 52, 'success', 'review seed', '2026-01-05 10:00:00'),
('kaze/test-repo', 1002, 'testuser', 'standard', 'claude-sonnet-4-6', 55, 'success', 'review seed', '2026-01-12 10:00:00'),
('kaze/test-repo', 1003, 'testuser', 'standard', 'claude-sonnet-4-6', 48, 'success', 'review seed', '2026-01-19 10:00:00'),
('kaze/test-repo', 1004, 'testuser', 'standard', 'claude-sonnet-4-6', 57, 'success', 'review seed', '2026-01-26 10:00:00'),
('kaze/test-repo', 1005, 'testuser', 'standard', 'claude-sonnet-4-6', 60, 'success', 'review seed', '2026-02-02 10:00:00'),
('kaze/test-repo', 1006, 'testuser', 'standard', 'claude-sonnet-4-6', 55, 'success', 'review seed', '2026-02-09 10:00:00'),
('kaze/test-repo', 1007, 'testuser', 'standard', 'claude-sonnet-4-6', 58, 'success', 'review seed', '2026-02-16 10:00:00'),
('kaze/test-repo', 1008, 'testuser', 'standard', 'claude-sonnet-4-6', 63, 'success', 'review seed', '2026-02-23 10:00:00'),
('kaze/test-repo', 1009, 'testuser', 'standard', 'claude-sonnet-4-6', 60, 'success', 'review seed', '2026-03-02 10:00:00'),
('kaze/test-repo', 1010, 'testuser', 'standard', 'claude-sonnet-4-6', 65, 'success', 'review seed', '2026-03-09 10:00:00'),
('kaze/test-repo', 1011, 'testuser', 'standard', 'claude-sonnet-4-6', 62, 'success', 'review seed', '2026-03-16 10:00:00'),
('kaze/test-repo', 1012, 'testuser', 'standard', 'claude-sonnet-4-6', 60, 'success', 'review seed', '2026-03-23 10:00:00'),
('kaze/test-repo', 1013, 'testuser', 'standard', 'claude-sonnet-4-6', 62, 'success', 'review seed', '2026-03-30 10:00:00');

-- anasouza: 68 -> 74 (steady climb)
INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score, status, summary, created_at) VALUES
('kaze/test-repo', 2001, 'anasouza', 'standard', 'claude-sonnet-4-6', 65, 'success', 'review seed', '2026-01-05 10:00:00'),
('kaze/test-repo', 2002, 'anasouza', 'standard', 'claude-sonnet-4-6', 68, 'success', 'review seed', '2026-01-12 10:00:00'),
('kaze/test-repo', 2003, 'anasouza', 'standard', 'claude-sonnet-4-6', 70, 'success', 'review seed', '2026-01-19 10:00:00'),
('kaze/test-repo', 2004, 'anasouza', 'standard', 'claude-sonnet-4-6', 68, 'success', 'review seed', '2026-01-26 10:00:00'),
('kaze/test-repo', 2005, 'anasouza', 'standard', 'claude-sonnet-4-6', 72, 'success', 'review seed', '2026-02-02 10:00:00'),
('kaze/test-repo', 2006, 'anasouza', 'standard', 'claude-sonnet-4-6', 70, 'success', 'review seed', '2026-02-09 10:00:00'),
('kaze/test-repo', 2007, 'anasouza', 'standard', 'claude-sonnet-4-6', 74, 'success', 'review seed', '2026-02-16 10:00:00'),
('kaze/test-repo', 2008, 'anasouza', 'standard', 'claude-sonnet-4-6', 72, 'success', 'review seed', '2026-02-23 10:00:00'),
('kaze/test-repo', 2009, 'anasouza', 'standard', 'claude-sonnet-4-6', 75, 'success', 'review seed', '2026-03-02 10:00:00'),
('kaze/test-repo', 2010, 'anasouza', 'standard', 'claude-sonnet-4-6', 73, 'success', 'review seed', '2026-03-09 10:00:00'),
('kaze/test-repo', 2011, 'anasouza', 'standard', 'claude-sonnet-4-6', 76, 'success', 'review seed', '2026-03-16 10:00:00'),
('kaze/test-repo', 2012, 'anasouza', 'standard', 'claude-sonnet-4-6', 74, 'success', 'review seed', '2026-03-23 10:00:00'),
('kaze/test-repo', 2013, 'anasouza', 'standard', 'claude-sonnet-4-6', 78, 'success', 'review seed', '2026-03-30 10:00:00');

-- lucasmendes: 72 -> 65 (dip then partial recovery)
INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score, status, summary, created_at) VALUES
('kaze/test-repo', 3001, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 72, 'success', 'review seed', '2026-01-05 10:00:00'),
('kaze/test-repo', 3002, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 75, 'success', 'review seed', '2026-01-12 10:00:00'),
('kaze/test-repo', 3003, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 70, 'success', 'review seed', '2026-01-19 10:00:00'),
('kaze/test-repo', 3004, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 65, 'success', 'review seed', '2026-01-26 10:00:00'),
('kaze/test-repo', 3005, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 58, 'success', 'review seed', '2026-02-02 10:00:00'),
('kaze/test-repo', 3006, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 52, 'success', 'review seed', '2026-02-09 10:00:00'),
('kaze/test-repo', 3007, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 48, 'success', 'review seed', '2026-02-16 10:00:00'),
('kaze/test-repo', 3008, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 55, 'success', 'review seed', '2026-02-23 10:00:00'),
('kaze/test-repo', 3009, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 58, 'success', 'review seed', '2026-03-02 10:00:00'),
('kaze/test-repo', 3010, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 60, 'success', 'review seed', '2026-03-09 10:00:00'),
('kaze/test-repo', 3011, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 63, 'success', 'review seed', '2026-03-16 10:00:00'),
('kaze/test-repo', 3012, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 62, 'success', 'review seed', '2026-03-23 10:00:00'),
('kaze/test-repo', 3013, 'lucasmendes', 'standard', 'claude-sonnet-4-6', 65, 'success', 'review seed', '2026-03-30 10:00:00');

-- pedroalves: 78 -> 87 (consistent improvement, best performer)
INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score, status, summary, created_at) VALUES
('kaze/test-repo', 4001, 'pedroalves', 'standard', 'claude-sonnet-4-6', 76, 'success', 'review seed', '2026-01-05 10:00:00'),
('kaze/test-repo', 4002, 'pedroalves', 'standard', 'claude-sonnet-4-6', 78, 'success', 'review seed', '2026-01-12 10:00:00'),
('kaze/test-repo', 4003, 'pedroalves', 'standard', 'claude-sonnet-4-6', 80, 'success', 'review seed', '2026-01-19 10:00:00'),
('kaze/test-repo', 4004, 'pedroalves', 'standard', 'claude-sonnet-4-6', 79, 'success', 'review seed', '2026-01-26 10:00:00'),
('kaze/test-repo', 4005, 'pedroalves', 'standard', 'claude-sonnet-4-6', 82, 'success', 'review seed', '2026-02-02 10:00:00'),
('kaze/test-repo', 4006, 'pedroalves', 'standard', 'claude-sonnet-4-6', 80, 'success', 'review seed', '2026-02-09 10:00:00'),
('kaze/test-repo', 4007, 'pedroalves', 'standard', 'claude-sonnet-4-6', 84, 'success', 'review seed', '2026-02-16 10:00:00'),
('kaze/test-repo', 4008, 'pedroalves', 'standard', 'claude-sonnet-4-6', 83, 'success', 'review seed', '2026-02-23 10:00:00'),
('kaze/test-repo', 4009, 'pedroalves', 'standard', 'claude-sonnet-4-6', 85, 'success', 'review seed', '2026-03-02 10:00:00'),
('kaze/test-repo', 4010, 'pedroalves', 'standard', 'claude-sonnet-4-6', 88, 'success', 'review seed', '2026-03-09 10:00:00'),
('kaze/test-repo', 4011, 'pedroalves', 'standard', 'claude-sonnet-4-6', 86, 'success', 'review seed', '2026-03-16 10:00:00'),
('kaze/test-repo', 4012, 'pedroalves', 'standard', 'claude-sonnet-4-6', 90, 'success', 'review seed', '2026-03-23 10:00:00'),
('kaze/test-repo', 4013, 'pedroalves', 'standard', 'claude-sonnet-4-6', 87, 'success', 'review seed', '2026-03-30 10:00:00');

-- rafaelcosta: 70 -> 74 (stable with minor oscillation)
INSERT INTO reviews (repo, pr_number, pr_author, review_type, model, score, status, summary, created_at) VALUES
('kaze/test-repo', 5001, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 68, 'success', 'review seed', '2026-01-05 10:00:00'),
('kaze/test-repo', 5002, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 72, 'success', 'review seed', '2026-01-12 10:00:00'),
('kaze/test-repo', 5003, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 70, 'success', 'review seed', '2026-01-19 10:00:00'),
('kaze/test-repo', 5004, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 74, 'success', 'review seed', '2026-01-26 10:00:00'),
('kaze/test-repo', 5005, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 71, 'success', 'review seed', '2026-02-02 10:00:00'),
('kaze/test-repo', 5006, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 73, 'success', 'review seed', '2026-02-09 10:00:00'),
('kaze/test-repo', 5007, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 75, 'success', 'review seed', '2026-02-16 10:00:00'),
('kaze/test-repo', 5008, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 72, 'success', 'review seed', '2026-02-23 10:00:00'),
('kaze/test-repo', 5009, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 76, 'success', 'review seed', '2026-03-02 10:00:00'),
('kaze/test-repo', 5010, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 73, 'success', 'review seed', '2026-03-09 10:00:00'),
('kaze/test-repo', 5011, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 75, 'success', 'review seed', '2026-03-16 10:00:00'),
('kaze/test-repo', 5012, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 74, 'success', 'review seed', '2026-03-23 10:00:00'),
('kaze/test-repo', 5013, 'rafaelcosta', 'standard', 'claude-sonnet-4-6', 76, 'success', 'review seed', '2026-03-30 10:00:00');

-- ============================================================
-- DEVELOPER_METRICS (weekly, ISO weeks Mon-Sun)
-- Computed from the reviews above: avg_score = review score for that week
-- ============================================================

INSERT INTO developer_metrics (developer, period_type, period_start, period_end, total_reviews, avg_score, issues_by_category, issues_by_severity, streak_clean_prs, badges) VALUES
-- testuser: 52 -> 62
('testuser', 'weekly', '2026-01-05', '2026-01-11', 1, 52.00, '{"bugs":2,"smells":1}', '{"critical":3,"attention":4}', 0, '[]'),
('testuser', 'weekly', '2026-01-12', '2026-01-18', 1, 55.00, '{"bugs":2,"smells":1}', '{"critical":3,"attention":3}', 0, '[]'),
('testuser', 'weekly', '2026-01-19', '2026-01-25', 1, 48.00, '{"bugs":3,"smells":1}', '{"critical":4,"attention":3}', 0, '[]'),
('testuser', 'weekly', '2026-01-26', '2026-02-01', 1, 57.00, '{"bugs":2,"smells":1}', '{"critical":3,"attention":3}', 0, '[]'),
('testuser', 'weekly', '2026-02-02', '2026-02-08', 1, 60.00, '{"bugs":1,"smells":1}', '{"critical":2,"attention":4}', 0, '[]'),
('testuser', 'weekly', '2026-02-09', '2026-02-15', 1, 55.00, '{"bugs":2,"smells":1}', '{"critical":3,"attention":3}', 0, '[]'),
('testuser', 'weekly', '2026-02-16', '2026-02-22', 1, 58.00, '{"bugs":2,"smells":1}', '{"critical":3,"attention":2}', 0, '[]'),
('testuser', 'weekly', '2026-02-23', '2026-03-01', 1, 63.00, '{"bugs":1,"smells":1}', '{"critical":2,"attention":3}', 0, '[]'),
('testuser', 'weekly', '2026-03-02', '2026-03-08', 1, 60.00, '{"bugs":2,"smells":1}', '{"critical":2,"attention":4}', 0, '[]'),
('testuser', 'weekly', '2026-03-09', '2026-03-15', 1, 65.00, '{"bugs":1,"smells":1}', '{"critical":2,"attention":3}', 0, '[]'),
('testuser', 'weekly', '2026-03-16', '2026-03-22', 1, 62.00, '{"bugs":1,"smells":1}', '{"critical":2,"attention":4}', 0, '[]'),
('testuser', 'weekly', '2026-03-23', '2026-03-29', 1, 60.00, '{"bugs":2,"smells":1}', '{"critical":2,"attention":4}', 0, '[]'),
('testuser', 'weekly', '2026-03-30', '2026-04-05', 1, 62.00, '{"bugs":1,"smells":1}', '{"critical":2,"attention":4}', 0, '[]'),

-- anasouza: 65 -> 78
('anasouza', 'weekly', '2026-01-05', '2026-01-11', 1, 65.00, '{"bugs":1,"security":1}', '{"critical":2,"attention":3}', 0, '[]'),
('anasouza', 'weekly', '2026-01-12', '2026-01-18', 1, 68.00, '{"bugs":1,"security":1}', '{"critical":2,"attention":2}', 0, '[]'),
('anasouza', 'weekly', '2026-01-19', '2026-01-25', 1, 70.00, '{"bugs":1}', '{"critical":1,"attention":4}', 1, '[]'),
('anasouza', 'weekly', '2026-01-26', '2026-02-01', 1, 68.00, '{"bugs":1,"security":1}', '{"critical":2,"attention":2}', 0, '[]'),
('anasouza', 'weekly', '2026-02-02', '2026-02-08', 1, 72.00, '{"bugs":1}', '{"critical":1,"attention":3}', 1, '[]'),
('anasouza', 'weekly', '2026-02-09', '2026-02-15', 1, 70.00, '{"bugs":1}', '{"critical":1,"attention":4}', 1, '[]'),
('anasouza', 'weekly', '2026-02-16', '2026-02-22', 1, 74.00, '{"security":1}', '{"critical":1,"attention":3}', 1, '[]'),
('anasouza', 'weekly', '2026-02-23', '2026-03-01', 1, 72.00, '{"bugs":1}', '{"critical":1,"attention":3}', 1, '[]'),
('anasouza', 'weekly', '2026-03-02', '2026-03-08', 1, 75.00, '{"bugs":1}', '{"attention":5}', 2, '[]'),
('anasouza', 'weekly', '2026-03-09', '2026-03-15', 1, 73.00, '{"bugs":1}', '{"critical":1,"attention":2}', 0, '[]'),
('anasouza', 'weekly', '2026-03-16', '2026-03-22', 1, 76.00, '{"smells":1}', '{"attention":4}', 3, '[]'),
('anasouza', 'weekly', '2026-03-23', '2026-03-29', 1, 74.00, '{"bugs":1}', '{"critical":1,"attention":3}', 0, '[]'),
('anasouza', 'weekly', '2026-03-30', '2026-04-05', 1, 78.00, '{"smells":1}', '{"attention":4}', 1, '[]'),

-- lucasmendes: 72 -> 48 -> 65 (dip and recovery)
('lucasmendes', 'weekly', '2026-01-05', '2026-01-11', 1, 72.00, '{"bugs":1,"architecture":1}', '{"critical":1,"attention":3}', 1, '[]'),
('lucasmendes', 'weekly', '2026-01-12', '2026-01-18', 1, 75.00, '{"bugs":1}', '{"critical":1,"attention":2}', 2, '[]'),
('lucasmendes', 'weekly', '2026-01-19', '2026-01-25', 1, 70.00, '{"bugs":1,"architecture":1}', '{"critical":2,"attention":2}', 0, '[]'),
('lucasmendes', 'weekly', '2026-01-26', '2026-02-01', 1, 65.00, '{"bugs":2,"security":1}', '{"critical":2,"attention":3}', 0, '[]'),
('lucasmendes', 'weekly', '2026-02-02', '2026-02-08', 1, 58.00, '{"bugs":2,"security":1}', '{"critical":3,"attention":3}', 0, '[]'),
('lucasmendes', 'weekly', '2026-02-09', '2026-02-15', 1, 52.00, '{"bugs":3,"security":1}', '{"critical":4,"attention":2}', 0, '[]'),
('lucasmendes', 'weekly', '2026-02-16', '2026-02-22', 1, 48.00, '{"bugs":3,"security":2}', '{"critical":4,"attention":3}', 0, '[]'),
('lucasmendes', 'weekly', '2026-02-23', '2026-03-01', 1, 55.00, '{"bugs":2,"security":1}', '{"critical":3,"attention":3}', 0, '[]'),
('lucasmendes', 'weekly', '2026-03-02', '2026-03-08', 1, 58.00, '{"bugs":2}', '{"critical":3,"attention":2}', 0, '[]'),
('lucasmendes', 'weekly', '2026-03-09', '2026-03-15', 1, 60.00, '{"bugs":1,"security":1}', '{"critical":2,"attention":4}', 0, '[]'),
('lucasmendes', 'weekly', '2026-03-16', '2026-03-22', 1, 63.00, '{"bugs":1}', '{"critical":2,"attention":3}', 0, '[]'),
('lucasmendes', 'weekly', '2026-03-23', '2026-03-29', 1, 62.00, '{"bugs":1,"smells":1}', '{"critical":2,"attention":4}', 0, '[]'),
('lucasmendes', 'weekly', '2026-03-30', '2026-04-05', 1, 65.00, '{"bugs":1}', '{"critical":2,"attention":3}', 0, '[]'),

-- pedroalves: 76 -> 87 (best performer, steady climb)
('pedroalves', 'weekly', '2026-01-05', '2026-01-11', 1, 76.00, '{"bugs":1}', '{"critical":1,"attention":3}', 1, '[]'),
('pedroalves', 'weekly', '2026-01-12', '2026-01-18', 1, 78.00, '{"bugs":1}', '{"critical":1,"attention":2}', 2, '[]'),
('pedroalves', 'weekly', '2026-01-19', '2026-01-25', 1, 80.00, '{"smells":1}', '{"attention":4}', 3, '[]'),
('pedroalves', 'weekly', '2026-01-26', '2026-02-01', 1, 79.00, '{"bugs":1}', '{"critical":1,"attention":2}', 0, '[]'),
('pedroalves', 'weekly', '2026-02-02', '2026-02-08', 1, 82.00, '{"smells":1}', '{"attention":3}', 1, '["consistent"]'),
('pedroalves', 'weekly', '2026-02-09', '2026-02-15', 1, 80.00, '{"bugs":1}', '{"attention":4}', 2, '["consistent"]'),
('pedroalves', 'weekly', '2026-02-16', '2026-02-22', 1, 84.00, '{}', '{"attention":3}', 3, '["consistent"]'),
('pedroalves', 'weekly', '2026-02-23', '2026-03-01', 1, 83.00, '{"smells":1}', '{"attention":3}', 4, '["consistent"]'),
('pedroalves', 'weekly', '2026-03-02', '2026-03-08', 1, 85.00, '{}', '{"attention":3}', 5, '["consistent","improver"]'),
('pedroalves', 'weekly', '2026-03-09', '2026-03-15', 1, 88.00, '{}', '{"attention":2}', 6, '["consistent","improver"]'),
('pedroalves', 'weekly', '2026-03-16', '2026-03-22', 1, 86.00, '{"smells":1}', '{"attention":2}', 7, '["consistent","improver"]'),
('pedroalves', 'weekly', '2026-03-23', '2026-03-29', 1, 90.00, '{}', '{"attention":2}', 8, '["consistent","improver","champion"]'),
('pedroalves', 'weekly', '2026-03-30', '2026-04-05', 1, 87.00, '{"smells":1}', '{"attention":2}', 0, '["consistent","improver"]'),

-- rafaelcosta: 68 -> 76 (stable with oscillation)
('rafaelcosta', 'weekly', '2026-01-05', '2026-01-11', 1, 68.00, '{"bugs":1,"security":1}', '{"critical":2,"attention":2}', 0, '[]'),
('rafaelcosta', 'weekly', '2026-01-12', '2026-01-18', 1, 72.00, '{"bugs":1}', '{"critical":1,"attention":3}', 1, '[]'),
('rafaelcosta', 'weekly', '2026-01-19', '2026-01-25', 1, 70.00, '{"bugs":1}', '{"critical":1,"attention":4}', 0, '[]'),
('rafaelcosta', 'weekly', '2026-01-26', '2026-02-01', 1, 74.00, '{"security":1}', '{"critical":1,"attention":3}', 1, '[]'),
('rafaelcosta', 'weekly', '2026-02-02', '2026-02-08', 1, 71.00, '{"bugs":1}', '{"critical":1,"attention":4}', 0, '[]'),
('rafaelcosta', 'weekly', '2026-02-09', '2026-02-15', 1, 73.00, '{"bugs":1}', '{"critical":1,"attention":3}', 1, '[]'),
('rafaelcosta', 'weekly', '2026-02-16', '2026-02-22', 1, 75.00, '{"smells":1}', '{"attention":5}', 2, '[]'),
('rafaelcosta', 'weekly', '2026-02-23', '2026-03-01', 1, 72.00, '{"bugs":1}', '{"critical":1,"attention":3}', 0, '[]'),
('rafaelcosta', 'weekly', '2026-03-02', '2026-03-08', 1, 76.00, '{}', '{"attention":4}', 1, '[]'),
('rafaelcosta', 'weekly', '2026-03-09', '2026-03-15', 1, 73.00, '{"bugs":1}', '{"critical":1,"attention":2}', 0, '[]'),
('rafaelcosta', 'weekly', '2026-03-16', '2026-03-22', 1, 75.00, '{"smells":1}', '{"attention":5}', 1, '[]'),
('rafaelcosta', 'weekly', '2026-03-23', '2026-03-29', 1, 74.00, '{"bugs":1}', '{"critical":1,"attention":3}', 0, '[]'),
('rafaelcosta', 'weekly', '2026-03-30', '2026-04-05', 1, 76.00, '{}', '{"attention":4}', 1, '[]');
