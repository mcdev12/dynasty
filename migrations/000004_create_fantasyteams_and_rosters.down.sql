-- Down migration for fantasy teams and roster slots

-- Drop indexes
DROP INDEX IF EXISTS idx_roster_slots_player;
DROP INDEX IF EXISTS idx_roster_slots_team_position;

-- Drop tables (order matters due to foreign keys)
DROP TABLE IF EXISTS roster_player;
DROP TABLE IF EXISTS fantasy_teams;

-- Drop enums
DROP TYPE IF EXISTS roster_position_enum;
DROP TYPE IF EXISTS acquisition_type_enum;