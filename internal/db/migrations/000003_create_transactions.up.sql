
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE transactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    user_id UUID NOT NULL,
    transaction_type VARCHAR(20) NOT NULL,
    receiver_id UUID,
    amount NUMERIC(12,2) NOT NULL,

    device_id VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,

    transaction_status VARCHAR(20) NOT NULL
        CHECK (transaction_status IN ('PENDING', 'COMPLETED', 'FLAGGED', 'BLOCKED')),

    transaction_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_transactions_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_transactions_user_id ON transactions(user_id);
CREATE INDEX idx_transactions_transaction_time ON transactions(transaction_time);
CREATE INDEX idx_transactions_transaction_status
    ON transactions(transaction_status);
