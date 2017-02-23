package katsubushi

import (
	"strconv"
	"time"

	"github.com/Songmu/retry"
	"github.com/bradfitz/gomemcache/memcache"
	"github.com/pkg/errors"
)

// Client is katsubushi client
type Client struct {
	memcacheClients []*memcache.Client
}

// NewClient creates Client
func NewClient(addrs []string) *Client {
	c := &Client{
		memcacheClients: make([]*memcache.Client, 0, len(addrs)),
	}
	for _, addr := range addrs {
		c.memcacheClients = append(c.memcacheClients, memcache.New(addr))
	}
	return c
}

// Fetch fetches id from katubushi
func (c *Client) Fetch() (uint64, error) {
	errs := errors.New("no servers available")
	for _, mc := range c.memcacheClients {
		var item *memcache.Item
		err := retry.Retry(2, 1*time.Millisecond, func() error {
			var _err error
			item, _err = mc.Get("id")
			return _err
		})
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
			continue
		}
		if id, err := strconv.ParseUint(string(item.Value), 10, 64); err == nil && id > 0 {
			return id, nil
		} else {
			errs = errors.Wrap(errs, err.Error())
		}
	}
	return 0, errs
}

// FetchMulti fetches multiple ids from katubushi
func (c *Client) FetchMulti(n int) ([]uint64, error) {
	keys := make([]string, 0, n)
	ids := make([]uint64, 0, n)
	for i := 0; i < n; i++ {
		keys = append(keys, strconv.Itoa(i))
	}

	errs := errors.New("no servers available")

	for _, mc := range c.memcacheClients {
		var items map[string]*memcache.Item
		err := retry.Retry(2, 1*time.Millisecond, func() error {
			var _err error
			items, _err = mc.GetMulti(keys)
			return _err
		})
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
			continue
		}
		for _, item := range items {
			if id, err := strconv.ParseUint(string(item.Value), 10, 64); err == nil && id > 0 {
				ids = append(ids, id)
				if len(ids) == n {
					return ids, nil
				}
			} else {
				errs = errors.Wrap(errs, err.Error())
			}
		}
	}
	return ids, errs
}
