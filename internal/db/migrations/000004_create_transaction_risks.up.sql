CREATE TABLE transaction_risks (
    transaction_id UUID PRIMARY KEY,
    risk_score INT NOT NULL,
    risk_level VARCHAR(20) NOT NULL,
    decision VARCHAR(10) NOT NULL CHECK (decision IN ('ALLOW','FLAG','BLOCK')),
    evaluated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_transaction_risk_transaction
        FOREIGN KEY (transaction_id)
        REFERENCES transactions(id)
        ON DELETE CASCADE
);
