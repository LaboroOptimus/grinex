CREATE TABLE IF NOT EXISTS rates (
    id BIGSERIAL PRIMARY KEY,
    ask NUMERIC(20, 8) NOT NULL,
    bid NUMERIC(20, 8) NOT NULL,
    calculation_type TEXT NOT NULL,
    n INTEGER NOT NULL,
    m INTEGER NOT NULL,
    "timestamp" TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_rates_timestamp ON rates ("timestamp");
