CREATE TABLE seller_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    full_name VARCHAR NOT NULL,
    birth_date DATE NOT NULL,
    address TEXT NOT NULL,
    photo_present BOOLEAN NOT NULL,
    identity_document_front_key VARCHAR,
    identity_document_back_key VARCHAR,
    status VARCHAR NOT NULL DEFAULT 'PENDING',
    rejection_reason TEXT,
    reviewed_at TIMESTAMPTZ,
    reviewed_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seller_registrations_user_id ON seller_registrations (user_id);
CREATE INDEX idx_seller_registrations_status ON seller_registrations (status);
