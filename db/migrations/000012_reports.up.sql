CREATE TABLE reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    target_type VARCHAR(32) NOT NULL,
    target_id UUID NOT NULL,
    reason VARCHAR(500) NOT NULL,
    detail TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    reviewed_by UUID REFERENCES users (id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    resolution TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_reports_reporter_id ON reports (reporter_id);
CREATE INDEX idx_reports_target ON reports (target_type, target_id);
CREATE INDEX idx_reports_status ON reports (status);
CREATE UNIQUE INDEX idx_reports_open_unique ON reports (reporter_id, target_type, target_id)
    WHERE status IN ('PENDING', 'IN_REVIEW');
