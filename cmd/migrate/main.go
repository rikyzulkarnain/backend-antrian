// Command migrate applies *.up.sql files in lexical order against
// DATABASE_URL. Tracks applied versions in a `schema_migrations` table so
// it's safe to re-run. Usage:
//
//	go run ./cmd/migrate          # apply pending
//	go run ./cmd/migrate -dry     # list pending without applying
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	dry := flag.Bool("dry", false, "print pending migrations without applying")
	flag.Parse()

	_ = godotenv.Load()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		fatal("DATABASE_URL not set (looked in env and .env)")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		fatal(fmt.Sprintf("connect: %v", err))
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`); err != nil {
		fatal(fmt.Sprintf("ensure schema_migrations: %v", err))
	}

	dir, err := findMigrationsDir()
	if err != nil {
		fatal(fmt.Sprintf("locate migrations dir: %v", err))
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		fatal(fmt.Sprintf("read dir: %v", err))
	}

	files := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	pending := 0
	for _, f := range files {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`,
			f,
		).Scan(&exists); err != nil {
			fatal(fmt.Sprintf("check %s: %v", f, err))
		}
		if exists {
			fmt.Printf("  skip %s\n", f)
			continue
		}
		pending++
		if *dry {
			fmt.Printf("[dry] would apply %s\n", f)
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			fatal(fmt.Sprintf("read %s: %v", f, err))
		}
		if _, err := pool.Exec(ctx, string(body)); err != nil {
			fatal(fmt.Sprintf("apply %s: %v", f, err))
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, f,
		); err != nil {
			fatal(fmt.Sprintf("record %s: %v", f, err))
		}
		fmt.Printf("✓ apply %s\n", f)
	}

	if pending == 0 {
		fmt.Println("\nNothing to apply — database is up to date.")
	} else if !*dry {
		fmt.Printf("\nApplied %d migration(s).\n", pending)
	}
}

// findMigrationsDir walks up from CWD looking for a `migrations` directory.
// Lets the command run from `backend/` or any subdir.
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
			return "", fmt.Errorf("not found from %s up", cwd)
		}
		dir = parent
	}
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "migrate:", msg)
	os.Exit(1)
}
