-- Next deadline column represents the next deadline for a draft pick in a draft
ALTER TABLE draft ADD COLUMN next_deadline TIMESTAMPTZ;


-- Creates an index on the next deadline and in progress draft states.
CREATE INDEX idx_drafts_state_deadline
    ON draft(status, next_deadline)
    WHERE status = 'IN_PROGRESS';