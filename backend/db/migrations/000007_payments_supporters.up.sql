CREATE TABLE connect_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    stripe_account_id VARCHAR NOT NULL UNIQUE,
    charges_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    payouts_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    details_submitted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE project_supporters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    plan_id UUID NOT NULL REFERENCES project_plans (id) ON DELETE RESTRICT,
    status VARCHAR(32) NOT NULL DEFAULT 'ACTIVE',
    billing_mode VARCHAR(32) NOT NULL,
    current_period_start TIMESTAMPTZ NOT NULL,
    current_period_end TIMESTAMPTZ NOT NULL,
    pending_plan_id UUID REFERENCES project_plans (id) ON DELETE SET NULL,
    plan_change_date TIMESTAMPTZ,
    stripe_customer_id VARCHAR,
    stripe_subscription_id VARCHAR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_supporter_project_user ON project_supporters (project_id, user_id);
CREATE INDEX idx_supporters_status_period_end ON project_supporters (status, current_period_end);
CREATE INDEX idx_supporters_pending_plan ON project_supporters (plan_change_date)
    WHERE pending_plan_id IS NOT NULL;

-- §5.3.1: 同一 user × project で ACTIVE は 1 件のみ
CREATE UNIQUE INDEX idx_supporter_one_active_per_project_user
    ON project_supporters (project_id, user_id)
    WHERE status = 'ACTIVE';

CREATE TABLE purchases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE RESTRICT,
    supporter_id UUID REFERENCES project_supporters (id) ON DELETE SET NULL,
    buyer_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    quantity INT NOT NULL DEFAULT 1,
    total_price INT NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    payment_method VARCHAR(32) NOT NULL,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    stripe_invoice_id VARCHAR,
    stripe_payment_intent_id VARCHAR UNIQUE,
    stripe_checkout_session_id VARCHAR UNIQUE,
    idempotency_key VARCHAR UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchases_project_id ON purchases (project_id);
CREATE INDEX idx_purchases_buyer_id ON purchases (buyer_id);
CREATE INDEX idx_purchases_supporter_id ON purchases (supporter_id);
CREATE INDEX idx_purchases_status ON purchases (status);

CREATE TABLE purchase_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purchase_id UUID NOT NULL REFERENCES purchases (id) ON DELETE CASCADE,
    from_status VARCHAR(32),
    to_status VARCHAR(32) NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_purchase_events_purchase_id ON purchase_events (purchase_id);

CREATE TABLE stripe_webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stripe_event_id VARCHAR NOT NULL UNIQUE,
    event_type VARCHAR(128) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL,
    payload_hash VARCHAR(64)
);

CREATE INDEX idx_stripe_webhook_events_type ON stripe_webhook_events (event_type);

CREATE TABLE batch_job_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_name VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    processed INT NOT NULL DEFAULT 0,
    failed INT NOT NULL DEFAULT 0,
    error_message TEXT,
    triggered_by VARCHAR(64) NOT NULL,
    dry_run BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX idx_batch_job_runs_job_started ON batch_job_runs (job_name, started_at);
