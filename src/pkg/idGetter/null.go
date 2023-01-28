package idGetter

import "go.uber.org/zap"

type nullClient struct {
	logger *zap.Logger
}

func (n *nullClient) GetId(collectionName string, element string) (id int32, err error) {
	n.logger.Debug("null client invoked", zap.String("method", "GetId"), zap.String("collectionName", collectionName), zap.String("element", element))
	return 0, nil
}

func NewNullClient(logger *zap.Logger) Client {
	return &nullClient{logger: logger}
}
