
CREATE TABLE IF NOT EXISTS recurrence (
    calendar_id UUID,
    recurrence_id UUID,
    recurrence_rule TEXT,
    recurrence_start TIMESTAMP,
    recurrence_end TIMESTAMP,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY (calendar_id, recurrence_id)
) WITH CLUSTERING ORDER BY (recurrence_id ASC);
