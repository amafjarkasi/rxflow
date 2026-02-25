// Package main provides the routing service entry point.
// Consumes prescription events and routes to pharmacies via NCPDP SCRIPT.
package main

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/drfirst/go-oec/internal/domain/prescription"
	"github.com/drfirst/go-oec/internal/infrastructure/redpanda"
	"github.com/drfirst/go-oec/pkg/circuitbreaker"
	"github.com/drfirst/go-oec/pkg/workerpool"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Load config
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://oec:oec_dev_password@localhost:5432/oec?sslmode=disable"
	}

	brokers := []string{"localhost:9092"}
	if b := os.Getenv("KAFKA_BROKERS"); b != "" {
		brokers = []string{b}
	}

	// Connect to database
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer pool.Close()

	// Create circuit breaker manager
	cbManager := circuitbreaker.NewManager(logger)

	// Create worker pool
	poolCfg := workerpool.DefaultConfig()
	poolCfg.Workers = 50

	workerPool, err := workerpool.New(poolCfg, func(ctx context.Context, task *workerpool.Task) *workerpool.Result {
		return processRoutingTask(ctx, task, pool, cbManager, logger)
	}, logger)
	if err != nil {
		logger.Fatal("worker pool creation failed", zap.Error(err))
	}

	workerPool.Start()
	defer workerPool.Stop()

	// Create consumer
	consumerCfg := redpanda.DefaultConsumerConfig()
	consumerCfg.Brokers = brokers
	consumerCfg.GroupID = "routing-service"
	consumerCfg.Topics = []string{redpanda.TopicRoutingRequests}

	consumer, err := redpanda.NewConsumer(consumerCfg, func(ctx context.Context, msg *redpanda.ConsumedMessage) error {
		task := &workerpool.Task{
			ID:      string(msg.Key),
			Payload: msg.Value,
			Context: ctx,
		}
		return workerPool.Submit(task)
	}, logger)
	if err != nil {
		logger.Fatal("consumer creation failed", zap.Error(err))
	}

	consumer.Start()
	logger.Info("routing service started")

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down")
	consumer.Stop()
	logger.Info("routing service stopped")
}

// RoutingRequest represents a routing request message
type RoutingRequest struct {
	PrescriptionID  string `json:"prescription_id"`
	PharmacyNCPDPID string `json:"pharmacy_ncpdp_id"`
	PharmacyName    string `json:"pharmacy_name"`
	Priority        string `json:"priority"`
}

func processRoutingTask(ctx context.Context, task *workerpool.Task, pool *pgxpool.Pool, cbManager *circuitbreaker.Manager, logger *zap.Logger) *workerpool.Result {
	var req RoutingRequest
	if err := json.Unmarshal(task.Payload.([]byte), &req); err != nil {
		return &workerpool.Result{TaskID: task.ID, Success: false, Error: err}
	}

	// Get circuit breaker for this pharmacy
	cb, err := cbManager.GetOrCreate(req.PharmacyNCPDPID, circuitbreaker.DefaultConfig(req.PharmacyNCPDPID))
	if err != nil {
		return &workerpool.Result{TaskID: task.ID, Success: false, Error: err}
	}

	// Execute with circuit breaker
	_, err = cb.Execute(ctx, func() (interface{}, error) {
		return routeToPharmacy(ctx, pool, &req, logger)
	})

	if err != nil {
		logger.Error("routing failed",
			zap.String("prescription_id", req.PrescriptionID),
			zap.String("pharmacy", req.PharmacyNCPDPID),
			zap.Error(err),
		)
		return &workerpool.Result{TaskID: task.ID, Success: false, Error: err}
	}

	logger.Info("prescription routed",
		zap.String("prescription_id", req.PrescriptionID),
		zap.String("pharmacy", req.PharmacyNCPDPID),
	)

	return &workerpool.Result{TaskID: task.ID, Success: true}
}

func routeToPharmacy(ctx context.Context, pool *pgxpool.Pool, req *RoutingRequest, logger *zap.Logger) (interface{}, error) {
	// Load prescription
	repo := prescription.NewRepository(pool, logger)
	agg, err := repo.Load(ctx, req.PrescriptionID)
	if err != nil {
		return nil, err
	}

	// Route prescription
	if err := agg.Route(req.PharmacyNCPDPID, req.PharmacyName); err != nil {
		return nil, err
	}

	// Save
	if err := repo.Save(ctx, agg); err != nil {
		return nil, err
	}

	// TODO: Transform to NCPDP SCRIPT and transmit
	// For now, just mark as transmitted
	if err := agg.MarkTransmitted("MSG-" + req.PrescriptionID[:8]); err != nil {
		return nil, err
	}

	return nil, repo.Save(ctx, agg)
}
