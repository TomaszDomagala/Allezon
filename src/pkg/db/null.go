package db

import (
	"time"

	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

type nullClient struct {
	logger *zap.Logger
}

type nullUserProfileClient struct {
	logger *zap.Logger
}

func (n *nullUserProfileClient) RemoveOverLimit(cookie string, action types.Action, limit int) error {
	n.logger.Debug("null user profile client invoked", zap.String("method", "RemoveOverLimit"), zap.String("cookie", cookie), zap.String("action", action.String()), zap.Int("limit", limit))
	return nil
}

func (n *nullUserProfileClient) Get(cookie string) (UserProfile, error) {
	n.logger.Debug("null user profile client invoked", zap.String("method", "Get"), zap.String("cookie", cookie))
	return UserProfile{}, nil
}

func (n *nullUserProfileClient) Add(tag types.UserTag) (int, error) {
	n.logger.Debug("null user profile client invoked", zap.String("method", "Add"), zap.Any("tag", tag))
	return 0, nil
}

type nullAggregatesClient struct {
	logger *zap.Logger
}

func (n *nullAggregatesClient) Get(time time.Time, action types.Action) ([]ActionAggregates, error) {
	n.logger.Debug("null aggregates client invoked", zap.String("method", "Get"), zap.Time("time", time), zap.String("action", action.String()))
	return nil, nil
}

func (n *nullAggregatesClient) Add(key AggregateKey, tag types.UserTag) error {
	n.logger.Debug("null aggregates client invoked", zap.String("method", "Add"), zap.Any("key", key), zap.Any("tag", tag))
	return nil
}

func (n *nullClient) UserProfiles() UserProfileClient {
	return &nullUserProfileClient{logger: n.logger}
}

func (n *nullClient) Aggregates() AggregatesClient {
	return &nullAggregatesClient{logger: n.logger}
}

func NewNullClient(logger *zap.Logger) Client {
	return &nullClient{
		logger: logger,
	}
}
