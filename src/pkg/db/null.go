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

func (n *nullUserProfileClient) Get(cookie string) (GetResult[UserProfile], error) {
	n.logger.Debug("null user profile client invoked", zap.String("method", "Get"), zap.String("cookie", cookie))
	return GetResult[UserProfile]{}, nil
}

func (n *nullUserProfileClient) Update(cookie string, userProfile UserProfile, generation Generation) error {
	n.logger.Debug("null user profile client invoked", zap.String("method", "Update"), zap.String("cookie", cookie), zap.Any("userProfile", userProfile), zap.Uint32("generation", uint32(generation)))
	return nil
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
