-- +goose Up
CREATE TABLE accounts (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL
);
CREATE TABLE sessions (
    token_hash TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL
);
CREATE INDEX sessions_expiry ON sessions(expires_at);
ALTER TABLE notes ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE notes ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0;
ALTER TABLE notes ADD COLUMN archived INTEGER NOT NULL DEFAULT 0;
ALTER TABLE notes ADD COLUMN trashed INTEGER NOT NULL DEFAULT 0;
ALTER TABLE notes ADD COLUMN version INTEGER NOT NULL DEFAULT 1;
CREATE INDEX notes_owner_updated ON notes(user_id, updated_at DESC, id);

-- +goose Down
DROP INDEX notes_owner_updated;
ALTER TABLE notes DROP COLUMN version;
ALTER TABLE notes DROP COLUMN trashed;
ALTER TABLE notes DROP COLUMN archived;
ALTER TABLE notes DROP COLUMN pinned;
ALTER TABLE notes DROP COLUMN title;
DROP TABLE sessions;
DROP TABLE accounts;
