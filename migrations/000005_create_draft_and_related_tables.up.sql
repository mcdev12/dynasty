CREATE TYPE draft_type AS ENUM ('SNAKE', 'AUCTION', 'ROOKIE');
CREATE TYPE draft_status AS ENUM ('NOT_STARTED', 'IN_PROGRESS', 'PAUSED', 'COMPLETED', 'CANCELLED');

CREATE TABLE draft
(
    id           UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    league_id    UUID         NOT NULL REFERENCES leagues (id),
    draft_type   draft_type   NOT NULL,
    status       draft_status NOT NULL DEFAULT 'NOT_STARTED',
    settings     JSONB        NOT NULL, -- DraftSettings JSON (rounds, time_per_pick, etc.)
    scheduled_at TIMESTAMPTZ,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE draft_picks
(
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    draft_id       UUID    NOT NULL REFERENCES draft (id),
    round          INTEGER NOT NULL,
    pick           INTEGER NOT NULL,
    overall_pick   INTEGER NOT NULL,
    team_id        UUID    NOT NULL REFERENCES fantasy_teams (id),
    player_id      UUID,                           -- filled when picked
    picked_at      TIMESTAMPTZ,
    auction_amount DECIMAL,                        -- for auction drafts
    keeper_pick    BOOLEAN          DEFAULT FALSE, -- flag indicating that a specific draft pick was used on a keeper player rather than a fresh selection during the draft.
    UNIQUE (draft_id, overall_pick)
);