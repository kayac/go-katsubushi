package katsubushi

import (
	"strconv"

	"github.com/Songmu/retry"
	"github.com/pkg/errors"
)

// Client is katsubushi client
type Client struct {
	memcacheClients []*memcacheClient
}

// NewClient creates Client
func NewClient(addrs ...string) *Client {
	c := &Client{
		memcacheClients: make([]*memcacheClient, 0, len(addrs)),
	}
	for _, addr := range addrs {
		c.memcacheClients = append(c.memcacheClients, newMemcacheClient(addr))
	}
	return c
}

// Fetch fetches id from katubushi
func (c *Client) Fetch() (uint64, error) {
	errs := errors.New("no servers available")
	for _, mc := range c.memcacheClients {
		var id uint64
		err := retry.Retry(2, 0, func() error {
			var _err error
			id, _err = mc.Get("id")
			return _err
		})
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
			continue
		}
		return id, nil
	}
	return 0, errs
}

// FetchMulti fetches multiple ids from katubushi
func (c *Client) FetchMulti(n int) ([]uint64, error) {
	keys := make([]string, 0, n)

	for i := 0; i < n; i++ {
		keys = append(keys, strconv.Itoa(i))
	}

	errs := errors.New("no servers available")

	for _, mc := range c.memcacheClients {
		var ids []uint64
		err := retry.Retry(2, 0, func() error {
			var _err error
			ids, _err = mc.GetMulti(keys)
			return _err
		})
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
			continue
		}
		return ids, nil
	}
	return nil, errs
}
