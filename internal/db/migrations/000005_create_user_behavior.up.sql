CREATE TABLE user_behavior (
    user_id UUID PRIMARY KEY,

    -- Transaction volume
    total_transactions BIGINT DEFAULT 0,

    -- Amount behavior (long-term)
    avg_transaction_amount NUMERIC(18,2) DEFAULT 0,
    amount_variance NUMERIC(18,4) DEFAULT 0,
    amount_std_dev NUMERIC(18,2) DEFAULT 0,

    -- Incremental variance accumulator (Welford)
    amount_variance_acc NUMERIC(18,4) DEFAULT 0,

    -- Short-term adaptive behavior
    recent_avg_amount NUMERIC(18,2) DEFAULT 0,
    ema_smoothing_factor NUMERIC(5,4) DEFAULT 0.1,

    -- Recent transaction tracking
    last_transaction_amount NUMERIC(18,2),
    last_transaction_time TIMESTAMP,

    -- Upper boundary behavior
    high_value_threshold NUMERIC(18,2), -- p95

    updated_at TIMESTAMP DEFAULT now(),

    CONSTRAINT fk_behavior_user
        FOREIGN KEY (user_id) REFERENCES users(id)
);
