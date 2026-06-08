// Command seed resets the queue data (table `queues`) and can optionally
// populate fresh sample tickets for local testing / demos.
//
// Queue numbers are derived dynamically from the rows already present for the
// current day+service (see repository/queue.go), so deleting every row is all
// that's needed to reset the daily numbering back to 01.
//
// Destructive actions require -yes; without it the command only previews.
//
// Usage:
//
//	go run ./cmd/seed -dry            # show how many tickets exist, change nothing
//	go run ./cmd/seed -yes            # delete ALL queue tickets (reset)
//	go run ./cmd/seed -sample -yes    # reset, then insert sample tickets for today
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	dry := flag.Bool("dry", false, "preview row counts without changing anything")
	yes := flag.Bool("yes", false, "confirm destructive actions (required to actually reset)")
	sample := flag.Bool("sample", false, "after reset, insert sample tickets for today")
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

	var total int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM queues`).Scan(&total); err != nil {
		fatal(fmt.Sprintf("count queues: %v", err))
	}
	fmt.Printf("queues currently holds %d ticket(s).\n", total)

	if *dry {
		fmt.Printf("[dry] would DELETE %d ticket(s)", total)
		if *sample {
			fmt.Print(", then insert sample tickets")
		}
		fmt.Println(" — nothing changed.")
		return
	}

	if !*yes {
		fmt.Println("\nRefusing to modify data without -yes.")
		fmt.Println("  preview:  go run ./cmd/seed -dry")
		fmt.Println("  reset:    go run ./cmd/seed -yes")
		fmt.Println("  reset+sample: go run ./cmd/seed -sample -yes")
		os.Exit(1)
	}

	tag, err := pool.Exec(ctx, `DELETE FROM queues`)
	if err != nil {
		fatal(fmt.Sprintf("reset queues: %v", err))
	}
	fmt.Printf("✓ reset — deleted %d ticket(s).\n", tag.RowsAffected())

	if *sample {
		n, err := seedSample(ctx, pool)
		if err != nil {
			fatal(fmt.Sprintf("seed sample: %v", err))
		}
		fmt.Printf("✓ inserted %d sample ticket(s) for today.\n", n)
	}

	fmt.Println("\nDone.")
}

// sampleSpec describes how many tickets to create per service and in which
// statuses. Numbers are allocated per service in the order listed, matching the
// real per-(day,service) sequence used by the app.
type sampleSpec struct {
	service  string
	statuses []string
}

// seedSample inserts a realistic spread of tickets for the current day across
// the active services. Completed tickets carry a rating; calling/serving
// tickets are stamped with the first available counter + staff user when one
// exists. Returns the number of tickets inserted.
func seedSample(ctx context.Context, pool *pgxpool.Pool) (int, error) {
	specs := []sampleSpec{
		{"UMUM", []string{"completed", "completed", "serving", "waiting", "waiting", "waiting"}},
		{"LAB", []string{"completed", "calling", "waiting", "waiting"}},
		{"AMP", []string{"serving", "waiting", "waiting"}},
		{"UTIL", []string{"completed", "skipped", "waiting"}},
		{"SEWA", []string{"waiting", "waiting"}},
	}

	// Optional FK targets for called/served tickets. Both may be empty.
	var counterID *int
	if id, err := firstCounterID(ctx, pool); err != nil {
		return 0, err
	} else {
		counterID = id
	}
	var userID *string
	if id, err := firstUserID(ctx, pool); err != nil {
		return 0, err
	} else {
		userID = id
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	inserted := 0
	for _, spec := range specs {
		var code string
		err := tx.QueryRow(ctx,
			`SELECT code FROM services WHERE key = $1 AND is_active = true`, spec.service,
		).Scan(&code)
		if errors.Is(err, pgx.ErrNoRows) {
			// Service not present/active in this DB — skip it.
			continue
		}
		if err != nil {
			return 0, err
		}

		for i, status := range spec.statuses {
			number := fmt.Sprintf("%s-%02d", code, i+1)
			if err := insertTicket(ctx, tx, number, spec.service, status, counterID, userID); err != nil {
				return 0, err
			}
			inserted++
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return inserted, nil
}

// insertTicket writes one sample ticket with timestamps and (where relevant)
// counter/user/rating consistent with its status.
func insertTicket(ctx context.Context, tx pgx.Tx, number, service, status string, counterID *int, userID *string) error {
	switch status {
	case "waiting":
		_, err := tx.Exec(ctx, `
			INSERT INTO queues (queue_number, service_type, status, created_at)
			VALUES ($1, $2, 'waiting', NOW() - (random() * interval '20 minutes'))
		`, number, service)
		return err

	case "calling":
		_, err := tx.Exec(ctx, `
			INSERT INTO queues (queue_number, service_type, status, created_at, called_at, counter_id, user_id)
			VALUES ($1, $2, 'calling', NOW() - interval '8 minutes', NOW(), $3, $4)
		`, number, service, counterID, userID)
		return err

	case "serving":
		_, err := tx.Exec(ctx, `
			INSERT INTO queues (queue_number, service_type, status, created_at, called_at, counter_id, user_id)
			VALUES ($1, $2, 'serving', NOW() - interval '12 minutes', NOW() - interval '3 minutes', $3, $4)
		`, number, service, counterID, userID)
		return err

	case "skipped":
		_, err := tx.Exec(ctx, `
			INSERT INTO queues (queue_number, service_type, status, created_at, called_at, counter_id, user_id)
			VALUES ($1, $2, 'skipped', NOW() - interval '25 minutes', NOW() - interval '15 minutes', $3, $4)
		`, number, service, counterID, userID)
		return err

	case "completed":
		// rating 4–5, completed a little while ago.
		rating := 4 + (len(number) % 2)
		_, err := tx.Exec(ctx, `
			INSERT INTO queues (queue_number, service_type, status, created_at, called_at, completed_at, counter_id, user_id, rating, feedback)
			VALUES ($1, $2, 'completed',
			        NOW() - interval '40 minutes',
			        NOW() - interval '35 minutes',
			        NOW() - interval '28 minutes',
			        $3, $4, $5, 'Pelayanan sample untuk pengujian.')
		`, number, service, counterID, userID, rating)
		return err

	default:
		return fmt.Errorf("unknown sample status %q", status)
	}
}

func firstCounterID(ctx context.Context, pool *pgxpool.Pool) (*int, error) {
	var id int
	err := pool.QueryRow(ctx, `SELECT id FROM counters ORDER BY id ASC LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func firstUserID(ctx context.Context, pool *pgxpool.Pool) (*string, error) {
	var id string
	// Prefer a staff account; fall back to any user.
	err := pool.QueryRow(ctx, `
		SELECT id FROM users
		WHERE is_active = true
		ORDER BY (role = 'staff') DESC, created_at ASC
		LIMIT 1
	`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func fatal(msg string) {
	fmt.Fprintln(os.Stderr, "seed:", msg)
	os.Exit(1)
}
