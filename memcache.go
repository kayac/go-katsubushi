package katsubushi

import (
	"bufio"
	"bytes"
	"errors"
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
}

func newMemcacheClient(addr string) *memcacheClient {
	return &memcacheClient{
		addr:    addr,
		timeout: memcacheDefaultTimeout,
	}
}

func (c *memcacheClient) connect() error {
	var err error
	c.conn, err = net.DialTimeout("tcp", c.addr, c.timeout)
	return err
}

func (c *memcacheClient) close() error {
	defer func() { c.conn = nil }()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *memcacheClient) Get(key string) (uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		if err := c.connect(); err != nil {
			return 0, err
		}
	}
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	b := make([]byte, 0, len(memdGets)+len(memdSpc)+len(key)+len(memdSep))
	b = append(b, memdGets...)
	b = append(b, memdSpc...)
	b = append(b, []byte(key)...)
	b = append(b, memdSep...)
	_, err := c.conn.Write(b)
	if err != nil {
		c.close()
		return 0, err
	}

	r := bufio.NewReader(c.conn)
	id, err := readValue(r)
	if err != nil {
		c.close()
		return 0, err
	}
	end, _, err := r.ReadLine()
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

func (c *memcacheClient) GetMulti(keys []string) ([]uint64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		if err := c.connect(); err != nil {
			return nil, err
		}
	}
	c.conn.SetDeadline(time.Now().Add(c.timeout))
	b := make([]byte, 0, len(memdGets)+len(memdSpc)+(len(keys[0])+len(memdSpc))*len(keys)+len(memdSep))
	b = append(b, memdGets...)
	for _, key := range keys {
		b = append(b, memdSpc...)
		b = append(b, []byte(key)...)
	}
	b = append(b, memdSep...)
	_, err := c.conn.Write(b)
	if err != nil {
		c.close()
		return nil, err
	}

	ids := make([]uint64, 0, len(keys))
	r := bufio.NewReader(c.conn)
	for i := 0; i < len(keys); i++ {
		id, err := readValue(r)
		if err != nil {
			c.close()
			return nil, err
		}
		ids = append(ids, id)
	}
	end, _, err := r.ReadLine()
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
