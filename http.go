package katsubushi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
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
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		slog.Info("Shutting down HTTP server")
		// ctx is already canceled here, so use a new context to wait for in-flight requests.
		sctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()
		if err := s.Shutdown(sctx); err != nil {
			slog.Warn("Failed to shutdown HTTP server gracefully", "error", err)
		}
	}()

	listener := cfg.HTTPListener
	if listener == nil {
		var err error
		listener, err = net.Listen("tcp", fmt.Sprintf(":%d", cfg.HTTPPort))
		if err != nil {
			return fmt.Errorf("failed to listen: %w", err)
		}
	}
	listener = app.wrapListener(listener)
	slog.Info("Listening HTTP server", "addr", listener.Addr().String())
	err := s.Serve(listener)
	select {
	case <-ctx.Done():
		// Serve returns as soon as the shutdown begins, so wait for it to complete.
		<-shutdownDone
	default:
	}
	return err
}

func (app *App) HTTPGetSingleID(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	app.cmdGet.Add(1)
	slog.Debug("HTTP GetSingleID request", "remote", req.RemoteAddr)
	id, err := app.NextID()
	if err != nil {
		slog.Error("Failed to generate ID", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	slog.Debug("HTTP Generated ID", "id", id)
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
	app.cmdGet.Add(1)
	slog.Debug("HTTP GetMultiID request", "remote", req.RemoteAddr)
	var n int64
	if ns := req.FormValue("n"); ns == "" {
		n = 1
	} else {
		var err error
		n, err = strconv.ParseInt(ns, 10, 64)
		if err != nil {
			slog.Error("Failed to parse n parameter", "error", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	if n < 1 || n > MaxHTTPBulkSize {
		msg := fmt.Sprintf("invalid n: %d, n should be between 1 and %d", n, MaxHTTPBulkSize)
		slog.Error(msg)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(msg))
		return
	}
	ids := make([]string, 0, n)
	for i := int64(0); i < n; i++ {
		id, err := app.NextID()
		if err != nil {
			slog.Error("Failed to generate ID", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		ids = append(ids, strconv.FormatUint(id, 10))
	}
	slog.Debug("HTTP Generated IDs", "ids", ids, "count", len(ids))
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
	slog.Debug("HTTP GetStats request", "remote", req.RemoteAddr)
	s := app.GetStats()
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		slog.Error("Failed to encode stats", "error", err)
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
		pathPrefix: pathPrefix,
		pool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
	for _, _u := range urls {
		u, err := url.Parse(_u)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %s: %w", _u, err)
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("invalid URL scheme: %s", u.Scheme)
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
	errs := fmt.Errorf("no servers available")
	for _, u := range c.urls {
		// copy the URL to avoid mutating the shared one
		id, err := func(u url.URL) (uint64, error) {
			u.Path = fmt.Sprintf("/%sid", c.pathPrefix)
			u.RawQuery = ""
			req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
			resp, err := c.client.Do(req)
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return 0, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
		}(*u)
		if err != nil {
			errs = fmt.Errorf("failed to fetch id from %s: %w", u, err)
			continue
		}
		return id, nil
	}
	return 0, errs
}

// FetchMulti fetches multiple ids from katsubushi via HTTP
func (c *HTTPClient) FetchMulti(ctx context.Context, n int) ([]uint64, error) {
	errs := fmt.Errorf("no servers available")
	for _, u := range c.urls {
		// copy the URL to avoid mutating the shared one
		ids, err := func(u url.URL) ([]uint64, error) {
			u.Path = fmt.Sprintf("/%sids", c.pathPrefix)
			u.RawQuery = fmt.Sprintf("n=%d", n)
			req, _ := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
			resp, err := c.client.Do(req)
			if err != nil {
				return nil, err
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
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
				return nil, fmt.Errorf("unexpected number of ids: got %d, want %d", len(bs), n)
			}
			ids := make([]uint64, 0, n)
			for _, b := range bs {
				id, err := strconv.ParseUint(string(b), 10, 64)
				if err != nil {
					return nil, err
				}
				ids = append(ids, id)
			}
			return ids, nil
		}(*u)
		if err != nil {
			errs = fmt.Errorf("failed to fetch ids from %s: %w", u, err)
			continue
		}
		return ids, nil
	}
	return nil, errs
}
