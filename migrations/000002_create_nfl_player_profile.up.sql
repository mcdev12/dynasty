-- NFL-specific profile
CREATE TABLE nfl_player_profiles
(
    player_id     UUID PRIMARY KEY REFERENCES players (id),
    position      TEXT,     -- 'QB'
    status        TEXT,     -- Status of a player ACT (Active), DUP (Duplicate Profile), EXE (Exempt), FRES (Reserve/Future), IR (Injured Reserve), IRD (Injured Reserve - Designated for Return), NON (Non-football related injured reserve), NWT (Not with team), PRA(Practice Squad), PRA_IR (Practice Squad Injured Reserve), PUP (Physically unable to perform), RET (Retired), SUS (Suspended), UDF (Unsigned draft pick), UFA (Unsigned free agent)
    college       TEXT,     -- 'Fresno State'
    jersey_number SMALLINT, -- 4
    experience    SMALLINT, -- 9
    birth_date    DATE,     -- 1995-09-17
    height_cm     INT,
    weight_kg     INT,
    height_desc   TEXT,     -- '6'' 3"'
    weight_desc   TEXT      -- '210 lbs'

    --- age, salary, group = offense/defense?

);