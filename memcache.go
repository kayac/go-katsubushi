package katsubushi

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strconv"
	"sync"
	"time"
)

const memcacheDefaultTimeout = 1 * time.Second

type memcacheClient struct {
	addr    string
	conn    net.Conn
	timeout time.Duration
	mu      sync.Mutex
	rw      *bufio.ReadWriter
}

func newMemcacheClient(addr string) *memcacheClient {
	return &memcacheClient{
		addr:    addr,
		timeout: memcacheDefaultTimeout,
	}
}

func (c *memcacheClient) SetTimeout(t time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.timeout = t
}

func (c *memcacheClient) connect(ctx context.Context) error {
	var err error
	d := net.Dialer{Timeout: c.timeout}
	c.conn, err = d.DialContext(ctx, "tcp", c.addr)
	c.rw = bufio.NewReadWriter(bufio.NewReader(c.conn), bufio.NewWriter(c.conn))
	return err
}

func (c *memcacheClient) close() error {
	defer func() { c.conn = nil }()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *memcacheClient) Get(ctx context.Context, key string) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		if err := c.connect(ctx); err != nil {
			return 0, err
		}
	}
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	c.rw.Write(memdGets)
	c.rw.Write(memdSpc)
	io.WriteString(c.rw, key)
	c.rw.Write(memdSep)
	if err := c.rw.Flush(); err != nil {
		c.close()
		return 0, err
	}

	id, err := readValue(c.rw.Reader)
	if err != nil {
		c.close()
		return 0, err
	}
	end, _, err := c.rw.ReadLine()
	if err != nil {
		c.close()
		return 0, err
	}
	if !bytes.Equal(end, memdEnd) {
		c.close()
		return 0, errors.New("unexpected response. not END")
	}
	return id, nil
}

func (c *memcacheClient) GetMulti(ctx context.Context, keys []string) ([]uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		if err := c.connect(ctx); err != nil {
			return nil, err
		}
	}
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	c.rw.Write(memdGets)
	for _, key := range keys {
		c.rw.Write(memdSpc)
		io.WriteString(c.rw, key)
	}
	c.rw.Write(memdSep)
	if err := c.rw.Flush(); err != nil {
		c.close()
		return nil, err
	}

	ids := make([]uint64, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		id, err := readValue(c.rw.Reader)
		if err != nil {
			c.close()
			return nil, err
		}
		ids = append(ids, id)
	}
	end, _, err := c.rw.ReadLine()
	if err != nil {
		c.close()
		return nil, err
	}
	if !bytes.Equal(end, memdEnd) {
		c.close()
		return nil, errors.New("unexpected response. not END")
	}
	return ids, nil
}

func readValue(r *bufio.Reader) (uint64, error) {
	line, _, err := r.ReadLine()
	if err != nil {
		return 0, err
	}
	if len(line) == 0 {
		return 0, errors.New("unexpected response")
	}
	fields := bytes.Fields(line)
	if len(fields) != 4 || !bytes.Equal(fields[0], memdValue) {
		return 0, errors.New("unexpected response. not VALUE")
	}
	value, _, err := r.ReadLine()
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseUint(string(value), 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}
