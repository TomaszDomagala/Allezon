package db

import (
	"errors"
	"fmt"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

var KeyNotFoundError = errors.New("key not found")
var GenerationMismatch = errors.New("generation mismatch")

type UserProfileClient interface {
	Get(cookie string) (UserProfile, error)
	Add(tag *types.UserTag) (newLen int, err error)
	RemoveOverLimit(cookie string, action types.Action, limit int) error
}

// UserProfile holds data about users views and buys.
// Tags are sorted in ascending order relative to time.
type UserProfile struct {
	Views []types.UserTag
	Buys  []types.UserTag
}

type AggregateKey struct {
	CategoryId uint16
	BrandId    uint16
	Origin     uint16
}

type ActionAggregates struct {
	Key   AggregateKey
	Sum   uint64
	Count uint16
}

type AggregatesClient interface {
	Get(time time.Time, action types.Action) ([]ActionAggregates, error)
	Add(key AggregateKey, tag types.UserTag) error
}

type Client interface {
	UserProfiles() UserProfileClient
	Aggregates() AggregatesClient
}

type Host = as.Host
type ClientPolicy = as.ClientPolicy

type client struct {
	cl *as.Client
	l  *zap.Logger
}

func NewClientFromAddresses(logger *zap.Logger, addresses ...string) (Client, error) {
	hosts, err := as.NewHosts(addresses...)
	if err != nil {
		return nil, fmt.Errorf("error getting hosts from addresses, %w", err)
	}
	return NewClient(nil, logger, hosts...)
}

func NewClient(clientPolicy *ClientPolicy, logger *zap.Logger, hosts ...*Host) (Client, error) {
	cl, err := as.NewClientWithPolicyAndHost(clientPolicy, hosts...)
	return client{cl: cl, l: logger}, err
}
