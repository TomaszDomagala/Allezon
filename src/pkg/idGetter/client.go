package idGetter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/api"
)

const (
	OriginCollection   = "origin"
	BrandCollection    = "brand"
	CategoryCollection = "category"
)

type Client interface {
	GetID(collection string, element string) (id int32, err error)
}

type client struct {
	httpClient http.Client
	addr       string

	cacheEnabled bool
	rwLock       sync.RWMutex
	cache        map[string]map[string]int32
}

func (c *client) GetID(collectionName string, element string) (int32, error) {
	id, ok := c.getFromCache(collectionName, element)
	if ok {
		return id, nil
	}
	id, err := c.getIDFromServer(collectionName, element)
	if err != nil {
		return id, fmt.Errorf("error getting id from the server, %w", err)
	}
	c.saveInCache(collectionName, element, id)

	return id, nil
}

func (c *client) getIDFromServer(collectionName string, element string) (int32, error) {
	body, err := json.Marshal(api.GetIDRequest{
		CollectionName: collectionName,
		Element:        element,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshall body, %w", err)
	}

	resp, err := c.httpClient.Post(fmt.Sprintf("http://%s%s", c.addr, api.GetIDUrl), "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to make request to ip_getter, %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ip_getter%s return not OK code %d with status %s", api.GetIDUrl, resp.StatusCode, resp.Status)
	}

	var res api.GetIdResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, fmt.Errorf("failed to unmarshall body, %w", err)
	}
	return res.ID, nil
}

func (c *client) getFromCache(name string, element string) (int32, bool) {
	if !c.cacheEnabled {
		return 0, false
	}

	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	if cache, ok := c.cache[name]; ok {
		idx, ok := cache[element]
		return idx, ok
	}
	return 0, false
}

func (c *client) saveInCache(name string, element string, id int32) {
	if !c.cacheEnabled {
		return
	}

	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if cache, ok := c.cache[name]; ok {
		cache[element] = id
	} else {
		c.cache[name] = map[string]int32{element: id}
	}
}

// NewClient returns a client with enabled cache.
func NewClient(cl http.Client, addr string) Client {
	return &client{
		httpClient:   cl,
		addr:         addr,
		cache:        make(map[string]map[string]int32),
		cacheEnabled: true,
	}
}

// NewPureClient returns a client with disabled cache.
func NewPureClient(cl http.Client, addr string) Client {
	return &client{
		httpClient:   cl,
		addr:         addr,
		cacheEnabled: false,
	}
}
