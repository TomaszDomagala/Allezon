package db

import (
	"go.uber.org/zap"
	"time"
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

func (n *nullAggregatesClient) Get(minuteStart time.Time) (GetResult[Aggregates], error) {
	n.logger.Debug("null aggregates client invoked", zap.String("method", "Get"), zap.Time("minuteStart", minuteStart))
	return GetResult[Aggregates]{}, nil
}

func (n *nullAggregatesClient) Update(minuteStart time.Time, aggregates Aggregates, generation Generation) error {
	n.logger.Debug("null aggregates client invoked", zap.String("method", "Update"), zap.Time("minuteStart", minuteStart), zap.Any("aggregates", aggregates), zap.Uint32("generation", uint32(generation)))
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
