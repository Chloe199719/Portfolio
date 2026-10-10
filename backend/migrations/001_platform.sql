CREATE TABLE IF NOT EXISTS content(
    id TEXT NOT NULL,
    type TEXT NOT NULL,
    state TEXT NOT NULL CHECK(state IN ('draft', 'published')),
    document JSONB NOT NULL,
    revision TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(id, state)
);
CREATE UNIQUE INDEX IF NOT EXISTS content_published_slug ON content(type,(document->'slug'->>'current'))
WHERE state = 'published'
    AND type IN ('projects', 'note');
CREATE TABLE IF NOT EXISTS guestbook(
    id UUID PRIMARY KEY,
    author_uid TEXT NOT NULL,
    name TEXT NOT NULL,
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'approved', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS messages(
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    subject TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT false
);
CREATE TABLE IF NOT EXISTS rate_limits(
    key TEXT PRIMARY KEY,
    count INTEGER NOT NULL,
    reset_at BIGINT NOT NULL
);
CREATE TABLE IF NOT EXISTS meta(key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE content_revisions(
    id UUID PRIMARY KEY,
    content_id TEXT NOT NULL,
    document JSONB NOT NULL,
    action TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX content_history ON content_revisions(content_id, created_at DESC);
CREATE TABLE content_trash(
    id TEXT PRIMARY KEY,
    documents JSONB NOT NULL,
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE assets(
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL UNIQUE,
    alt TEXT NOT NULL DEFAULT '',
    caption TEXT NOT NULL DEFAULT '',
    width INT NOT NULL,
    height INT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE asset_references(
    asset_id TEXT NOT NULL,
    content_id TEXT NOT NULL,
    state TEXT NOT NULL,
    PRIMARY KEY(asset_id, content_id, state),
    FOREIGN KEY(content_id, state) REFERENCES content(id, state) ON DELETE CASCADE
);
CREATE TABLE schedules(
    id UUID PRIMARY KEY,
    content_id TEXT NOT NULL,
    revision_id UUID NOT NULL REFERENCES content_revisions(id),
    run_at TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(
        status IN ('pending', 'published', 'cancelled', 'failed')
    ),
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX one_pending_schedule ON schedules(content_id)
WHERE status = 'pending';
CREATE TABLE accounts(
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL DEFAULT '',
    verified BOOLEAN NOT NULL DEFAULT false,
    owner BOOLEAN NOT NULL DEFAULT false,
    totp_secret TEXT NOT NULL DEFAULT '',
    totp_last BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX one_owner ON accounts(owner)
WHERE owner;
CREATE TABLE sessions(
    hash TEXT PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    kind TEXT NOT NULL,
    mfa BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    agent TEXT NOT NULL DEFAULT ''
);
CREATE INDEX sessions_account ON sessions(account_id);
CREATE TABLE one_time_tokens(
    hash TEXT PRIMARY KEY,
    purpose TEXT NOT NULL,
    account_id UUID REFERENCES accounts(id) ON DELETE CASCADE,
    payload JSONB NOT NULL DEFAULT '{}',
    expires_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE social_accounts(
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    PRIMARY KEY(provider, subject),
    UNIQUE(provider, account_id)
);
CREATE TABLE passkeys(
    id TEXT PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    credential JSONB NOT NULL,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE oauth_clients(
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    document JSONB NOT NULL,
    origins JSONB NOT NULL DEFAULT '[]',
    disabled BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE oauth_sessions(
    kind TEXT NOT NULL,
    signature TEXT NOT NULL,
    request_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    client_id TEXT NOT NULL,
    document JSONB NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY(kind, signature)
);
CREATE INDEX oauth_request ON oauth_sessions(request_id);
CREATE TABLE oauth_consents(
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    client_id TEXT NOT NULL REFERENCES oauth_clients(id),
    scopes JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(account_id, client_id)
);
CREATE TABLE email_jobs(
    id UUID PRIMARY KEY,
    recipient TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE audit_events(
    id BIGSERIAL PRIMARY KEY,
    actor TEXT NOT NULL,
    event TEXT NOT NULL,
    target TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE legacy_ownership_map(
    legacy_uid TEXT PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id),
    mapped_at TIMESTAMPTZ NOT NULL DEFAULT now()
);