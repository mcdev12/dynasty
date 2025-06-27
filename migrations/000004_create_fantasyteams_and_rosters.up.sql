CREATE TABLE fantasy_teams
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    league_id  UUID        NOT NULL REFERENCES leagues (id),
    owner_id   UUID        NOT NULL REFERENCES users (id),
    name       TEXT        NOT NULL,
    logo_url   TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (league_id, owner_id)
);

-- Add enum for acquisition types
CREATE TYPE acquisition_type_enum AS ENUM ('DRAFT', 'WAIVER', 'FREE_AGENT', 'TRADE', 'KEEPER');

-- Add enum for roster positions if needed
CREATE TYPE roster_position_enum AS ENUM ('STARTING', 'BENCH', 'IR', 'TAXI');

-- Roster slots (static assignment)
CREATE TABLE roster_slots
(
    id               UUID PRIMARY KEY               DEFAULT gen_random_uuid(),
    fantasy_team_id  UUID                  NOT NULL REFERENCES fantasy_teams (id),
    player_id        UUID                  NOT NULL REFERENCES players (id),
    position         roster_position_enum  NOT NULL DEFAULT 'BENCH',
    acquired_at      TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    acquisition_type acquisition_type_enum NOT NULL,
    keeper_data      JSONB,
    UNIQUE (fantasy_team_id, player_id)
);

-- Index for common queries
CREATE INDEX idx_roster_slots_team_position ON roster_slots (fantasy_team_id, position);
CREATE INDEX idx_roster_slots_player ON roster_slots (player_id);