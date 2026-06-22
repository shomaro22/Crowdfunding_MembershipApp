CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL REFERENCES users (id) ON DELETE RESTRICT,
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    target_amount BIGINT NOT NULL DEFAULT 0,
    current_amount BIGINT NOT NULL DEFAULT 0,
    supporter_count INT NOT NULL DEFAULT 0,
    duration_type VARCHAR(32) NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    main_image_url VARCHAR,
    delete_requested BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_projects_seller_id ON projects (seller_id);
CREATE INDEX idx_projects_status ON projects (status);
CREATE INDEX idx_projects_public_list ON projects (status, duration_type, end_date, created_at);

CREATE TABLE project_plans (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    membership_type VARCHAR(32) NOT NULL,
    price INT NOT NULL,
    max_supporters INT,
    benefits TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    supporter_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, membership_type)
);

CREATE INDEX idx_project_plans_project_id ON project_plans (project_id);

CREATE TABLE project_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    image_url VARCHAR NOT NULL,
    is_main BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_project_images_project_id ON project_images (project_id);
