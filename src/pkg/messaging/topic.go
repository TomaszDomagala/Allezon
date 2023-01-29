package messaging

import (
	"fmt"
	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

// Initialize creates a topic for user tags if it doesn't exist.
func Initialize(logger *zap.Logger, addresses []string, details *sarama.TopicDetail) error {
	config := sarama.NewConfig()
	admin, err := sarama.NewClusterAdmin(addresses, config)
	if err != nil {
		return fmt.Errorf("failed to create cluster admin: %w", err)
	}
	defer func() {
		err := admin.Close()
		if err != nil {
			logger.Error("failed to close cluster admin", zap.Error(err))
		}
	}()

	topics, err := admin.ListTopics()
	if err != nil {
		return fmt.Errorf("failed to list topics: %w", err)
	}

	if details, ok := topics[UserTagsTopic]; ok {
		logger.Info("topic already exists", zap.String("topic", UserTagsTopic), zap.Int32("partitions", details.NumPartitions), zap.Int16("replication factor", details.ReplicationFactor))
		return nil
	}

	err = admin.CreateTopic(UserTagsTopic, details, false)

	if err != nil {
		return fmt.Errorf("failed to create topic: %w", err)
	}
	logger.Info("topic created", zap.String("topic", UserTagsTopic), zap.Int32("partitions", details.NumPartitions), zap.Int16("replication factor", details.ReplicationFactor))
	return nil
}
