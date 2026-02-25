// Package main provides the ingestion API service entry point.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/drfirst/go-oec/internal/api/handlers"
	"github.com/drfirst/go-oec/internal/api/middleware"
	"github.com/drfirst/go-oec/internal/domain/prescription"
)

// Config holds application configuration
type Config struct {
	Port        string
	DatabaseURL string
	APIKeys     map[string]string
	LogLevel    string
}

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load configuration
	cfg := loadConfig()

	// Connect to database
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}
	defer pool.Close()

	// Verify database connection
	if err := pool.Ping(context.Background()); err != nil {
		logger.Fatal("database ping failed", zap.Error(err))
	}
	logger.Info("connected to database")

	// Initialize repositories
	prescriptionRepo := prescription.NewRepository(pool, logger)

	// Initialize handlers
	prescriptionHandler := handlers.NewPrescriptionHandler(prescriptionRepo, logger)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.CORS)
	r.Use(middleware.Recover(logger))
	r.Use(middleware.Logger(logger))
	r.Use(middleware.Tracing("ingestion-api"))

	// Health check (no auth)
	r.Get("/health", healthHandler)
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		if err := pool.Ping(r.Context()); err != nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	})

	// API routes (with auth)
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(cfg.APIKeys))
		r.Mount("/prescriptions", prescriptionHandler.Routes())
	})

	// Start server
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		logger.Info("shutting down server")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("shutdown error", zap.Error(err))
		}
	}()

	logger.Info("starting ingestion API", zap.String("port", cfg.Port))
	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		logger.Fatal("server error", zap.Error(err))
	}

	logger.Info("server stopped")
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://oec:oec_dev_password@localhost:5432/oec?sslmode=disable"
	}

	// Simple API keys for demo
	apiKeys := map[string]string{
		"demo-api-key-12345": "demo-client",
		"test-api-key-67890": "test-client",
	}

	// Override from environment if set
	if key := os.Getenv("API_KEY"); key != "" {
		apiKeys[key] = "env-client"
	}

	return Config{
		Port:        port,
		DatabaseURL: dbURL,
		APIKeys:     apiKeys,
		LogLevel:    os.Getenv("LOG_LEVEL"),
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"healthy","service":"ingestion-api","version":"1.0.0"}`)
}
