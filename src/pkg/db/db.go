package db

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	as "github.com/aerospike/aerospike-client-go/v6"
	"time"
)

var KeyNotFoundError = fmt.Errorf("key not found")
var GenerationMismatch = fmt.Errorf("generation mismatch")

type Generation = int32

type GetResult[T any] struct {
	Generation Generation
	Result     T
}

type UserProfileModifier interface {
	UserProfileGetter
}

type UserProfile struct {
	Views []types.UserTag
	Buys  []types.UserTag
}

type UserProfileGetter interface {
	Get(cookie string) (GetResult[UserProfile], error)
	Update(cookie string, userProfile UserProfile, generation Generation) error
	Add(cookie string, userProfile UserProfile) error
}

type ActionAggregates struct {
	CategoryId uint16
	BrandId    uint8
	Origin     uint8
	Data       uint32
}

type TypeAggregates struct {
	Views ActionAggregates
	Buys  ActionAggregates
}

type Aggregates struct {
	Count TypeAggregates
	Sum   TypeAggregates
}

type AggregatesGetter interface {
	Get(minuteStart *time.Time) (GetResult[Aggregates], error)
}

type AggregatesModifier interface {
	AggregatesGetter
	Update(minuteStart *time.Time, aggregates Aggregates, generation Generation) error
	Add(minuteStart *time.Time, aggregates Aggregates) error
}

type Getter interface {
	UserProfiles() UserProfileGetter
	Aggregates() AggregatesGetter
}

type Modifier interface {
	UserProfiles() UserProfileModifier
	Aggregates() AggregatesModifier
}

type Host = as.Host
type ClientPolicy = as.ClientPolicy

type getter struct {
	cl *as.Client
}

type modifier struct {
	cl *as.Client
}

func NewGetterFromAddresses(addresses []string) (Getter, error) {
	hosts, err := as.NewHosts(addresses...)
	if err != nil {
		return nil, err
	}
	return NewGetter(nil, hosts...)
}

func NewGetter(clientPolicy *ClientPolicy, hosts ...*Host) (Getter, error) {
	cl, err := as.NewClientWithPolicyAndHost(clientPolicy, hosts...)
	return getter{cl: cl}, err
}

func NewModifierFromAddresses(addresses []string) (Modifier, error) {
	hosts, err := as.NewHosts(addresses...)
	if err != nil {
		return nil, err
	}
	return NewModifier(nil, hosts...)
}

func NewModifier(clientPolicy *ClientPolicy, hosts ...*Host) (Modifier, error) {
	cl, err := as.NewClientWithPolicyAndHost(clientPolicy, hosts...)
	return modifier{cl: cl}, err
}
