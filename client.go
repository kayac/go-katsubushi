package katsubushi

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/Songmu/retry"
)

// DefaultClientTimeout is default timeout for katsubushi client
var DefaultClientTimeout = 5 * time.Second

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
	c.SetTimeout(DefaultClientTimeout)
	return c
}

// SetTimeout sets timeout to katsubushi servers
func (c *Client) SetTimeout(t time.Duration) {
	for _, mc := range c.memcacheClients {
		mc.SetTimeout(t)
	}
}

// Fetch fetches id from katsubushi
func (c *Client) Fetch(ctx context.Context) (uint64, error) {
	errs := fmt.Errorf("no servers available")
	for _, mc := range c.memcacheClients {
		var id uint64
		err := retry.Retry(2, 0, func() error {
			var _err error
			id, _err = mc.Get(ctx, "id")
			return _err
		})
		if err != nil {
			errs = fmt.Errorf("%s: %w", err.Error(), errs)
			continue
		}
		return id, nil
	}
	return 0, errs
}

// FetchMulti fetches multiple ids from katsubushi
func (c *Client) FetchMulti(ctx context.Context, n int) ([]uint64, error) {
	keys := make([]string, 0, n)

	for i := range n {
		keys = append(keys, strconv.Itoa(i))
	}

	errs := fmt.Errorf("no servers available")

	for _, mc := range c.memcacheClients {
		var ids []uint64
		err := retry.Retry(2, 0, func() error {
			var _err error
			ids, _err = mc.GetMulti(ctx, keys)
			return _err
		})
		if err != nil {
			errs = fmt.Errorf("%s: %w", err.Error(), errs)
			continue
		}
		return ids, nil
	}
	return nil, errs
}
