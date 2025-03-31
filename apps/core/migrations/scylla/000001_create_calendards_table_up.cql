CREATE TABLE calendars (
    calendar_id UUID,
    user_id INT,
    is_default BOOLEAN,
    name TEXT,
    description TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY ((user_id), calendar_id)  -- Partition by user_id, cluster by calendar_id
) WITH CLUSTERING ORDER BY (calendar_id ASC);
