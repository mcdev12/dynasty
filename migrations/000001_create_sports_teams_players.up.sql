CREATE TABLE sports
(
    id         TEXT PRIMARY KEY,            -- e.g. 'nfl', 'nba'
    name       TEXT        NOT NULL,        -- 'National Football League'
    plugin_key TEXT        NOT NULL UNIQUE, -- matches your Go plugin key
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2) Seed it with your default rows
INSERT INTO sports (id, name, plugin_key, created_at)
VALUES ('nfl', 'National Football League', 'nfl', NOW()),
       ('nba', 'National Basketball Assoc.', 'nba', NOW())
ON CONFLICT (id) DO UPDATE
    SET name       = EXCLUDED.name,
        plugin_key = EXCLUDED.plugin_key;

CREATE TABLE teams
(
    id               UUID PRIMARY KEY,
    sport_id         TEXT        NOT NULL REFERENCES sports (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    external_id      TEXT        NOT NULL, -- ID from external API
    name             TEXT        NOT NULL, -- e.g. 'Las Vegas Raiders'
    code             TEXT        NOT NULL, -- e.g. 'LV'
    city             TEXT        NOT NULL, -- 'Las Vegas'
    coach            TEXT,                 -- 'Antonio Pierce (interim)'
    owner            TEXT,                 -- 'Carol and Mark Davis'
    stadium          TEXT,                 -- 'Allegiant Stadium'
    established_year INT,                  -- 1960
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (sport_id, external_id)
);

CREATE TABLE players
(
    id          UUID PRIMARY KEY,
    sport_id    TEXT        NOT NULL REFERENCES sports (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    external_id TEXT        NOT NULL, -- ID from external API
    full_name   TEXT        NOT NULL, -- 'Derek Carr'
    team_id     UUID        REFERENCES teams (id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (sport_id, external_id)
);

