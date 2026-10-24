-- +goose NO TRANSACTION
-- +goose Up
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;

CREATE TABLE IF NOT EXISTS audit_logs (
    seq_id      INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp   TEXT NOT NULL,
    profile_id  TEXT NOT NULL,
    action      TEXT NOT NULL,
    actor       TEXT NOT NULL,
    details     TEXT NOT NULL,
    prev_hash   TEXT NOT NULL,
    hash        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_audit_profile_id ON audit_logs(profile_id);
CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);

-- SQLite Triggers enforcing append-only immutability
-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS prevent_audit_update
BEFORE UPDATE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'Audit ledger entries are immutable and cannot be updated');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER IF NOT EXISTS prevent_audit_delete
BEFORE DELETE ON audit_logs
BEGIN
    SELECT RAISE(ABORT, 'Audit ledger entries are immutable and cannot be deleted');
END;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS prevent_audit_delete;
DROP TRIGGER IF EXISTS prevent_audit_update;
DROP INDEX IF EXISTS idx_audit_timestamp;
DROP INDEX IF EXISTS idx_audit_action;
DROP INDEX IF EXISTS idx_audit_profile_id;
DROP TABLE IF EXISTS audit_logs;


