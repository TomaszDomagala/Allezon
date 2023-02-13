package idGetter

import "go.uber.org/zap"

type nullClient struct {
	logger *zap.Logger
}

func (n *nullClient) GetID(collectionName string, element string, createMissing bool) (id int32, err error) {
	n.logger.Debug("null client invoked", zap.String("method", "GetID"), zap.String("collectionName", collectionName), zap.String("element", element), zap.Bool("createMissing", createMissing))
	return 0, nil
}

func NewNullClient(logger *zap.Logger) Client {
	return &nullClient{logger: logger}
}
