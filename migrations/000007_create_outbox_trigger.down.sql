-- Drop the trigger
DROP TRIGGER IF EXISTS draft_outbox_notify_trigger ON draft_outbox;

-- Drop the function
DROP FUNCTION IF EXISTS notify_outbox_event();

-- Drop the index
DROP INDEX IF EXISTS idx_draft_outbox_sent_at;