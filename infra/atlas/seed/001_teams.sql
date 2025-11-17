-- Seed Data: NBA Teams (30 franchises)
-- All 30 permanent NBA teams with standard abbreviations

INSERT INTO teams (sport, abbreviation, short_name, full_name, city, conference, division, external_id, is_active) VALUES
-- Eastern Conference - Atlantic Division
('basketball_nba', 'BOS', 'Celtics', 'Boston Celtics', 'Boston', 'East', 'Atlantic', '2', true),
('basketball_nba', 'BKN', 'Nets', 'Brooklyn Nets', 'Brooklyn', 'East', 'Atlantic', '17', true),
('basketball_nba', 'NYK', 'Knicks', 'New York Knicks', 'New York', 'East', 'Atlantic', '18', true),
('basketball_nba', 'PHI', '76ers', 'Philadelphia 76ers', 'Philadelphia', 'East', 'Atlantic', '20', true),
('basketball_nba', 'TOR', 'Raptors', 'Toronto Raptors', 'Toronto', 'East', 'Atlantic', '28', true),

-- Eastern Conference - Central Division
('basketball_nba', 'CHI', 'Bulls', 'Chicago Bulls', 'Chicago', 'East', 'Central', '4', true),
('basketball_nba', 'CLE', 'Cavaliers', 'Cleveland Cavaliers', 'Cleveland', 'East', 'Central', '5', true),
('basketball_nba', 'DET', 'Pistons', 'Detroit Pistons', 'Detroit', 'East', 'Central', '8', true),
('basketball_nba', 'IND', 'Pacers', 'Indiana Pacers', 'Indiana', 'East', 'Central', '11', true),
('basketball_nba', 'MIL', 'Bucks', 'Milwaukee Bucks', 'Milwaukee', 'East', 'Central', '15', true),

-- Eastern Conference - Southeast Division
('basketball_nba', 'ATL', 'Hawks', 'Atlanta Hawks', 'Atlanta', 'East', 'Southeast', '1', true),
('basketball_nba', 'CHA', 'Hornets', 'Charlotte Hornets', 'Charlotte', 'East', 'Southeast', '30', true),
('basketball_nba', 'MIA', 'Heat', 'Miami Heat', 'Miami', 'East', 'Southeast', '14', true),
('basketball_nba', 'ORL', 'Magic', 'Orlando Magic', 'Orlando', 'East', 'Southeast', '19', true),
('basketball_nba', 'WAS', 'Wizards', 'Washington Wizards', 'Washington', 'East', 'Southeast', '27', true),

-- Western Conference - Northwest Division
('basketball_nba', 'DEN', 'Nuggets', 'Denver Nuggets', 'Denver', 'West', 'Northwest', '7', true),
('basketball_nba', 'MIN', 'Timberwolves', 'Minnesota Timberwolves', 'Minnesota', 'West', 'Northwest', '16', true),
('basketball_nba', 'OKC', 'Thunder', 'Oklahoma City Thunder', 'Oklahoma City', 'West', 'Northwest', '25', true),
('basketball_nba', 'POR', 'Trail Blazers', 'Portland Trail Blazers', 'Portland', 'West', 'Northwest', '22', true),
('basketball_nba', 'UTA', 'Jazz', 'Utah Jazz', 'Utah', 'West', 'Northwest', '26', true),

-- Western Conference - Pacific Division
('basketball_nba', 'GSW', 'Warriors', 'Golden State Warriors', 'Golden State', 'West', 'Pacific', '9', true),
('basketball_nba', 'LAC', 'Clippers', 'LA Clippers', 'Los Angeles', 'West', 'Pacific', '12', true),
('basketball_nba', 'LAL', 'Lakers', 'Los Angeles Lakers', 'Los Angeles', 'West', 'Pacific', '13', true),
('basketball_nba', 'PHX', 'Suns', 'Phoenix Suns', 'Phoenix', 'West', 'Pacific', '21', true),
('basketball_nba', 'SAC', 'Kings', 'Sacramento Kings', 'Sacramento', 'West', 'Pacific', '23', true),

-- Western Conference - Southwest Division
('basketball_nba', 'DAL', 'Mavericks', 'Dallas Mavericks', 'Dallas', 'West', 'Southwest', '6', true),
('basketball_nba', 'HOU', 'Rockets', 'Houston Rockets', 'Houston', 'West', 'Southwest', '10', true),
('basketball_nba', 'MEM', 'Grizzlies', 'Memphis Grizzlies', 'Memphis', 'West', 'Southwest', '29', true),
('basketball_nba', 'NOP', 'Pelicans', 'New Orleans Pelicans', 'New Orleans', 'West', 'Southwest', '3', true),
('basketball_nba', 'SAS', 'Spurs', 'San Antonio Spurs', 'San Antonio', 'West', 'Southwest', '24', true);
