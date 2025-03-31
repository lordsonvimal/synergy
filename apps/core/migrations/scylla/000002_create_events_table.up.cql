CREATE TABLE IF NOT EXISTS events (
    calendar_id UUID,
    event_id UUID,
    title TEXT,
    description TEXT,
    location TEXT,
    start_time TIMESTAMP,
    end_time TIMESTAMP,
    is_all_day BOOLEAN,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY ((calendar_id), start_time, event_id)
) WITH CLUSTERING ORDER BY (start_time ASC, event_id ASC);
