// Package testutil contains test-only helpers shared across packages.
// Anything here is intended to be imported only from _test.go files.
package testutil

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// migrationsApplied guards against re-running migrations within one test
// binary invocation. Multiple test files share the same DB; the first one
// in pays the migration cost, the rest just truncate.
var migrationsApplied sync.Once

// SetupDB returns a connection pool to TEST_DATABASE_URL with migrations
// applied. The test is skipped (not failed) when the env var is unset so
// `go test ./...` stays green for contributors without a test DB handy.
//
// On every call the queues + videos tables are truncated so each test sees
// a clean slate. The seeded counters and users (admin@local, staff@local)
// remain in place because tests rely on them.
func SetupDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping integration test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}

	var migErr error
	migrationsApplied.Do(func() {
		migErr = applyMigrations(ctx, pool)
	})
	if migErr != nil {
		t.Fatalf("migrate: %v", migErr)
	}

	// Wipe per-test state. CASCADE so any FK fan-out from seeded counters
	// is handled. Counters + users are seeded by migration 003 and re-asked
	// for by some tests, so we leave them intact.
	if _, err := pool.Exec(ctx, `TRUNCATE TABLE queues, videos RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return pool
}

func applyMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	// Test DB is owned by tests — reset to a pristine schema before
	// re-applying migrations so reruns don't fail on "relation already
	// exists". This is why TEST_DATABASE_URL must point at a dedicated
	// branch/database, never at production data.
	if _, err := pool.Exec(ctx, `DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;`); err != nil {
		return err
	}

	dir, err := findMigrationsDir()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	files := []string{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(name, ".up.sql") {
			files = append(files, name)
		}
	}
	sort.Strings(files)

	for _, f := range files {
		body, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			return err
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			return err
		}
	}
	return nil
}

// findMigrationsDir walks up from CWD looking for backend/migrations. Tests
// can be invoked from any package under backend/, so a fixed relative path
// is fragile.
func findMigrationsDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}
