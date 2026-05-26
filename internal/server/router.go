// Package server wires repositories, services, and handlers into an HTTP
// router. Exporting it lets integration tests build the same router as
// main.go without duplicating wiring.
package server

import (
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bbpjn-sumsel/sistem-antrian/internal/config"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/handler"
	mw "github.com/bbpjn-sumsel/sistem-antrian/internal/middleware"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/repository"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/service"
	"github.com/bbpjn-sumsel/sistem-antrian/internal/sse"
)

// Deps groups the long-lived dependencies the router holds onto. Returned
// so callers (main, tests) can shut them down in the right order.
type Deps struct {
	Broker *sse.Broker
}

// New builds a fully-wired chi.Router. Caller owns the *pgxpool.Pool (must
// outlive the returned handler) and is responsible for closing it.
func New(cfg *config.Config, pool *pgxpool.Pool) (chi.Router, *Deps) {
	// repositories
	queueRepo := repository.NewQueueRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	videoRepo := repository.NewVideoRepository(pool)
	counterRepo := repository.NewCounterRepository(pool)
	analyticsRepo := repository.NewAnalyticsRepository(pool)
	serviceRepo := repository.NewServiceRepository(pool)

	// sse broker
	broker := sse.NewBroker()

	// services
	queueSvc := service.NewQueueService(queueRepo, broker)
	authSvc := service.NewAuthService(userRepo, []byte(cfg.JWTSecret))
	videoSvc := service.NewVideoService(videoRepo).WithCloudinary(service.CloudinaryConfig{
		APIKey:       cfg.CloudinaryAPIKey,
		APISecret:    cfg.CloudinaryAPISecret,
		CloudName:    cfg.CloudinaryCloudName,
		UploadPreset: cfg.CloudinaryUploadPreset,
		Folder:       cfg.CloudinaryUploadFolder,
	})
	userSvc := service.NewUserService(userRepo)
	counterSvc := service.NewCounterService(counterRepo)
	analyticsSvc := service.NewAnalyticsService(analyticsRepo)
	serviceSvc := service.NewServiceService(serviceRepo)

	// handlers
	queueH := handler.NewQueueHandler(queueSvc)
	authH := handler.NewAuthHandler(authSvc, userSvc, cfg.CookieSecure)
	counterH := handler.NewCounterHandler(counterSvc)
	videoH := handler.NewVideoHandler(videoSvc)
	userH := handler.NewUserHandler(userSvc)
	analyticsH := handler.NewAnalyticsHandler(analyticsSvc)
	serviceH := handler.NewServiceHandler(serviceSvc)
	sseH := handler.NewSSEHandler(broker)

	r := chi.NewRouter()
	r.Use(mw.Recover)
	r.Use(mw.Logger)
	r.Use(mw.CORS(cfg.CORSOrigins))

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", handler.Health)

		// Auth — public except /me which requires bearer.
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)
		r.Post("/auth/logout", authH.Logout)

		// Public reads (kiosk + display TV consume without auth).
		r.Get("/queues", queueH.List)
		r.Get("/queues/{id}", queueH.Get)
		r.Post("/queues", queueH.Create)
		r.Post("/queues/{id}/rating", queueH.Rate)
		r.Get("/counters", counterH.List)
		r.Get("/videos", videoH.List)
		r.Get("/services", serviceH.List)
		r.Get("/stream", sseH.Stream)
		r.Get("/sse/queues", sseH.Stream)

		// Staff + admin (any authenticated user) — queue operations.
		r.Group(func(r chi.Router) {
			r.Use(mw.JWT([]byte(cfg.JWTSecret)))
			r.Get("/auth/me", authH.Me)
			r.Post("/auth/change-password", authH.ChangePassword)

			r.Post("/queues/call", queueH.Call)
			r.Post("/queues/{id}/recall", queueH.Recall)
			r.Post("/queues/{id}/complete", queueH.Complete)
			r.Post("/queues/{id}/skip", queueH.Skip)

			r.Get("/analytics/overview", analyticsH.Overview)
			r.Get("/analytics/by-service", analyticsH.ByService)
			r.Get("/analytics/by-hour", analyticsH.ByHour)
			r.Get("/analytics/staff", analyticsH.Staff)
			r.Get("/analytics/ratings", analyticsH.Ratings)
			r.Get("/analytics/issues", analyticsH.Issues)
		})

		// Admin-only CRUD.
		r.Group(func(r chi.Router) {
			r.Use(mw.JWT([]byte(cfg.JWTSecret)))
			r.Use(mw.RequireRole("admin"))

			r.Get("/users", userH.List)
			r.Get("/users/{id}", userH.Get)
			r.Post("/users", userH.Create)
			r.Patch("/users/{id}", userH.Update)
			r.Delete("/users/{id}", userH.Delete)

			r.Get("/counters/{id}", counterH.Get)
			r.Post("/counters", counterH.Create)
			r.Patch("/counters/{id}", counterH.Update)
			r.Delete("/counters/{id}", counterH.Delete)

			r.Get("/videos/{id}", videoH.Get)
			r.Post("/videos", videoH.Create)
			r.Patch("/videos/{id}", videoH.Update)
			r.Delete("/videos/{id}", videoH.Delete)
			r.Post("/videos/upload-signature", videoH.UploadSignature)

			r.Get("/services/{key}", serviceH.Get)
			r.Patch("/services/{key}", serviceH.Update)
		})
	})

	return r, &Deps{Broker: broker}
}
