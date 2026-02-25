// Package redpanda provides high-performance Kafka-compatible streaming with franz-go.
// Configured for 5,000+ TPS prescription processing with aggressive batching.
package redpanda

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// ProducerConfig holds configuration for the Redpanda producer
type ProducerConfig struct {
	// Brokers is a list of broker addresses
	Brokers []string
	// BatchMaxBytes is the maximum batch size (default 16MB for high throughput)
	BatchMaxBytes int32
	// LingerMS is the time to wait before sending a batch (default 50ms)
	LingerMS int64
	// MaxBufferedRecords is the maximum number of records to buffer (default 1M)
	MaxBufferedRecords int
	// Compression is the compression codec to use
	Compression string
	// RequiredAcks sets the required acks level (-1 for all, 1 for leader)
	RequiredAcks int16
	// MaxRetries is the maximum number of retries for failed sends
	MaxRetries int
	// RetryBackoffMS is the backoff time between retries
	RetryBackoffMS int64
}

// DefaultProducerConfig returns optimized defaults for high-throughput prescription processing
func DefaultProducerConfig() ProducerConfig {
	return ProducerConfig{
		Brokers:            []string{"localhost:9092"},
		BatchMaxBytes:      16 * 1024 * 1024, // 16MB batches
		LingerMS:           50,               // 50ms linger for batching
		MaxBufferedRecords: 1_000_000,        // 1M buffered records
		Compression:        "lz4",            // LZ4 for fast compression
		RequiredAcks:       -1,               // Wait for all replicas (durability)
		MaxRetries:         3,
		RetryBackoffMS:     100,
	}
}

// Producer provides high-performance message production to Redpanda
type Producer struct {
	client *kgo.Client
	config ProducerConfig
	logger *zap.Logger
	tracer trace.Tracer

	// Metrics
	mu            sync.RWMutex
	messagesSent  int64
	bytesSent     int64
	errorCount    int64
	lastFlushTime time.Time
}

// NewProducer creates a new Redpanda producer with optimized settings
func NewProducer(cfg ProducerConfig, logger *zap.Logger) (*Producer, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ProducerBatchMaxBytes(cfg.BatchMaxBytes),
		kgo.ProducerLinger(time.Duration(cfg.LingerMS) * time.Millisecond),
		kgo.MaxBufferedRecords(cfg.MaxBufferedRecords),
		kgo.RecordRetries(cfg.MaxRetries),
		kgo.RetryBackoffFn(func(attempt int) time.Duration {
			return time.Duration(cfg.RetryBackoffMS) * time.Millisecond * time.Duration(attempt+1)
		}),
	}

	// Set required acks
	switch cfg.RequiredAcks {
	case -1:
		opts = append(opts, kgo.RequiredAcks(kgo.AllISRAcks()))
	case 0:
		opts = append(opts, kgo.RequiredAcks(kgo.NoAck()))
	case 1:
		opts = append(opts, kgo.RequiredAcks(kgo.LeaderAck()))
	}

	// Set compression
	switch cfg.Compression {
	case "lz4":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.Lz4Compression()))
	case "snappy":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.SnappyCompression()))
	case "gzip":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.GzipCompression()))
	case "zstd":
		opts = append(opts, kgo.ProducerBatchCompression(kgo.ZstdCompression()))
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &Producer{
		client:        client,
		config:        cfg,
		logger:        logger,
		tracer:        otel.Tracer("redpanda-producer"),
		lastFlushTime: time.Now(),
	}, nil
}

// ProduceMessage sends a single message to the specified topic
func (p *Producer) ProduceMessage(ctx context.Context, topic, key string, value []byte) error {
	ctx, span := p.tracer.Start(ctx, "produce_message",
		trace.WithAttributes(
			attribute.String("topic", topic),
			attribute.String("key", key),
			attribute.Int("value_size", len(value)),
		))
	defer span.End()

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	// Inject trace context into record headers
	injectTraceHeaders(ctx, record)

	// Produce with callback
	var produceErr error
	var wg sync.WaitGroup
	wg.Add(1)

	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		defer wg.Done()
		if err != nil {
			produceErr = err
			p.incrementErrorCount()
			p.logger.Error("failed to produce message",
				zap.String("topic", topic),
				zap.String("key", key),
				zap.Error(err))
			span.RecordError(err)
		} else {
			p.incrementMetrics(len(r.Value))
			p.logger.Debug("message produced",
				zap.String("topic", r.Topic),
				zap.Int32("partition", r.Partition),
				zap.Int64("offset", r.Offset))
		}
	})

	wg.Wait()
	return produceErr
}

