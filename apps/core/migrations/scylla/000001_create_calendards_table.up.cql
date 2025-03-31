CREATE TABLE synergy.calendars (
    calendar_id UUID,
    user_id INT,
    is_default BOOLEAN,
    name TEXT,
    description TEXT,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    PRIMARY KEY ((user_id), calendar_id)
) WITH CLUSTERING ORDER BY (calendar_id ASC);
