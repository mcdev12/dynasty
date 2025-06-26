-- 1. Drop players (depends on teams and sports)
DROP TABLE IF EXISTS players;

-- 2. Drop teams (depends on sports)
DROP TABLE IF EXISTS teams;

-- 3. Drop sports
DELETE FROM sports WHERE id IN ('nfl','nba');
DROP TABLE IF EXISTS sports;