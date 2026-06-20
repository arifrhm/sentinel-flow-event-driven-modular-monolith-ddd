CREATE TABLE IF NOT EXISTS events (
	event_id VARCHAR(255) PRIMARY KEY,
	user_id VARCHAR(255) NOT NULL,
	event_type VARCHAR(100) NOT NULL,
	ip_address VARCHAR(100),
	user_agent TEXT,
	payload JSONB,
	timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
	gdpr_consent BOOLEAN NOT NULL,
	is_fraudulent BOOLEAN NOT NULL,
	fraud_score DOUBLE PRECISION NOT NULL,
	fraud_reason TEXT,
	screened_at TIMESTAMP WITH TIME ZONE NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events(user_id);

CREATE TABLE IF NOT EXISTS crm_contacts (
	user_id VARCHAR(255) PRIMARY KEY,
	email VARCHAR(255) NOT NULL,
	workflow_triggered TEXT NOT NULL,
	sync_status VARCHAR(50) NOT NULL,
	synced_at TIMESTAMP WITH TIME ZONE NOT NULL,
	retry_count INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS privacy_logs (
	log_id VARCHAR(255) PRIMARY KEY,
	user_id VARCHAR(255) NOT NULL,
	action VARCHAR(100) NOT NULL,
	timestamp TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE IF NOT EXISTS metrics_counters (
	metric_name VARCHAR(100) PRIMARY KEY,
	counter_value BIGINT NOT NULL DEFAULT 0
);
