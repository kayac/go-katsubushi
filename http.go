package katsubushi

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
)

const (
	MaxHTTPBulkSize = 1000
)

func (app *App) RunHTTPServer(ctx context.Context, cfg *Config) error {
	mux := http.NewServeMux()
	mux.HandleFunc(fmt.Sprintf("/%sid", cfg.HTTPPathPrefix), app.HTTPGetSingleID)
	mux.HandleFunc(fmt.Sprintf("/%sids", cfg.HTTPPathPrefix), app.HTTPGetMultiID)
	mux.HandleFunc(fmt.Sprintf("/%sstats", cfg.HTTPPathPrefix), app.HTTPGetStats)
	log.Infof("Listening HTTP server at :%d", cfg.HTTPPort)
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler: mux,
		ConnState: func(conn net.Conn, state http.ConnState) {
			switch state {
			case http.StateNew:
				log.Debugf("Connected HTTP from %s", conn.RemoteAddr())
				atomic.AddInt64(&app.totalConnections, 1)
				atomic.AddInt64(&app.currConnections, 1)
			case http.StateClosed:
				log.Debugf("Closed HTTP %s", conn.RemoteAddr())
				atomic.AddInt64(&app.currConnections, -1)
			}
		},
	}
	// shutdown
	go func() {
		<-ctx.Done()
		log.Infof("Shutting down HTTP server")
		s.Shutdown(ctx)
	}()
	return s.ListenAndServe()
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
