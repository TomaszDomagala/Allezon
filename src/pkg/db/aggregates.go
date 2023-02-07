package db

import (
	"errors"
	"fmt"
	"time"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

const (
	// aggregatesSet is the name of the set used for storing aggregates.
	aggregatesSet = "aggregates"

	aggregatesViewsBin = "views"
	aggregatesBuysBin  = "buys"
)

// For some peculiar reason client devs decided that even though it's int64 in the db they are going to use int.
// Fortunately on Linux x86_64 it's 64bit.
type aerospikeInt = int

type aggregatesClient struct {
	cl *as.Client
	l  *zap.Logger
}

func timeToKey(t time.Time) string {
	return t.Format(time.RFC822)
}

func (a *AggregateKey) decode(key aerospikeInt) {
	a.Origin = uint16(key)
	key >>= 16
	a.BrandId = uint16(key)
	key >>= 16
	a.CategoryId = uint16(key)
}

func (a *AggregateKey) encode() aerospikeInt {
	return aerospikeInt(a.CategoryId)<<32 | aerospikeInt(a.BrandId)<<16 | aerospikeInt(a.Origin)
}

func encodeSumAndCount(price uint32) uint64 {
	return 1<<48 | uint64(price)
}

func decodeSumAndCount(v uint64) (sum uint64, count uint16) {
	count = uint16(v >> 48)
	sum = (v << 16) >> 16
	return
}

func (a aggregatesClient) Get(t time.Time) (res Aggregates, err error) {
	key, err := as.NewKey(AllezonNamespace, aggregatesSet, timeToKey(t))
	if err != nil {
		return res, err
	}
	r, err := a.cl.Get(nil, key, aggregatesViewsBin, aggregatesBuysBin)
	if err != nil {
		if errors.Is(err, as.ErrKeyNotFound) {
			return res, fmt.Errorf("aggregates for minute %s not found, %w", timeToKey(t), KeyNotFoundError)
		}
		return res, fmt.Errorf("failed to get aggregates, %w", err)
	}

	if vI, ok := r.Bins[aggregatesViewsBin]; ok {
		if views, ok := vI.([]as.MapPair); ok {
			res.Views, err = unmarshallActionAggregates(views)
			if err != nil {
				return res, fmt.Errorf("couldn't unmarshall views, %w", err)
			}
		} else {
			return res, fmt.Errorf("views have wrong type: %T", r.Bins[aggregatesViewsBin])
		}
	}

	if bI, ok := r.Bins[aggregatesBuysBin]; ok {
		if buys, ok := bI.([]as.MapPair); ok {
			res.Buys, err = unmarshallActionAggregates(buys)
			if err != nil {
				return res, fmt.Errorf("couldn't unmarshall buys, %w", err)
			}
		} else {
			return res, fmt.Errorf("buys have wrong type: %T", r.Bins[aggregatesBuysBin])
		}
	}

	return
}

func unmarshallActionAggregates(kvs []as.MapPair) ([]ActionAggregates, error) {
	res := make([]ActionAggregates, 0, len(kvs))
	for _, kv := range kvs {
		kI, ok := kv.Key.(aerospikeInt)
		if !ok {
			return nil, fmt.Errorf(`key "%s" is not an %T but %T`, kv.Key, kI, kv.Key)
		}
		vI, ok := kv.Value.(uint64) // No idea why it's uint64 and not aerospikeInt.
		if !ok {
			return nil, fmt.Errorf(`value "%s" is not an %T but %T`, kv.Value, vI, kv.Value)
		}

		var a ActionAggregates
		a.Sum, a.Count = decodeSumAndCount(vI)
		a.Key.decode(kI)

		res = append(res, a)
	}
	return res, nil
}

func (a aggregatesClient) actionToBin(action types.Action) string {
	switch action {
	case types.Buy:
		return aggregatesBuysBin
	case types.View:
		return aggregatesViewsBin
	default:
		a.l.Fatal("unexpected value for action", zap.Int8("action", int8(action)))
		panic(nil)
	}
}

func (a aggregatesClient) binToAction(bin string) types.Action {
	switch bin {
	case aggregatesBuysBin:
		return types.Buy
	case aggregatesViewsBin:
		return types.View
	default:
		a.l.Fatal("unexpected bin", zap.String("bin", bin))
		panic(nil)
	}
}

func (a aggregatesClient) Add(aKey AggregateKey, tag types.UserTag) error {
	name := timeToKey(tag.Time)
	key, ae := as.NewKey(AllezonNamespace, aggregatesSet, name)
	if ae != nil {
		return ae
	}

	policy := as.NewWritePolicy(0, as.TTLServerDefault)
	policy.RecordExistsAction = as.UPDATE

	binName := a.actionToBin(tag.Action)
	mapPolicy := as.NewMapPolicy(as.MapOrder.KEY_ORDERED, as.MapWriteMode.UPDATE)
	increaseCountOp := as.MapIncrementOp(mapPolicy, binName, aKey.encode(), encodeSumAndCount(tag.ProductInfo.Price))

	if _, err := a.cl.Operate(policy, key, increaseCountOp); err != nil {
		return fmt.Errorf("error while trying to add to aggregates, time: %s, aKey: %s, price: %d, action %s, %w", name, spew.Sprint(aKey), tag.ProductInfo.Price, tag.Action, err)
	}
	return nil
}

func (c client) Aggregates() AggregatesClient {
	return aggregatesClient(c)
}
