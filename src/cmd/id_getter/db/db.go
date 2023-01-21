package db

import (
	"errors"
	"fmt"
	as "github.com/aerospike/aerospike-client-go/v6"
	"github.com/aerospike/aerospike-client-go/v6/types"
)

var KeyNotFoundError = errors.New("key not found")
var ElementExists = errors.New("element exists")

type Host = as.Host
type ClientPolicy = as.ClientPolicy

type Client interface {
	Get(name string) ([]string, error)
	Append(name string, el string) (newLen int, err error)
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

const namespace = "ids"
const bin = "ids"

type client struct {
	cl *as.Client
}

func (c client) Append(name string, el string) (int, error) {
	key, ae := as.NewKey(namespace, "", name)
	if ae != nil {
		return 0, ae
	}

	policy := as.NewWritePolicy(0, as.TTLServerDefault)
	policy.RecordExistsAction = as.UPDATE

	r, createErr := c.cl.Operate(policy, key, as.ListAppendWithPolicyOp(as.NewListPolicy(as.ListOrderUnordered, as.ListWriteFlagsAddUnique), bin, el))
	if createErr != nil {
		if createErr.Matches(types.FAIL_ELEMENT_EXISTS) {
			return 0, fmt.Errorf("%w while trying to append to %s, %s", ElementExists, name, createErr)
		}
		return 0, fmt.Errorf("error while trying to append to %s, %w", name, createErr)
	}

	newLen := r.Bins[bin]
	if nL, ok := newLen.(int); ok {
		return nL, nil
	}
	return 0, fmt.Errorf("unexpected type of new array length, %T", newLen)
}

func (c client) Get(name string) (res []string, err error) {
	key, err := as.NewKey(namespace, "", name)
	if err != nil {
		return res, err
	}
	r, err := c.cl.Get(nil, key, bin)
	if err != nil {
		return res, fmt.Errorf("failed to get user profiles, %w", err)
	}
	if r == nil {
		return res, fmt.Errorf("ids for name %s not found, %w", name, KeyNotFoundError)
	}

	if els, ok := r.Bins[bin].([]interface{}); ok {
		res = make([]string, len(els))
		for i, e := range els {
			if s, ok := e.(string); ok {
				res[i] = s
			} else {
				return res, fmt.Errorf("list element '%s' at index %d has unexpected type: %T", e, i, e)
			}
		}
	} else {
		return res, fmt.Errorf("els have wrong type: %T", r.Bins[bin])
	}

	return
}
