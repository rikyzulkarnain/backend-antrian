package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/config"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		slog.Error("db ping", "err", err)
		os.Exit(1)
	}

	r, _ := server.New(cfg, pool)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("Server started", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("listen", "err", err)
			cancel()
		}
	}()

	<-ctx.Done()
	slog.Info("Server shutting down")

	shutdownCtx, sCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer sCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
	slog.Info("Server stopped")
}
