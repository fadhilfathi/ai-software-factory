-- Migration: 011_create_code_and_review_tables
-- Description: Create tables for code generation, versioning, and reviews

-- Code Generation Requests
CREATE TABLE IF NOT EXISTS code_generation_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    specification TEXT NOT NULL,
    files TEXT[] NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    execution_id UUID,
    output TEXT,
    estimated_time INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_code_gen_project_id ON code_generation_requests(project_id);
CREATE INDEX IF NOT EXISTS idx_code_gen_task_id ON code_generation_requests(task_id);
CREATE INDEX IF NOT EXISTS idx_code_gen_status ON code_generation_requests(status);

-- Commits (Code Versions)
CREATE TABLE IF NOT EXISTS commits (
    sha VARCHAR(40) PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    branch VARCHAR(255) NOT NULL DEFAULT 'main',
    message TEXT NOT NULL,
    author VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_commits_project_id ON commits(project_id);
CREATE INDEX IF NOT EXISTS idx_commits_author ON commits(author);

-- Commit Files
CREATE TABLE IF NOT EXISTS commit_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    commit_sha VARCHAR(40) NOT NULL REFERENCES commits(sha) ON DELETE CASCADE,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    UNIQUE(commit_sha, path)
);

CREATE INDEX IF NOT EXISTS idx_commit_files_sha ON commit_files(commit_sha);

-- Project Files (Current State)
CREATE TABLE IF NOT EXISTS project_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(50),
    size INT DEFAULT 0,
    last_modified TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    modified_by VARCHAR(255),
    UNIQUE(project_id, path)
);

CREATE INDEX IF NOT EXISTS idx_project_files_project_id ON project_files(project_id);

-- Reviews
CREATE TABLE IF NOT EXISTS reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    commit_sha VARCHAR(40) NOT NULL REFERENCES commits(sha) ON DELETE CASCADE,
    target_agent_id UUID,
    reviewer_type VARCHAR(50) NOT NULL, -- 'agent', 'user'
    reviewer_id UUID REFERENCES users(id) ON DELETE SET NULL,
    agent_id UUID REFERENCES agents(id) ON DELETE SET NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'in_progress',
    result VARCHAR(50), -- 'approved', 'changes_requested', 'rejected'
    score DECIMAL(3,1),
    metrics JSONB DEFAULT '{}', -- Complexity, TestCoverage, Duplications, etc.
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reviews_project_id ON reviews(project_id);
CREATE INDEX IF NOT EXISTS idx_reviews_commit_sha ON reviews(commit_sha);
CREATE INDEX IF NOT EXISTS idx_reviews_status ON reviews(status);

-- Review Issues
CREATE TABLE IF NOT EXISTS review_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id UUID NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    severity VARCHAR(50) NOT NULL,
    file TEXT NOT NULL,
    line INT NOT NULL,
    message TEXT NOT NULL,
    suggestion TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_review_issues_review_id ON review_issues(review_id);
