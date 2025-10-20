CREATE TABLE IF NOT EXISTS processed_incidents (
    id VARCHAR(255) NOT NULL PRIMARY KEY,
    blockchain VARCHAR(50) NOT NULL,
    incident_type VARCHAR(50) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'completed',
    rollback_from_block BIGINT,
    rollback_to_block BIGINT,
    error_message TEXT,
    started_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_incidents_blockchain ON processed_incidents(blockchain);
CREATE INDEX IF NOT EXISTS idx_processed_incidents_created_at ON processed_incidents(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_processed_incidents_status ON processed_incidents(blockchain, status);
