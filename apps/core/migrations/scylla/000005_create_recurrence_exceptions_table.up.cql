CREATE TABLE IF NOT EXISTS synergy.recurrence_exceptions (
    recurrence_id UUID,
    exception_date TIMESTAMP,
    exception_message TEXT,
    created_at TIMESTAMP,
    PRIMARY KEY (recurrence_id, exception_date)
) WITH CLUSTERING ORDER BY (exception_date ASC);
