-- Seed Data: NBA Seasons (recent seasons for context)

INSERT INTO seasons (sport, season_year, season_type, start_date, end_date, is_active, total_games) VALUES
-- Current season (2025-26)
('basketball_nba', '2025-26', 'regular', '2025-10-21', '2026-04-12', true, 1230),

-- Recent completed seasons (for historical context)
('basketball_nba', '2024-25', 'regular', '2024-10-22', '2025-04-13', false, 1230),
('basketball_nba', '2023-24', 'regular', '2023-10-24', '2024-04-14', false, 1230),
('basketball_nba', '2022-23', 'regular', '2022-10-18', '2023-04-09', false, 1230),
('basketball_nba', '2021-22', 'regular', '2021-10-19', '2022-04-10', false, 1230),
('basketball_nba', '2020-21', 'regular', '2020-12-22', '2021-05-16', false, 1080),
('basketball_nba', '2019-20', 'regular', '2019-10-22', '2020-08-14', false, 1059)

ON CONFLICT (sport, season_year, season_type) DO UPDATE SET
    start_date = EXCLUDED.start_date,
    end_date = EXCLUDED.end_date,
    is_active = EXCLUDED.is_active,
    total_games = EXCLUDED.total_games,
    updated_at = NOW();
