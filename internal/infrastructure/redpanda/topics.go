// Package redpanda provides topic management and configuration.
package redpanda

import (
	"context"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
	"go.uber.org/zap"
)

// Predefined topic names for the prescription orchestration engine
const (
	TopicPrescriptionEvents   = "prescription.events"
	TopicPrescriptionCommands = "prescription.commands"
	TopicRoutingRequests      = "routing.requests"
	TopicRoutingResponses     = "routing.responses"
	TopicAuditTrail           = "audit.trail"
	TopicDeadLetter           = "dead.letter"
	TopicNCPDPOutbound        = "ncpdp.outbound"
	TopicNCPDPInbound         = "ncpdp.inbound"
)

// TopicConfig holds configuration for a Kafka topic
type TopicConfig struct {
	Name              string
	Partitions        int32
	ReplicationFactor int16
	Configs           map[string]*string
}

// DefaultTopicConfigs returns optimized topic configurations for prescription processing
func DefaultTopicConfigs() []TopicConfig {
	// Helper for string pointers
	ptr := func(s string) *string { return &s }

	return []TopicConfig{
		{
			Name:              TopicPrescriptionEvents,
			Partitions:        12, // High parallelism for event sourcing
			ReplicationFactor: 1,  // Set to 3 in production
			Configs: map[string]*string{
				"retention.ms":        ptr("604800000"), // 7 days
				"cleanup.policy":      ptr("delete"),
				"compression.type":    ptr("lz4"),
				"min.insync.replicas": ptr("1"),         // Set to 2 in production
				"segment.bytes":       ptr("536870912"), // 512MB segments
			},
		},
		{
			Name:              TopicPrescriptionCommands,
			Partitions:        6,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("86400000"), // 1 day
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
		{
			Name:              TopicRoutingRequests,
			Partitions:        12,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("86400000"),
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
		{
			Name:              TopicRoutingResponses,
			Partitions:        12,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("86400000"),
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
		{
			Name:              TopicAuditTrail,
			Partitions:        6,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("2592000000"), // 30 days (legal compliance)
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
		{
			Name:              TopicDeadLetter,
			Partitions:        3,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("604800000"), // 7 days
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
		{
			Name:              TopicNCPDPOutbound,
			Partitions:        12,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("86400000"),
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
		{
			Name:              TopicNCPDPInbound,
			Partitions:        12,
			ReplicationFactor: 1,
			Configs: map[string]*string{
				"retention.ms":     ptr("86400000"),
				"cleanup.policy":   ptr("delete"),
				"compression.type": ptr("lz4"),
			},
		},
	}
}

// Admin provides administrative operations for Redpanda
type Admin struct {
	client *kadm.Client
	logger *zap.Logger
}

// NewAdmin creates a new admin client
func NewAdmin(brokers []string, logger *zap.Logger) (*Admin, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	kgoClient, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}

	return &Admin{
		client: kadm.NewClient(kgoClient),
		logger: logger,
	}, nil
}

// CreateTopics creates the specified topics
func (a *Admin) CreateTopics(ctx context.Context, configs []TopicConfig) error {
	for _, cfg := range configs {
		resp, err := a.client.CreateTopics(ctx, int32(cfg.Partitions), cfg.ReplicationFactor, cfg.Configs, cfg.Name)
		if err != nil {
			return fmt.Errorf("failed to create topic %s: %w", cfg.Name, err)
		}

		for _, r := range resp {
			if r.Err != nil {
				// Ignore "topic already exists" errors
				if r.Err.Error() == "TOPIC_ALREADY_EXISTS" {
					a.logger.Info("topic already exists", zap.String("topic", r.Topic))
					continue
				}
				return fmt.Errorf("failed to create topic %s: %w", r.Topic, r.Err)
			}
			a.logger.Info("topic created",
				zap.String("topic", r.Topic),
				zap.Int32("partitions", cfg.Partitions))
		}
	}
	return nil
}

// EnsureTopics ensures all required topics exist with proper configuration
func (a *Admin) EnsureTopics(ctx context.Context) error {
	return a.CreateTopics(ctx, DefaultTopicConfigs())
}

// DeleteTopics deletes the specified topics
func (a *Admin) DeleteTopics(ctx context.Context, topics ...string) error {
	resp, err := a.client.DeleteTopics(ctx, topics...)
	if err != nil {
		return fmt.Errorf("failed to delete topics: %w", err)
	}

	for _, r := range resp {
		if r.Err != nil {
			a.logger.Warn("failed to delete topic",
				zap.String("topic", r.Topic),
				zap.Error(r.Err))
		} else {
			a.logger.Info("topic deleted", zap.String("topic", r.Topic))
		}
	}
	return nil
}

// ListTopics lists all topics
func (a *Admin) ListTopics(ctx context.Context) ([]string, error) {
	topics, err := a.client.ListTopics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list topics: %w", err)
	}

	var names []string
	for _, t := range topics {
		names = append(names, t.Topic)
	}
	return names, nil
}

// DescribeTopic returns details about a topic
func (a *Admin) DescribeTopic(ctx context.Context, topic string) (*TopicDetails, error) {
	topics, err := a.client.ListTopics(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to describe topic: %w", err)
	}

	t, ok := topics[topic]
	if !ok {
		return nil, fmt.Errorf("topic %s not found", topic)
	}

	var partitions []PartitionDetails
	for _, p := range t.Partitions {
		partitions = append(partitions, PartitionDetails{
			ID:       p.Partition,
			Leader:   p.Leader,
			Replicas: p.Replicas,
			ISR:      p.ISR,
		})
	}

	return &TopicDetails{
		Name:       topic,
		Partitions: partitions,
	}, nil
}

// GetConsumerGroupLag returns the lag for a consumer group
func (a *Admin) GetConsumerGroupLag(ctx context.Context, groupID string) (map[string]map[int32]int64, error) {
	described, err := a.client.Lag(ctx, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer group lag: %w", err)
	}

	result := make(map[string]map[int32]int64)
	described.Each(func(l kadm.DescribedGroupLag) {
		for topic, partitions := range l.Lag {
			if result[topic] == nil {
				result[topic] = make(map[int32]int64)
			}
			for partition, lag := range partitions {
				result[topic][partition] = lag.Lag
			}
		}
	})
	return result, nil
}

// Close closes the admin client
func (a *Admin) Close() {
	a.client.Close()
}

// TopicDetails holds topic information
type TopicDetails struct {
	Name       string
	Partitions []PartitionDetails
}

// PartitionDetails holds partition information
type PartitionDetails struct {
	ID       int32
	Leader   int32
	Replicas []int32
	ISR      []int32
}

// HealthCheck verifies Redpanda connectivity
func HealthCheck(ctx context.Context, brokers []string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := kgo.NewClient(kgo.SeedBrokers(brokers...))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	return nil
}
