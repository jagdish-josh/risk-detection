CREATE TABLE user_security (
    user_id UUID PRIMARY KEY,
    device_id VARCHAR(255) NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_user_security_user
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE
);
