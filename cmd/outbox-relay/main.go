// Package main provides the outbox relay service entry point.
// Implements the Transactional Outbox pattern relay.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/drfirst/go-oec/internal/infrastructure/postgres"
	"github.com/drfirst/go-oec/internal/infrastructure/redpanda"
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

	logger.Info("connected to database")

	// Create Redpanda producer
	producerCfg := redpanda.DefaultProducerConfig()
	producerCfg.Brokers = brokers

	producer, err := redpanda.NewProducer(producerCfg, logger)
	if err != nil {
		logger.Fatal("producer creation failed", zap.Error(err))
	}
	defer producer.Close()

	logger.Info("connected to Redpanda", zap.Strings("brokers", brokers))

	// Create outbox processor
	outboxCfg := postgres.DefaultOutboxConfig()
	outbox := postgres.NewOutbox(pool, &producerAdapter{producer}, outboxCfg, logger)

	// Start processing
	outbox.Start()
	logger.Info("outbox relay started")

	// Wait for shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down")
	outbox.Stop()
	logger.Info("outbox relay stopped")
}

// producerAdapter adapts the Redpanda producer to OutboxPublisher interface
type producerAdapter struct {
	producer *redpanda.Producer
}

func (a *producerAdapter) Publish(ctx context.Context, topic, key string, value []byte) error {
	return a.producer.ProduceMessage(ctx, topic, key, value)
}
