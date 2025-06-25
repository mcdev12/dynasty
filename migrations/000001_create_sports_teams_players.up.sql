CREATE TABLE sports
(
    id         TEXT PRIMARY KEY,            -- e.g. 'nfl', 'nba'
    name       TEXT        NOT NULL,        -- 'National Football League'
    plugin_key TEXT        NOT NULL UNIQUE, -- matches your Go plugin key
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

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
    id            UUID PRIMARY KEY,
    sport_id      TEXT        NOT NULL REFERENCES sports (id) ON DELETE RESTRICT ON UPDATE CASCADE,
    external_id   TEXT        NOT NULL, -- ID from external API
    full_name     TEXT        NOT NULL, -- 'Derek Carr'
    age           INT,                  -- 31
    height_desc   TEXT,                 -- '6'' 3"'
    weight_desc   TEXT,                 -- '210 lbs'
    college       TEXT,                 -- 'Fresno State'
    group_role    TEXT,                 -- 'Offense'
    position      TEXT,                 -- 'QB'
    jersey_number SMALLINT,             -- 4
    salary_desc   TEXT,                 -- '$19,375,000'
    experience    SMALLINT,             -- 9
    team_id       UUID        REFERENCES teams (id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (sport_id, external_id)
);
