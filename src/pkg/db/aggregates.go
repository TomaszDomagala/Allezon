package db

import (
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/pkg/db/pb"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-client-go/v6/types"
	"google.golang.org/protobuf/proto"
	"time"
)

//go:generate protoc -I=./ --go_out=./ ./aggregates.proto

const aggregatesNamespace = "aggregates"
const aggregatesViewsBin = "views"
const aggregatesBuysBin = "buys"

type aggregatesClient struct {
	cl *as.Client
}

func timeToKey(t time.Time) string {
	t = t.Add(-(time.Duration(t.Nanosecond()) + time.Second*time.Duration(t.Second())))
	return fmt.Sprint(t.Unix())
}

func (a *ActionAggregates) decode(v *pb.ActionAggregate) {
	key := v.SerializedKey
	a.Origin = uint8(key)
	key >>= 8
	a.BrandId = uint8(key)
	key >>= 8
	a.CategoryId = uint16(key)

	a.Data = v.Data
}

func (a *ActionAggregates) encode() *pb.ActionAggregate {
	return &pb.ActionAggregate{
		SerializedKey: uint32(a.CategoryId)<<16 | uint32(a.BrandId)<<8 | uint32(a.Origin),
		Data:          a.Data,
	}
}

func unmarshallTypeAggregates(data []byte, a *TypeAggregates) error {
	aPb := &pb.TypeAggregate{}
	if err := proto.Unmarshal(data, aPb); err != nil {
		return err
	}

	a.Sum = make([]ActionAggregates, len(aPb.Sum))
	for i, v := range aPb.Sum {
		a.Sum[i].decode(v)
	}

	a.Count = make([]ActionAggregates, len(aPb.Count))
	for i, v := range aPb.Count {
		a.Count[i].decode(v)
	}

	return nil
}

func marshallTypeAggregates(t TypeAggregates) ([]byte, error) {
	aPb := &pb.TypeAggregate{
		Count: make([]*pb.ActionAggregate, len(t.Count)),
		Sum:   make([]*pb.ActionAggregate, len(t.Sum)),
	}

	for i, v := range t.Count {
		aPb.Count[i] = v.encode()
	}

	for i, b := range t.Sum {
		aPb.Sum[i] = b.encode()
	}

	return proto.Marshal(aPb)
}

func (a aggregatesClient) Get(minuteStart time.Time) (res GetResult[Aggregates], err error) {
	// TODO we can only get required bin.
	key, err := as.NewKey(aggregatesNamespace, "", timeToKey(minuteStart))
	if err != nil {
		return res, err
	}
	r, err := a.cl.Get(nil, key, aggregatesViewsBin, aggregatesBuysBin)
	if err != nil {
		return res, fmt.Errorf("failed to get aggregates, %w", err)
	}
	if r == nil {
		return res, fmt.Errorf("aggregates for minute %s not found, %w", timeToKey(minuteStart), KeyNotFoundError)
	}

	if views, ok := r.Bins[aggregatesViewsBin].([]byte); ok {
		if err = unmarshallTypeAggregates(views, &res.Result.Views); err != nil {
			return res, fmt.Errorf("couldn't unmarshall views, %w", err)
		}
	} else {
		return res, fmt.Errorf("views have wrong type: %T", r.Bins[aggregatesViewsBin])
	}

	if buys, ok := r.Bins[aggregatesBuysBin].([]byte); ok {
		if err = unmarshallTypeAggregates(buys, &res.Result.Buys); err != nil {
			return res, fmt.Errorf("couldn't unmarshall buys, %w", err)
		}
	} else {
		return res, fmt.Errorf("buys have wrong type: %T", r.Bins[aggregatesBuysBin])
	}

	res.Generation = r.Generation

	return
}

func (a aggregatesClient) Update(minuteStart time.Time, aggregates Aggregates, generation Generation) error {
	// TODO we can only update required bin.
	name := timeToKey(minuteStart)
	key, ae := as.NewKey(aggregatesNamespace, "", name)
	if ae != nil {
		return ae
	}

	policy := as.NewWritePolicy(generation, as.TTLServerDefault)
	policy.RecordExistsAction = as.UPDATE

	views, err := marshallTypeAggregates(aggregates.Views)
	if err != nil {
		return fmt.Errorf("couldn't marshal views, %w", err)
	}
	buys, err := marshallTypeAggregates(aggregates.Buys)
	if err != nil {
		return fmt.Errorf("couldn't marshal buys, %w", err)
	}

	if putErr := a.cl.Put(policy, key, as.BinMap{aggregatesBuysBin: buys, aggregatesViewsBin: views}); putErr != nil {
		if putErr.Matches(types.GENERATION_ERROR) {
			return fmt.Errorf("%w while trying to update %s, %s", GenerationMismatch, name, putErr)
		}
		return fmt.Errorf("error while trying to update %s, %w", name, putErr)
	}
	return nil
}

func (c client) Aggregates() AggregatesClient {
	return aggregatesClient{cl: c.cl}
}
