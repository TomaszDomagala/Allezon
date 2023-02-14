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

const namespace = "allezon"

type Client interface {
	GetElements(name string) ([]string, error)
	AppendElement(name string, el string) (newLen int, err error)
}

func NewClientFromAddresses(addresses ...string) (Client, error) {
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

const set = "ids"
const bin = "ids"

type client struct {
	cl *as.Client
}

// AppendElement appends element to the list of elements for given category, and returns new length of the list.
// If element already exists in the list, it returns ElementExists error.
func (c client) AppendElement(category string, element string) (int, error) {
	key, ae := as.NewKey(namespace, set, category)
	if ae != nil {
		return 0, ae
	}

	policy := as.NewWritePolicy(0, as.TTLDontExpire)
	policy.RecordExistsAction = as.UPDATE

	r, createErr := c.cl.Operate(policy, key, as.ListAppendWithPolicyOp(as.NewListPolicy(as.ListOrderUnordered, as.ListWriteFlagsAddUnique), bin, element))
	if createErr != nil {
		if createErr.Matches(types.FAIL_ELEMENT_EXISTS) {
			return 0, fmt.Errorf("%w while trying to append to %s, %s", ElementExists, category, createErr)
		}
		return 0, fmt.Errorf("error while trying to append to %s, %w", category, createErr)
	}

	newLen := r.Bins[bin]
	if nL, ok := newLen.(int); ok {
		return nL, nil
	}
	return 0, fmt.Errorf("unexpected type of new array length, %T", newLen)
}

// GetElements returns all elements for given category.
func (c client) GetElements(category string) (elements []string, err error) {
	key, err := as.NewKey(namespace, set, category)
	if err != nil {
		return elements, err
	}
	r, getError := c.cl.Get(nil, key, bin)
	if getError != nil {
		if getError.Matches(types.KEY_NOT_FOUND_ERROR) {
			return elements, fmt.Errorf("ids for category %s not found, %w", category, KeyNotFoundError)
		}
		return elements, fmt.Errorf("failed to get elements for category %s, %w", category, err)
	}

	if els, ok := r.Bins[bin].([]interface{}); ok {
		elements = make([]string, len(els))
		for i, e := range els {
			if s, ok := e.(string); ok {
				elements[i] = s
			} else {
				return elements, fmt.Errorf("list element '%s' at index %d has unexpected type: %T", e, i, e)
			}
		}
	} else {
		return elements, fmt.Errorf("els have wrong type: %T", r.Bins[bin])
	}

	return
}