// ProduceAsync sends a message asynchronously without waiting for acknowledgment
func (p *Producer) ProduceAsync(ctx context.Context, topic, key string, value []byte, callback func(error)) {
	ctx, span := p.tracer.Start(ctx, "produce_async",
		trace.WithAttributes(
			attribute.String("topic", topic),
			attribute.String("key", key),
		))

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	injectTraceHeaders(ctx, record)

	p.client.Produce(ctx, record, func(r *kgo.Record, err error) {
		span.End()
		if err != nil {
			p.incrementErrorCount()
			p.logger.Error("async produce failed",
				zap.String("topic", topic),
				zap.Error(err))
		} else {
			p.incrementMetrics(len(r.Value))
		}
		if callback != nil {
			callback(err)
		}
	})
}

// ProduceBatch sends multiple messages in an optimized batch
func (p *Producer) ProduceBatch(ctx context.Context, records []*Record) error {
	ctx, span := p.tracer.Start(ctx, "produce_batch",
		trace.WithAttributes(
			attribute.Int("batch_size", len(records)),
		))
	defer span.End()

	var wg sync.WaitGroup
	var errors []error
	var errorsMu sync.Mutex

	for _, rec := range records {
		kgoRecord := &kgo.Record{
			Topic: rec.Topic,
			Key:   []byte(rec.Key),
			Value: rec.Value,
		}

		if len(rec.Headers) > 0 {
			for k, v := range rec.Headers {
				kgoRecord.Headers = append(kgoRecord.Headers, kgo.RecordHeader{
					Key:   k,
					Value: []byte(v),
				})
			}
		}

		injectTraceHeaders(ctx, kgoRecord)

		wg.Add(1)
		p.client.Produce(ctx, kgoRecord, func(r *kgo.Record, err error) {
			defer wg.Done()
			if err != nil {
				errorsMu.Lock()
				errors = append(errors, err)
				errorsMu.Unlock()
				p.incrementErrorCount()
			} else {
				p.incrementMetrics(len(r.Value))
			}
		})
	}

	wg.Wait()

	if len(errors) > 0 {
		return fmt.Errorf("batch produce failed with %d errors, first: %w", len(errors), errors[0])
	}
	return nil
}

// Flush blocks until all buffered records are sent
func (p *Producer) Flush(ctx context.Context) error {
	ctx, span := p.tracer.Start(ctx, "flush")
	defer span.End()

	if err := p.client.Flush(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("flush failed: %w", err)
	}

	p.mu.Lock()
	p.lastFlushTime = time.Now()
	p.mu.Unlock()

	return nil
}

// Close flushes and closes the producer
func (p *Producer) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := p.client.Flush(ctx); err != nil {
		p.logger.Warn("error flushing on close", zap.Error(err))
	}

	p.client.Close()
	return nil
}

// Stats returns current producer statistics
func (p *Producer) Stats() ProducerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return ProducerStats{
		MessagesSent:  p.messagesSent,
		BytesSent:     p.bytesSent,
		ErrorCount:    p.errorCount,
		LastFlushTime: p.lastFlushTime,
	}
}

// ProducerStats holds producer statistics
type ProducerStats struct {
	MessagesSent  int64
	BytesSent     int64
	ErrorCount    int64
	LastFlushTime time.Time
}

// Record represents a message to be produced
type Record struct {
	Topic   string
	Key     string
	Value   []byte
	Headers map[string]string
}

// Helper methods
func (p *Producer) incrementMetrics(bytes int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.messagesSent++
	p.bytesSent += int64(bytes)
}

func (p *Producer) incrementErrorCount() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorCount++
}

// injectTraceHeaders adds OpenTelemetry trace context to record headers
func injectTraceHeaders(ctx context.Context, record *kgo.Record) {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return
	}

	sc := span.SpanContext()
	record.Headers = append(record.Headers,
		kgo.RecordHeader{Key: "traceparent", Value: []byte(fmt.Sprintf("00-%s-%s-%02x",
			sc.TraceID().String(),
			sc.SpanID().String(),
			sc.TraceFlags()))},
	)
}
