-- Create a function that sends a notification when a new event is inserted into the outbox
CREATE OR REPLACE FUNCTION notify_outbox_event() RETURNS TRIGGER AS $$
BEGIN
    -- Send notification with the event ID
    -- Using 'draft_outbox_events' as the channel name
    PERFORM pg_notify('draft_outbox_events', NEW.id::text);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger that fires after insert on draft_outbox
CREATE TRIGGER draft_outbox_notify_trigger
AFTER INSERT ON draft_outbox
FOR EACH ROW
EXECUTE FUNCTION notify_outbox_event();

-- Add index for efficient queries on sent_at for fallback polling
CREATE INDEX IF NOT EXISTS idx_draft_outbox_sent_at 
ON draft_outbox(sent_at) 
WHERE sent_at IS NULL;