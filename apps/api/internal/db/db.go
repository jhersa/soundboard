// Package db abre la base SQLite y aplica las migraciones. Es compatible con el archivo que
// generaba D1/miniflare: mismo esquema y misma tabla d1_migrations.
package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "modernc.org/sqlite"
)

// Formato de fecha de toda la base: el mismo que strftime('%Y-%m-%dT%H:%M:%fZ') y toISOString().
const TimeFormat = "2006-01-02T15:04:05.000Z"

func Now() string { return time.Now().UTC().Format(TimeFormat) }

// Open abre (o crea) la base. Usa una sola conexión: las peticiones son cortas y así no hay
// SQLITE_BUSY entre escritores; ":memory:" sirve para tests.
func Open(path string) (*sql.DB, error) {
	if path != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
	}
	query := url.Values{}
	for _, pragma := range []string{"foreign_keys(1)", "journal_mode(WAL)", "busy_timeout(5000)"} {
		query.Add("_pragma", pragma)
	}
	database, err := sql.Open("sqlite", "file:"+path+"?"+query.Encode())
	if err != nil {
		return nil, err
	}
	database.SetMaxOpenConns(1)
	if err := database.Ping(); err != nil {
		database.Close()
		return nil, err
	}
	return database, nil
}

// Migrate aplica en orden los .sql que aún no estén registrados en d1_migrations.
func Migrate(ctx context.Context, database *sql.DB, migrations fs.FS) ([]string, error) {
	if _, err := database.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS d1_migrations(
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT UNIQUE,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL
	)`); err != nil {
		return nil, err
	}
	names, err := fs.Glob(migrations, "*.sql")
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	var applied []string
	for _, name := range names {
		var exists int
		if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM d1_migrations WHERE name = ?", name).Scan(&exists); err != nil {
			return applied, err
		}
		if exists > 0 {
			continue
		}
		script, err := fs.ReadFile(migrations, name)
		if err != nil {
			return applied, err
		}
		tx, err := database.BeginTx(ctx, nil)
		if err != nil {
			return applied, err
		}
		if _, err := tx.ExecContext(ctx, string(script)); err != nil {
			tx.Rollback()
			return applied, fmt.Errorf("migración %s: %w", name, err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO d1_migrations (name) VALUES (?)", name); err != nil {
			tx.Rollback()
			return applied, err
		}
		if err := tx.Commit(); err != nil {
			return applied, err
		}
		applied = append(applied, name)
	}
	return applied, nil
}
