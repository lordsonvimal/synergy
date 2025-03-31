CREATE TABLE IF NOT EXISTS reminders (
    event_id UUID,
    reminder_time TIMESTAMP,
    message TEXT,
    created_at TIMESTAMP,
    PRIMARY KEY ((event_id), reminder_time)
) WITH CLUSTERING ORDER BY (reminder_time ASC);
