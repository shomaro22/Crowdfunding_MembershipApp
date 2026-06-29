CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    admin_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    action VARCHAR(128) NOT NULL,
    resource VARCHAR(64) NOT NULL,
    resource_id VARCHAR(128) NOT NULL,
    details TEXT,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_admin_created ON audit_logs (admin_id, created_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs (resource, resource_id);

CREATE TABLE review_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reviewer_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    resource_type VARCHAR(64) NOT NULL,
    resource_id UUID NOT NULL,
    decision VARCHAR(32) NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_review_decisions_resource ON review_decisions (resource_type, resource_id);
