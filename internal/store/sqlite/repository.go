package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Repository struct{ db *sql.DB }

func Open(ctx context.Context, path string) (*Repository, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "caption-release-gate.db"
	}
	dsn := path
	if path != ":memory:" && !strings.HasPrefix(path, "file:") {
		dsn = "file:" + path
	}
	if strings.Contains(dsn, "?") {
		dsn += "&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	} else {
		dsn += "?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("打开 SQLite: %w", err)
	}
	if path == ":memory:" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(8)
	}
	db.SetConnMaxLifetime(30 * time.Minute)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("连接 SQLite: %w", err)
	}
	if err := initialize(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	return &Repository{db: db}, nil
}

func (r *Repository) Close() error { return r.db.Close() }
func (r *Repository) DB() *sql.DB  { return r.db }
