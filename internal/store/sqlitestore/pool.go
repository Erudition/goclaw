//go:build sqlite || sqliteonly

package sqlitestore

import (
	"database/sql"
	"fmt"
	"log/slog"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

// OpenDB opens a SQLite database at the given path with WAL mode and recommended pragmas.
// Uses modernc.org/sqlite (pure Go, zero CGo).
//
// Desktop app concurrency model:
// - WAL mode: allows concurrent readers alongside a single writer
// - MaxOpenConns(2): limits contention; one reader + one writer can proceed in parallel
// - busy_timeout=15000: writer retries for 15s before SQLITE_BUSY error
// - _txlock=immediate: write transactions acquire lock immediately (fail-fast on contention)
func OpenDB(path string) (*sql.DB, error) {
	dsn := fmt.Sprintf("file:%s?_txlock=immediate&_foreign_keys=ON", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite is single-writer; WAL allows concurrent readers.
	// Keep pool small to reduce lock contention.
	db.SetMaxOpenConns(2)

	// Set PRAGMAs explicitly — DSN params may not be applied by modernc.org/sqlite.
	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 15000",
		"PRAGMA synchronous = NORMAL",
		"PRAGMA cache_size = -8000", // 8MB cache
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			slog.Warn("sqlite pragma failed", "pragma", p, "error", err)
		}
	}

	// Verify connection works.
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	return db, nil
}
