package messaging

import (
	"encoding/json"
	"fmt"
	"hash/adler32"
	"time"

	"github.com/Shopify/sarama"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type UserTagsProducer interface {
	Send(tag types.UserTag) error
}

type Producer struct {
	logger           *zap.Logger
	client           sarama.Client
	producer         sarama.SyncProducer
	partitionsNumber int32
}

func NewProducer(logger *zap.Logger, addresses []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	client, err := sarama.NewClient(addresses, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kafka client: %w", err)
	}
	producer, err := sarama.NewSyncProducerFromClient(client)
	if err != nil {
		return nil, fmt.Errorf("failed to create producer: %w", err)
	}
	partitionsNumber, err := getPartitionNumber(client, UserTagsTopic)
	if err != nil {
		return nil, fmt.Errorf("failed to get partitions number: %w", err)
	}

	return &Producer{
		logger:           logger,
		client:           client,
		producer:         producer,
		partitionsNumber: partitionsNumber,
	}, nil
}

func (p *Producer) Send(tag types.UserTag) error {
	start := time.Now()

	tagJson, err := json.Marshal(tag)
	if err != nil {
		return fmt.Errorf("failed to marshal user tag: %w", err)
	}
	partition, offset, err := p.producer.SendMessage(&sarama.ProducerMessage{
		Topic:     UserTagsTopic,
		Value:     sarama.ByteEncoder(tagJson),
		Partition: tagPartition(tag, p.partitionsNumber),
	})

	logOpts := []zap.Field{
		zap.String("topic", UserTagsTopic),
		zap.ByteString("value", tagJson),
		zap.Duration("duration", time.Since(start)),
	}
	if err != nil {
		p.logger.Error("failed to send kafka message", append(logOpts, zap.Error(err))...)
		return fmt.Errorf("failed to send kafka message: %w", err)
	}
	p.logger.Debug("kafka message sent", append(logOpts, zap.Int32("partition", partition), zap.Int64("offset", offset))...)
	return nil
}

// getPartitionNumber returns the number of partitions for the given topic.
func getPartitionNumber(client sarama.Client, topic string) (int32, error) {
	partitions, err := client.Partitions(topic)
	if err != nil {
		return 0, fmt.Errorf("failed to get partitions: %w", err)
	}
	return int32(len(partitions)), nil
}

// tagPartition hashes tag.Cookie and returns the modulo of the number of partitions.
func tagPartition(tag types.UserTag, partitionsNumber int32) int32 {
	return int32(adler32.Checksum([]byte(tag.Cookie))) % partitionsNumber
}
