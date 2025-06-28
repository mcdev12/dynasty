-- Outbox table: one row per domain event
CREATE TABLE draft_outbox
(
    id         UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    draft_id   UUID        NOT NULL REFERENCES draft (id),
    event_type TEXT        NOT NULL, -- e.g. 'PickMade', 'DraftStarted'
    payload    JSONB       NOT NULL, -- complete event body
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    sent_at    TIMESTAMPTZ           -- NULL = not forwarded yet
);

-- indexes
-- Fast “unsent queue” scan for the relay process
CREATE INDEX draft_outbox_unsent_idx
    ON draft_outbox (sent_at)
    WHERE sent_at IS NULL;

-- If you’ll often query/replay by draft_id
CREATE INDEX draft_outbox_draft_idx
    ON draft_outbox (draft_id, created_at);
