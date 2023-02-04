package db

import (
	"errors"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

// AllezonNamespace is the one and only namespace used by Allezon.
const AllezonNamespace = "allezon"

var KeyNotFoundError = errors.New("key not found")
var GenerationMismatch = errors.New("generation mismatch")

type Generation = uint32

type GetResult[T any] struct {
	Generation Generation
	Result     T
}

type UserProfileClient interface {
	Get(cookie string) (GetResult[UserProfile], error)
	Update(cookie string, userProfile UserProfile, generation Generation) error
}

type UserProfile struct {
	Views []types.UserTag
	Buys  []types.UserTag
}

type ActionAggregates struct {
	CategoryId uint16
	BrandId    uint8
	Origin     uint8
	Data       uint32
}

type TypeAggregates struct {
	Sum   []ActionAggregates
	Count []ActionAggregates
}

type Aggregates struct {
	Views TypeAggregates
	Buys  TypeAggregates
}

type AggregatesClient interface {
	Get(minuteStart time.Time) (GetResult[Aggregates], error)
	Update(minuteStart time.Time, aggregates Aggregates, generation Generation) error
}

type Client interface {
	UserProfiles() UserProfileClient
	Aggregates() AggregatesClient
}

type Host = as.Host
type ClientPolicy = as.ClientPolicy

type client struct {
	cl *as.Client
}

func NewClientFromAddresses(addresses []string) (Client, error) {
	hosts, err := as.NewHosts(addresses...)
	if err != nil {
		return nil, err
	}
	return NewClient(nil, hosts...)
}

func NewClient(clientPolicy *ClientPolicy, hosts ...*Host) (Client, error) {
	cl, err := as.NewClientWithPolicyAndHost(clientPolicy, hosts...)
	return client{cl: cl}, err
}
