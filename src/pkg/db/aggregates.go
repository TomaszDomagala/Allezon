package db

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	as "github.com/aerospike/aerospike-client-go/v6"
	asTypes "github.com/aerospike/aerospike-client-go/v6/types"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const (
	aggregatesNamespace = "allezon"

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

func toKey(ts int64, key AggregateKey) string {
	return fmt.Sprintf("%d_%d", ts, key.encode())
}

func toTs(t time.Time) int64 {
	return t.Unix() / 60
}

func toSet(ts int64) string {
	const numSets = 1000
	return strconv.Itoa(int(ts % numSets))
}

func (a *AggregateKey) decode(key uint64) {
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

func (a aggregatesClient) Get(t time.Time, action types.Action) (agg []ActionAggregates, err error) {
	ts := toTs(t)
	if err != nil {
		return nil, err
	}

	binName := a.actionToBin(action)

	sP := as.NewScanPolicy()
	sP.MaxRetries = 0
	rs, err := a.cl.ScanAll(sP, aggregatesNamespace, toSet(ts), binName)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregates, %w", err)
	}
	defer func() {
		if err := rs.Close(); err != nil {
			a.l.Warn("error closing record set", zap.Error(err))
		}
	}()
	for r := range rs.Results() {
		if r.Err != nil {
			return nil, fmt.Errorf("error parsing aggregates, %w", r.Err)
		}
		raw, ok := r.Record.Bins[binName]
		if !ok {
			continue
		}
		sumCount, ok := raw.(aerospikeInt)
		if !ok {
			return nil, fmt.Errorf(`bin "%s" is not an %T but %T`, binName, sumCount, raw)
		}
		rTs, key, err := a.decodeKey(r.Record.Key)
		if err != nil {
			return nil, fmt.Errorf("error parsing key, %w", err)
		}
		if ts != rTs {
			continue
		}

		var a ActionAggregates
		a.Sum, a.Count = decodeSumAndCount(uint64(sumCount))
		a.Key.decode(key)

		agg = append(agg, a)
	}
	return
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

func (a aggregatesClient) Add(aKey AggregateKey, tag types.UserTag) error {
	ts := toTs(tag.Time)
	name := toKey(ts, aKey)
	key, ae := as.NewKey(aggregatesNamespace, toSet(ts), name)
	if ae != nil {
		return ae
	}

	updatePolicy := as.NewWritePolicy(0, as.TTLServerDefault)
	updatePolicy.RecordExistsAction = as.UPDATE_ONLY

	binName := a.actionToBin(tag.Action)
	encoded := encodeSumAndCount(tag.ProductInfo.Price)
	bin := as.Bin{
		Name:  binName,
		Value: as.NewLongValue(int64(encoded)),
	}
	increaseCountOp := as.AddOp(&bin)

	if _, err := a.cl.Operate(updatePolicy, key, increaseCountOp); err != nil {
		if err.Matches(asTypes.KEY_NOT_FOUND_ERROR) {
			createPolicy := as.NewWritePolicy(0, as.TTLServerDefault)
			createPolicy.RecordExistsAction = as.CREATE_ONLY
			createPolicy.SendKey = true
			if err := a.cl.Put(createPolicy, key, as.BinMap{
				binName: int64(encoded),
			}); err != nil {
				return fmt.Errorf("error while trying to add to aggregates, time: %s, aKey: %s, price: %d, action %s, %w", name, spew.Sprint(aKey), tag.ProductInfo.Price, tag.Action, err)
			}
		} else {
			return fmt.Errorf("error while trying to add to aggregates, time: %s, aKey: %s, price: %d, action %s, %w", name, spew.Sprint(aKey), tag.ProductInfo.Price, tag.Action, err)
		}
	}

	return nil
}

func (a aggregatesClient) decodeKey(key *as.Key) (ts int64, aKey uint64, err error) {
	split := strings.Split(key.Value().String(), "_")
	if len(split) != 2 {
		return 0, 0, fmt.Errorf("wrong key format %s", split)
	}
	ts, err = strconv.ParseInt(split[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	aKey, err = strconv.ParseUint(split[1], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	return
}

func (c client) Aggregates() AggregatesClient {
	return aggregatesClient(c)
}
