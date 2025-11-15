BEGIN;

CREATE TABLE teams (
    team_name VARCHAR(255) PRIMARY KEY,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE users (
    user_id VARCHAR(255) PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    team_name VARCHAR(255) NOT NULL REFERENCES teams(team_name) ON DELETE CASCADE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE pull_requests (
    pull_request_id VARCHAR(255) PRIMARY KEY,
    pull_request_name VARCHAR(255) NOT NULL,
    author_id VARCHAR(255) NOT NULL REFERENCES users(user_id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
    created_at TIMESTAMP DEFAULT NOW(),
    merged_at TIMESTAMP NULL
);

CREATE TABLE pr_reviewers (
    pull_request_id VARCHAR(255) REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    user_id VARCHAR(255) REFERENCES users(user_id),
    assigned_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (pull_request_id, user_id)
);

-- Индексы для производительности при выборке активных юзеров команды и при поиске PR по ревьюверу
CREATE INDEX idx_users_team_active ON users(team_name, is_active);
CREATE INDEX idx_pr_reviewers_user ON pr_reviewers(user_id);


COMMIT;