package idGetter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/TomaszDomagala/Allezon/src/cmd/id_getter/api"
	"net/http"
	"sync"
)

const (
	OriginCollection   = "origin"
	BrandCollection    = "brand"
	CategoryCollection = "category"
)

type Client interface {
	GetId(collectionName string, element string) (id int32, err error)
}

type client struct {
	httpClient http.Client
	addr       string

	rwLock sync.RWMutex
	cache  map[string]map[string]int32
}

func (c *client) GetId(collectionName string, element string) (int32, error) {
	id, ok := c.getFromCache(collectionName, element)
	if ok {
		return id, nil
	}
	id, err := c.getIdFromServer(collectionName, element)
	if err != nil {
		return id, fmt.Errorf("error gettind id from server, %w", err)
	}
	c.saveInCache(collectionName, element, id)

	return id, nil
}

func (c *client) getIdFromServer(collectionName string, element string) (int32, error) {
	body, err := json.Marshal(api.GetIdRequest{
		CollectionName: collectionName,
		Element:        element,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to marshall body, %w", err)
	}

	resp, err := c.httpClient.Post(fmt.Sprintf("http://%s%s", c.addr, api.GetIdUrl), "application/json", bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("failed to make request to ip_getter, %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("ip_getter%s return not OK code %d with status %s", api.GetIdUrl, resp.StatusCode, resp.Status)
	}

	var res api.GetIdResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, fmt.Errorf("failed to unmarshall body, %w", err)
	}
	return res.Id, nil
}

func (c *client) getFromCache(name string, element string) (int32, bool) {
	c.rwLock.RLock()
	defer c.rwLock.RUnlock()

	if cache, ok := c.cache[name]; ok {
		idx, ok := cache[element]
		return idx, ok
	}
	return 0, false
}

func (c *client) saveInCache(name string, element string, id int32) {
	c.rwLock.Lock()
	defer c.rwLock.Unlock()

	if cache, ok := c.cache[name]; ok {
		cache[element] = id
	} else {
		c.cache[name] = map[string]int32{element: id}
	}
}

func NewClient(cl http.Client, addr string) Client {
	return &client{
		httpClient: cl,
		addr:       addr,
	}
}
