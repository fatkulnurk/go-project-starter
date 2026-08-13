CREATE TABLE IF NOT EXISTS users (
    id                VARCHAR(64)  NOT NULL PRIMARY KEY,
    name              VARCHAR(255) NOT NULL,
    email             VARCHAR(255) NULL,
    phone             VARCHAR(32)  NULL,
    password_hash     VARCHAR(255) NOT NULL,
    email_verified_at TIMESTAMP    NULL,
    phone_verified_at TIMESTAMP    NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (email),
    UNIQUE (phone)
);
