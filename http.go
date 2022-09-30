package katsubushi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

const (
	MaxHTTPBulkSize = 1000
)

func (app *App) RunHTTPServer(ctx context.Context, wg *sync.WaitGroup, port int) error {
	defer wg.Done()

	mux := http.NewServeMux()
	mux.HandleFunc("/id", app.httpGetSingleID)
	mux.HandleFunc("/ids", app.httpGetMultiID)
	log.Infof("Listening HTTP server at :%d", port)
	s := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	// shutdown
	go func() {
		<-ctx.Done()
		log.Infof("Shutting down HTTP server")
		s.Shutdown(ctx)
	}()
	return s.ListenAndServe()
}

func (app *App) httpGetSingleID(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	id, err := app.NextID()
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if strings.Contains(req.Header.Get("Accept"), "application/json") {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"%d"}`, id)
	} else {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "%d", id)
	}
}

func (app *App) httpGetMultiID(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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
