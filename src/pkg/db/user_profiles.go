package db

import (
	"encoding/json"
	"fmt"
	as "github.com/aerospike/aerospike-client-go/v6"
)

const userProfilesNamespace = "user_profiles"
const userProfilesViewsBin = "views"
const userProfilesBuysBin = "buys"

type userProfilesGetter struct {
	cl *as.Client
}

func (g userProfilesGetter) Get(cookie string) (res GetResult[UserProfile], err error) {
	// TODO Get in case of update may be optimized to only Get affected bin.
	key, err := as.NewKey(userProfilesNamespace, "", cookie)
	if err != nil {
		return res, err
	}
	r, err := g.cl.Get(nil, key, userProfilesBuysBin, userProfilesViewsBin)
	if err != nil {
		return res, err
	}
	if r == nil {
		return res, fmt.Errorf("user profiles for cookie %s not found, %w", cookie, KeyNotFoundError)
	}

	if views, ok := r.Bins[userProfilesViewsBin].([]byte); ok {
		if err = json.Unmarshal(views, &res.Result.Views); err != nil {
			return res, fmt.Errorf("couldn't unmarshall views")
		}
	} else {
		return res, fmt.Errorf("views have wrong type: %T", views)
	}

	if buys, ok := r.Bins[userProfilesBuysBin].([]byte); ok {
		if err = json.Unmarshal(buys, &res.Result.Buys); err != nil {
			return res, fmt.Errorf("couldn't unmarshall buys")
		}
	} else {
		return res, fmt.Errorf("buys have wrong type: %T", buys)
	}

	res.Generation = r.Generation

	return
}

func (g getter) UserProfiles() UserProfileGetter {
	return userProfilesGetter{cl: g.cl}
}

type userProfileModifier struct {
	cl *as.Client
}

func (u userProfileModifier) Get(cookie string) (GetResult[UserProfile], error) {
	return userProfilesGetter{cl: u.cl}.Get(cookie)
}

func (u userProfileModifier) Update(cookie string, userProfile UserProfile, generation Generation) error {
	// TODO Update may only update affected bin.
	key, ae := as.NewKey(userProfilesNamespace, "", cookie)
	if ae != nil {
		return ae
	}

	policy := as.NewWritePolicy(generation, 0)
	policy.RecordExistsAction = as.UPDATE

	views, err := json.Marshal(userProfile.Views)
	if err != nil {
		return fmt.Errorf("couldn't marshal views")
	}

	buys, err := json.Marshal(userProfile.Buys)
	if err != nil {
		return fmt.Errorf("couldn't marshal buys")
	}

	return u.cl.Put(policy, key, as.BinMap{userProfilesBuysBin: buys, userProfilesViewsBin: views})
}

func (u userProfileModifier) Add(cookie string, userProfile UserProfile) error {
	return u.Update(cookie, userProfile, 0)
}

func (m modifier) UserProfiles() UserProfileModifier {
	return userProfileModifier{cl: m.cl}
}
