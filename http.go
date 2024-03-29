package katsubushi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
)

const (
	MaxHTTPBulkSize = 1000
)

func (app *App) RunHTTPServer(ctx context.Context, cfg *Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/%sid", cfg.HTTPPathPrefix), app.HTTPGetSingleID)
	mux.HandleFunc(fmt.Sprintf("/%sids", cfg.HTTPPathPrefix), app.HTTPGetMultiID)
	mux.HandleFunc(fmt.Sprintf("/%sstats", cfg.HTTPPathPrefix), app.HTTPGetStats)
	s := &http.Server{
		Handler: mux,
	}
	// shutdown
	go func() {
		<-ctx.Done()
		log.Infof("Shutting down HTTP server")
		s.Shutdown(ctx)
	}()

	listener := cfg.HTTPListener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.HTTPPort))
		if err != nil {
			return errors.Wrap(err, "failed to listen")
		}
	}
	listener = app.wrapListener(listener)
	log.Infof("Listening HTTP server at %s", listener.Addr())
	return s.Serve(listener)
}

func (app *App) HTTPGetSingleID(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	atomic.AddInt64(&app.cmdGet, 1)
	id, err := app.NextID()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	log.Debugf("Generated ID: %d", id)
	if strings.Contains(req.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"%d"}`, id)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%d", id)
	}
}

func (app *App) HTTPGetMultiID(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	atomic.AddInt64(&app.cmdGet, 1)
	var n int64
	if ns := req.FormValue("n"); ns == "" {
		n = 1
	} else {
		var err error
		n, err = strconv.ParseInt(ns, 10, 64)
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if n > MaxHTTPBulkSize {
		msg := fmt.Sprintf("too many IDs requested: %d, n should be smaller than %d", n, MaxHTTPBulkSize)
		log.Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}
	ids := make([]string, 0, n)
	for i := int64(0); i < n; i++ {
		id, err := app.NextID()
		if err != nil {
			log.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ids = append(ids, strconv.FormatUint(id, 10))
	}
	log.Debugf("Generated IDs: %v", ids)
	if strings.Contains(req.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			IDs []string `json:"ids"`
		}{ids})
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, strings.Join(ids, "\n"))
	}
}

func (app *App) HTTPGetStats(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	s := app.GetStats()
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

type HTTPClient struct {
	client     *http.Client
	urls       []*url.URL
	pathPrefix string
	pool       *sync.Pool
}

// NewHTTPClient creates HTTPClient
func NewHTTPClient(urls []string, pathPrefix string) (*HTTPClient, error) {
	c := &HTTPClient{
		client: &http.Client{
			Timeout: DefaultClientTimeout,
		},
		pool: &sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		},
	}
	for _, _u := range urls {
		u, err := url.Parse(_u)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse URL: %s", _u)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, errors.Errorf("invalid URL scheme: %s", u.Scheme)
		}
		c.urls = append(c.urls, u)
	}
	return c, nil
}

// SetTimeout sets timeout to katsubushi servers
func (c *HTTPClient) SetTimeout(t time.Duration) {
	c.client.Timeout = t
}

// Fetch fetches id from katsubushi via HTTP
func (c *HTTPClient) Fetch(ctx context.Context) (uint64, error) {
	errs := errors.New("no servers available")
	for _, u := range c.urls {
		id, err := func(u *url.URL) (uint64, error) {
			u.Path = fmt.Sprintf("/%sid", c.pathPrefix)
			req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
			resp, err := c.client.Do(req)
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return 0, errors.Errorf("unexpected status code: %d", resp.StatusCode)
			}
			b := c.pool.Get().(*bytes.Buffer)
			defer func() {
				b.Reset()
				c.pool.Put(b)
			}()
			if _, err := io.Copy(b, resp.Body); err != nil {
				return 0, err
			}
			if id, err := strconv.ParseUint(b.String(), 10, 64); err != nil {
				return 0, err
			} else {
				return id, nil
			}
		}(u)
		if err != nil {
			errs = errors.Wrapf(err, "failed to fetch id from %s", u)
		}
		return id, nil
	}
	return 0, errs
}

// FetchMulti fetches multiple ids from katsubushi via HTTP
func (c *HTTPClient) FetchMulti(ctx context.Context, n int) ([]uint64, error) {
	errs := errors.New("no servers available")
	ids := make([]uint64, 0, n)
	for _, u := range c.urls {
		ids, err := func(u *url.URL) ([]uint64, error) {
			u.Path = fmt.Sprintf("/%sids", c.pathPrefix)
			u.RawQuery = fmt.Sprintf("n=%d", n)
			req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
			resp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
			}

			b := c.pool.Get().(*bytes.Buffer)
			defer func() {
				b.Reset()
				c.pool.Put(b)
			}()
			if _, err := io.Copy(b, resp.Body); err != nil {
				return nil, err

			}
			bs := bytes.Split(b.Bytes(), []byte("\n"))
			if len(bs) != n {
				return nil, err
			}
			for _, b := range bs {
				if id, err := strconv.ParseUint(string(b), 10, 64); err != nil {
					return nil, err
				} else {
					ids = append(ids, id)
				}
			}
			return ids, nil
		}(u)
		if err != nil {
			errs = errors.Wrapf(errs, "failed to fetch ids from %s", u)
		}
		return ids, nil
	}
	return nil, errs
}
