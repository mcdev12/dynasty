-- NFL-specific profile
CREATE TABLE nfl_player_profiles
(
    player_id     UUID PRIMARY KEY REFERENCES players (id),
    height_cm     INT,
    weight_kg     INT,
    group_role    TEXT,     -- 'Offense'
    position      TEXT,     -- 'QB'
    age           INT,      -- 31
    height_desc   TEXT,     -- '6'' 3"'
    weight_desc   TEXT,     -- '210 lbs'
    college       TEXT,     -- 'Fresno State'
    jersey_number SMALLINT, -- 4
    salary_desc   TEXT,     -- '$19,375,000'
    experience    SMALLINT  -- 9
);