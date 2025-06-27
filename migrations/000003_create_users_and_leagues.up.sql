-- Users
CREATE TABLE users
(
    id         UUID PRIMARY KEY,
    username   TEXT UNIQUE NOT NULL,
    email      TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- enum for league type
CREATE TYPE league_type AS ENUM ('REDRAFT', 'KEEPER', 'DYNASTY');

-- leagues table
CREATE TABLE leagues
(
    id              UUID PRIMARY KEY      DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL,
    sport_id        VARCHAR(20)  NOT NULL REFERENCES sports (id),
    league_type     league_type  NOT NULL,
    commissioner_id UUID         NOT NULL REFERENCES users (id),
    league_settings JSONB        NOT NULL, -- remaining flexible settings
    status          VARCHAR(20)  NOT NULL, -- e.g. 'pending', 'active', 'completed'
    season          VARCHAR(10)  NOT NULL, -- e.g. '2025'
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);