CREATE TABLE IF NOT EXISTS users
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    avatar_url TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    created_by_names VARCHAR(255),
    updated_by VARCHAR(255),
    updated_by_names VARCHAR(255),
    deleted_at TIMESTAMP DEFAULT NULL
);

CREATE INDEX
IF NOT EXISTS idx_users_email     ON users (email);
CREATE INDEX
IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);