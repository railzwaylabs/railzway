CREATE TABLE event_consumer_offsets (
    consumer_id TEXT PRIMARY KEY,
    last_event_id BIGINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
