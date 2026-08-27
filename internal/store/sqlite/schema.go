package sqlite

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaVersion = 1

var schemaStatements = []string{
	`CREATE TABLE IF NOT EXISTS schema_metadata (key TEXT PRIMARY KEY, value INTEGER NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS packages (package_id TEXT PRIMARY KEY, version INTEGER NOT NULL, status TEXT NOT NULL, updated_at TEXT NOT NULL, body BLOB NOT NULL)`,
	`CREATE TABLE IF NOT EXISTS revisions (revision_id TEXT PRIMARY KEY, package_id TEXT NOT NULL, submitted_at TEXT NOT NULL, body BLOB NOT NULL, FOREIGN KEY(package_id) REFERENCES packages(package_id))`,
	`CREATE INDEX IF NOT EXISTS revisions_package_idx ON revisions(package_id, submitted_at, revision_id)`,
	`CREATE TABLE IF NOT EXISTS findings (finding_id TEXT PRIMARY KEY, revision_id TEXT NOT NULL, status TEXT NOT NULL, body BLOB NOT NULL, FOREIGN KEY(revision_id) REFERENCES revisions(revision_id))`,
	`CREATE INDEX IF NOT EXISTS findings_revision_idx ON findings(revision_id, status, finding_id)`,
	`CREATE TABLE IF NOT EXISTS check_summaries (revision_id TEXT PRIMARY KEY, body BLOB NOT NULL, FOREIGN KEY(revision_id) REFERENCES revisions(revision_id))`,
	`CREATE TABLE IF NOT EXISTS reviews (decision_id TEXT PRIMARY KEY, package_id TEXT NOT NULL, revision_id TEXT NOT NULL, decided_at TEXT NOT NULL, body BLOB NOT NULL, FOREIGN KEY(package_id) REFERENCES packages(package_id))`,
	`CREATE INDEX IF NOT EXISTS reviews_package_idx ON reviews(package_id, decided_at DESC, decision_id DESC)`,
	`CREATE TABLE IF NOT EXISTS manifests (manifest_id TEXT PRIMARY KEY, package_id TEXT NOT NULL, issued_at TEXT NOT NULL, body BLOB NOT NULL, FOREIGN KEY(package_id) REFERENCES packages(package_id))`,
	`CREATE INDEX IF NOT EXISTS manifests_package_idx ON manifests(package_id, issued_at DESC, manifest_id DESC)`,
	`CREATE TABLE IF NOT EXISTS audit_events (event_id TEXT PRIMARY KEY, package_id TEXT NOT NULL, sequence INTEGER NOT NULL, digest TEXT NOT NULL, previous_digest TEXT NOT NULL, body BLOB NOT NULL, UNIQUE(package_id, sequence), FOREIGN KEY(package_id) REFERENCES packages(package_id))`,
	`CREATE INDEX IF NOT EXISTS audit_package_idx ON audit_events(package_id, sequence)`,
	`CREATE TABLE IF NOT EXISTS idempotency (key TEXT PRIMARY KEY, command TEXT NOT NULL, package_id TEXT NOT NULL, response BLOB NOT NULL, FOREIGN KEY(package_id) REFERENCES packages(package_id))`,
}

func initialize(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, statement := range schemaStatements {
		if _, err := tx.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("初始化 SQLite schema: %w", err)
		}
	}
	var version int
	err = tx.QueryRowContext(ctx, `SELECT value FROM schema_metadata WHERE key = 'schema_version'`).Scan(&version)
	if err == sql.ErrNoRows {
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_metadata(key, value) VALUES('schema_version', ?)`, schemaVersion); err != nil {
			return err
		}
		version = schemaVersion
	} else if err != nil {
		return err
	}
	if version > schemaVersion {
		return fmt.Errorf("数据库 schemaVersion %d 高于程序支持的 %d", version, schemaVersion)
	}
	if version < schemaVersion {
		return fmt.Errorf("数据库 schemaVersion %d 需要迁移到 %d", version, schemaVersion)
	}
	return tx.Commit()
}
