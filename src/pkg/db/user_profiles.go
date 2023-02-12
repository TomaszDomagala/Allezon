package db

import (
	"errors"
	"fmt"

	as "github.com/aerospike/aerospike-client-go/v6"
	asTypes "github.com/aerospike/aerospike-client-go/v6/types"
	"github.com/bytedance/sonic"
	"go.uber.org/zap"

	"github.com/TomaszDomagala/Allezon/src/pkg/types"
)

const (
	userProfilesNamespace = "allezon"

	userProfilesSet = "user_profiles"

	userProfilesViewsBin = "views"
	userProfilesBuysBin  = "buys"
)

type userProfileClient struct {
	cl *as.Client
	l  *zap.Logger
}

func marshallTag(tag *types.UserTag) ([]byte, error) {
	return sonic.ConfigFastest.Marshal(tag)
}

func unmarshallTag(data []byte, tag *types.UserTag) (err error) {
	return sonic.ConfigFastest.Unmarshal(data, tag)
}

func (u userProfileClient) decodeBin(up *[]types.UserTag, action types.Action, bins as.BinMap) error {
	binName := u.actionToBin(action)
	raw, ok := bins[binName]
	if !ok {
		return nil
	}
	pairs, ok := raw.([]as.MapPair)
	if !ok {
		return fmt.Errorf("bin %s has a wrong type: %T", binName, raw)
	}
	*up = make([]types.UserTag, len(pairs))
	for i, kv := range pairs {
		value, ok := kv.Value.([]byte)
		if !ok {
			return fmt.Errorf("unexpected type %T for key %s in bin %s", kv.Value, kv.Key, kv.Value)
		}
		if err := unmarshallTag(value, &(*up)[i]); err != nil {
			return fmt.Errorf("cannot unmarshall tag %s, %w", string(value), err)
		}
	}
	return nil
}

func (u userProfileClient) Get(cookie string) (up UserProfile, err error) {
	key, err := as.NewKey(userProfilesNamespace, userProfilesSet, cookie)
	if err != nil {
		return UserProfile{}, err
	}
	r, err := u.cl.Get(nil, key, userProfilesViewsBin, userProfilesBuysBin)
	if err != nil {
		if errors.Is(err, as.ErrKeyNotFound) {
			return UserProfile{}, fmt.Errorf("aggregates for minute %s not found, %w", cookie, KeyNotFoundError)
		}
		return UserProfile{}, fmt.Errorf("failed to get aggregates, %w", err)
	}

	if err := u.decodeBin(&up.Views, types.View, r.Bins); err != nil {
		return UserProfile{}, fmt.Errorf("error parsing views, %w", err)
	}
	if err := u.decodeBin(&up.Buys, types.Buy, r.Bins); err != nil {
		return UserProfile{}, fmt.Errorf("error parsing buys, %w", err)
	}
	return
}

func (u userProfileClient) Add(tag *types.UserTag) (int, error) {
	name := tag.Cookie
	key, ae := as.NewKey(userProfilesNamespace, userProfilesSet, name)
	if ae != nil {
		return 0, fmt.Errorf("error creating key %s, %w", name, ae)
	}
	marshalledTag, err := marshallTag(tag)
	if err != nil {
		return 0, fmt.Errorf("error marshalling tag %#v, %w", tag, err)
	}

	policy := as.NewWritePolicy(0, as.TTLServerDefault)
	policy.RecordExistsAction = as.UPDATE

	binName := u.actionToBin(tag.Action)
	mapPolicy := as.NewMapPolicy(as.MapOrder.KEY_ORDERED, as.MapWriteMode.UPDATE)
	increaseCountOp := as.MapPutOp(mapPolicy, binName, tag.Time.UnixMilli(), marshalledTag)

	r, err := u.cl.Operate(policy, key, increaseCountOp)
	if err != nil {
		return 0, fmt.Errorf("error while trying to add to user profiles, tag %#v, %w", tag, err)
	}
	newLen := r.Bins[binName]
	if nL, ok := newLen.(int); ok {
		return nL, nil
	}
	return 0, fmt.Errorf("unexpected type of new map length, %T", newLen)
}

func (u userProfileClient) RemoveOverLimit(cookie string, action types.Action, limit int) error {
	key, err := as.NewKey(userProfilesNamespace, userProfilesSet, cookie)
	if err != nil {
		return fmt.Errorf("error creating key %s, %w", cookie, err)
	}

	sizePolicy := as.NewWritePolicy(0, as.TTLServerDefault)
	sizePolicy.RecordExistsAction = as.UPDATE_ONLY

	binName := u.actionToBin(action)
	sizeOp := as.MapSizeOp(binName)
	r, err := u.cl.Operate(sizePolicy, key, sizeOp)
	if err != nil {
		if err.Matches(asTypes.KEY_NOT_FOUND_ERROR) {
			return nil
		}
		return fmt.Errorf("error getting size of action %s for cookie %s, %w", cookie, action, err)
	}
	lengthRaw, ok := r.Bins[binName]
	if !ok {
		return fmt.Errorf("bin %s missing", binName)
	}
	if lengthRaw == nil {
		// Bin not yet populated.
		return nil
	}
	length, ok := lengthRaw.(int)
	if !ok {
		return fmt.Errorf("length has unexpected type %T", lengthRaw)
	}
	toRemove := length - limit
	if toRemove <= 0 {
		return nil
	}

	removePolicy := as.NewWritePolicy(r.Generation, as.TTLServerDefault)
	removePolicy.RecordExistsAction = as.UPDATE_ONLY
	removePolicy.GenerationPolicy = as.EXPECT_GEN_EQUAL

	removeOp := as.MapRemoveByIndexRangeCountOp(binName, 0, toRemove, as.MapReturnType.COUNT)
	if _, err := u.cl.Operate(removePolicy, key, removeOp); err != nil {
		if err.Matches(asTypes.GENERATION_ERROR) {
			return fmt.Errorf("%w while trying to remove over limit %s, %s", GenerationMismatch, cookie, err)
		}
		return fmt.Errorf("error removing over limit of action %s for cookie %s, %w", cookie, action, err)
	}

	return nil
}

func (u userProfileClient) actionToBin(action types.Action) string {
	switch action {
	case types.Buy:
		return userProfilesBuysBin
	case types.View:
		return userProfilesViewsBin
	default:
		u.l.Fatal("unexpected value for action", zap.Int8("action", int8(action)))
		panic(nil)
	}
}

func (c client) UserProfiles() UserProfileClient {
	return userProfileClient(c)
}
