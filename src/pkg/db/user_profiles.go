package db

import (
	"encoding/json"
	"fmt"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-client-go/v6/types"
)

const userProfilesSet = "user_profiles"
const userProfilesViewsBin = "views"
const userProfilesBuysBin = "buys"

type userProfileClient struct {
	cl *as.Client
}

func (u userProfileClient) Get(cookie string) (res GetResult[UserProfile], err error) {
	// TODO Get in case of update may be optimized to only Get affected bin.
	key, err := as.NewKey(AllezonNamespace, userProfilesSet, cookie)
	if err != nil {
		return res, err
	}
	r, err := u.cl.Get(nil, key, userProfilesBuysBin, userProfilesViewsBin)
	if err != nil {
		return res, fmt.Errorf("failed to get user profiles, %w", err)
	}
	if r == nil {
		return res, fmt.Errorf("user profiles for cookie %s not found, %w", cookie, KeyNotFoundError)
	}

	if views, ok := r.Bins[userProfilesViewsBin].([]byte); ok {
		if err = json.Unmarshal(views, &res.Result.Views); err != nil {
			return res, fmt.Errorf("couldn't unmarshall views, %w", err)
		}
	} else {
		return res, fmt.Errorf("views have wrong type: %T", r.Bins[userProfilesViewsBin])
	}

	if buys, ok := r.Bins[userProfilesBuysBin].([]byte); ok {
		if err = json.Unmarshal(buys, &res.Result.Buys); err != nil {
			return res, fmt.Errorf("couldn't unmarshall buys, %w", err)
		}
	} else {
		return res, fmt.Errorf("buys have wrong type: %T", r.Bins[userProfilesBuysBin])
	}

	res.Generation = r.Generation

	return
}

func (u userProfileClient) Update(cookie string, userProfile UserProfile, generation Generation) error {
	// TODO Update may only update affected bin.
	key, ae := as.NewKey(AllezonNamespace, userProfilesSet, cookie)
	if ae != nil {
		return ae
	}

	policy := as.NewWritePolicy(generation, as.TTLServerDefault)
	policy.RecordExistsAction = as.UPDATE
	policy.GenerationPolicy = as.EXPECT_GEN_EQUAL

	views, err := json.Marshal(userProfile.Views)
	if err != nil {
		return fmt.Errorf("couldn't marshal views, %w", err)
	}

	buys, err := json.Marshal(userProfile.Buys)
	if err != nil {
		return fmt.Errorf("couldn't marshal buys, %w", err)
	}

	if putErr := u.cl.Put(policy, key, as.BinMap{userProfilesBuysBin: buys, userProfilesViewsBin: views}); putErr != nil {
		if putErr.Matches(types.GENERATION_ERROR) {
			return fmt.Errorf("%w while trying to update %s, %s", GenerationMismatch, cookie, putErr)
		}
		return fmt.Errorf("error while trying to update %s, %w", cookie, putErr)
	}
	return nil
}

func (c client) UserProfiles() UserProfileClient {
	return userProfileClient{cl: c.cl}
}
