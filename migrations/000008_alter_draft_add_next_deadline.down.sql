-- DORP index
DROP index IF EXISTS draft_next_deadline_idx;

ALTER TABLE draft DROP COLUMN next_deadline;