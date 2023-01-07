package messaging

import (
	"github.com/Shopify/sarama"
	"go.uber.org/zap"
)

type null struct {
	logger *zap.Logger
}

// NewNullProducer returns a producer that does nothing but log invoked methods.
func NewNullProducer(logger *zap.Logger) sarama.SyncProducer {
	return &null{logger: logger}
}

func (n *null) SendMessage(msg *sarama.ProducerMessage) (partition int32, offset int64, err error) {
	n.logger.Info("NullProducer:SendMessage", zap.Any("msg", msg))
	return 0, 0, nil
}

func (n *null) SendMessages(msgs []*sarama.ProducerMessage) error {
	n.logger.Info("NullProducer:SendMessages", zap.Any("msgs", msgs))
	return nil
}

func (n *null) Close() error {
	n.logger.Info("NullProducer:Close")
	return nil
}

func (n *null) TxnStatus() sarama.ProducerTxnStatusFlag {
	n.logger.Info("NullProducer:TxnStatus")
	return sarama.ProducerTxnFlagUninitialized
}

func (n *null) IsTransactional() bool {
	n.logger.Info("NullProducer:IsTransactional")
	return false
}

func (n *null) BeginTxn() error {
	n.logger.Info("NullProducer:BeginTxn")
	return nil
}

func (n *null) CommitTxn() error {
	n.logger.Info("NullProducer:CommitTxn")
	return nil
}

func (n *null) AbortTxn() error {
	n.logger.Info("NullProducer:AbortTxn")
	return nil
}

func (n *null) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	n.logger.Info("NullProducer:AddOffsetsToTxn", zap.Any("offsets", offsets), zap.String("groupId", groupId))
	return nil
}

func (n *null) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	n.logger.Info("NullProducer:AddMessageToTxn", zap.Any("msg", msg), zap.String("groupId", groupId), zap.String("metadata", *metadata))
	return nil
}
