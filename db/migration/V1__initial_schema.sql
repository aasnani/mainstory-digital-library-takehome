CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'MEMBER',
    CONSTRAINT users_role_check CHECK (role IN ('MEMBER', 'LIBRARIAN', 'ADMIN'))
);

CREATE TABLE IF NOT EXISTS books (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    author VARCHAR(255) NOT NULL DEFAULT '',
    genre VARCHAR(100) NOT NULL DEFAULT '',
    is_fiction BOOLEAN NOT NULL DEFAULT TRUE,
    published_date DATE,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    language VARCHAR(16) NOT NULL DEFAULT 'en',
    price_cents INT NOT NULL,
    content TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS entitlements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    book_id UUID REFERENCES books(id),
    type VARCHAR(50) NOT NULL, -- 'SINGLE_PURCHASE', 'SUBSCRIPTION'
    status VARCHAR(50) NOT NULL, -- 'ACTIVE', 'CANCELLED', 'PAST_DUE'
    ends_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT entitlements_type_check CHECK (type IN ('SINGLE_PURCHASE', 'SUBSCRIPTION')),
    CONSTRAINT entitlements_status_check CHECK (status IN ('ACTIVE', 'CANCELLED', 'PAST_DUE')),
    CONSTRAINT entitlements_type_book_shape CHECK (
        (type = 'SINGLE_PURCHASE' AND book_id IS NOT NULL)
        OR (type = 'SUBSCRIPTION' AND book_id IS NULL)
    )
);

-- Idempotent purchases: at most one SINGLE_PURCHASE row per (user_id, book_id)
CREATE UNIQUE INDEX IF NOT EXISTS entitlements_unique_single_purchase_per_book
    ON entitlements (user_id, book_id)
    WHERE type = 'SINGLE_PURCHASE';

-- One current subscription per user (MVP): at most one ACTIVE SUBSCRIPTION per user_id
CREATE UNIQUE INDEX IF NOT EXISTS entitlements_unique_active_subscription_per_user
    ON entitlements (user_id)
    WHERE type = 'SUBSCRIPTION' AND status = 'ACTIVE';
