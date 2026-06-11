package katsubushi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kayac/go-katsubushi/v2"
)

var httpApp *katsubushi.App
var httpPort int

func init() {
	var err error
	httpApp, err = katsubushi.New(80)
	if err != nil {
		panic(err)
	}
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(err)
	}
	httpPort = listener.Addr().(*net.TCPAddr).Port
	go httpApp.RunHTTPServer(context.Background(), &katsubushi.Config{HTTPListener: listener})
	time.Sleep(3 * time.Second)
}

func TestHTTPSingle(t *testing.T) {
	req := httptest.NewRequest("GET", "/id", nil)
	w := httptest.NewRecorder()

	httpApp.HTTPGetSingleID(w, req)
	if w.Code != 200 {
		t.Errorf("status code should be 200 but %d", w.Code)
	}
	b := new(bytes.Buffer)
	if _, err := io.Copy(b, w.Body); err != nil {
		t.Errorf("failed to read body: %v", err)
	}
	if id, err := strconv.ParseUint(b.String(), 10, 64); err != nil {
		t.Errorf("body should be a number uint64: %v", err)
	} else {
		t.Logf("HTTP fetched single ID: %d", id)
	}
}

func TestHTTPSingleJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/id", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	httpApp.HTTPGetSingleID(w, req)
	if w.Code != 200 {
		t.Errorf("status code should be 200 but %d", w.Code)
	}
	v := struct {
		ID string `json:"id"`
	}{}
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Errorf("failed to decode body: %v", err)
	}
	if id, err := strconv.ParseUint(v.ID, 10, 64); err != nil {
		t.Errorf("body should be a number uint64: %v", err)
	} else {
		t.Logf("HTTP fetched single ID as JSON: %d", id)
	}
}

func TestHTTPMulti(t *testing.T) {
	req := httptest.NewRequest("GET", "/ids?n=10", nil)
	w := httptest.NewRecorder()

	httpApp.HTTPGetMultiID(w, req)
	if w.Code != 200 {
		t.Errorf("status code should be 200 but %d", w.Code)
	}
	b := new(bytes.Buffer)
	if _, err := io.Copy(b, w.Body); err != nil {
		t.Errorf("failed to read body: %v", err)
	}
	bs := bytes.Split(b.Bytes(), []byte("\n"))
	if len(bs) != 10 {
		t.Errorf("body should contain 10 lines but %d", len(bs))
	}
	for _, b := range bs {
		if id, err := strconv.ParseUint(string(b), 10, 64); err != nil {
			t.Errorf("body should be a number uint64: %v", err)
		} else {
			t.Logf("HTTP fetched ID: %d", id)
		}
	}
}

func TestHTTPMultiJSON(t *testing.T) {
	req := httptest.NewRequest("GET", "/ids?n=10", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()

	httpApp.HTTPGetMultiID(w, req)
	if w.Code != 200 {
		t.Errorf("status code should be 200 but %d", w.Code)
	}
	v := struct {
		IDs []string `json:"ids"`
	}{}
	if err := json.NewDecoder(w.Body).Decode(&v); err != nil {
		t.Errorf("failed to decode body: %v", err)
	}
	if len(v.IDs) != 10 {
		t.Errorf("body should contain 10 lines but %d", len(v.IDs))
	}
	for _, id := range v.IDs {
		if i, err := strconv.ParseUint(id, 10, 64); err != nil {
			t.Errorf("body should be a number uint64: %v", err)
		} else {
			t.Logf("HTTP fetched single ID as JSON: %d", i)
		}
	}
}

func testHTTPStats(t *testing.T) *katsubushi.MemdStats {
	req := httptest.NewRequest("GET", "/stats", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	httpApp.HTTPGetStats(w, req)
	if w.Code != 200 {
		t.Errorf("status code should be 200 but %d", w.Code)
	}
	var s katsubushi.MemdStats
	if err := json.NewDecoder(w.Body).Decode(&s); err != nil {
		t.Errorf("failed to read body: %v", err)
	}
	t.Logf("%#v", s)
	return &s
}

func TestHTTPStats(t *testing.T) {
	TestHTTPSingle(t)
	s1 := testHTTPStats(t)

	TestHTTPSingle(t)
	s2 := testHTTPStats(t)
	if s2.CmdGet != s1.CmdGet+1 {
		t.Errorf("cmd_get should be incremented by 1 but %d", s2.CmdGet-s1.CmdGet)
	}
	if s2.GetHits != s1.GetHits+1 {
		t.Errorf("get_hits should be incremented by 1 but %d", s2.GetHits-s1.GetHits)
	}

	TestHTTPMulti(t)
	s3 := testHTTPStats(t)
	if s3.CmdGet != s2.CmdGet+1 {
		t.Errorf("cmd_get should be incremented by 10 but %d", s3.CmdGet-s2.CmdGet)
	}
	if s3.GetHits != s2.GetHits+10 {
		t.Errorf("get_hits should be incremented by 10 but %d", s3.GetHits-s2.GetHits)
	}
}

func TestHTTPSingleCS(t *testing.T) {
	u := fmt.Sprintf("http://localhost:%d", httpPort)
	client, err := katsubushi.NewHTTPClient([]string{u}, "")
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		id, err := client.Fetch(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if id == 0 {
			t.Fatal("id should not be 0")
		}
		t.Logf("HTTP fetched single ID: %d", id)
	}
}

func TestHTTPMultiCS(t *testing.T) {
	u := fmt.Sprintf("http://localhost:%d", httpPort)
	client, err := katsubushi.NewHTTPClient([]string{u}, "")
	if err != nil {
		t.Fatal(err)
	}
	for range 10 {
		ids, err := client.FetchMulti(context.Background(), 10)
		if err != nil {
			t.Fatal(err)
		}
		if len(ids) != 10 {
			t.Fatalf("ids should contain 10 elements %v", ids)
		}
		for _, id := range ids {
			if id == 0 {
				t.Fatal("id should not be 0")
			}
		}
		t.Logf("HTTP fetched IDs: %v", ids)
	}
}

func TestHTTPMultiInvalidN(t *testing.T) {
	for _, n := range []string{"0", "-1", "1001", "foo"} {
		req := httptest.NewRequest("GET", "/ids?n="+n, nil)
		w := httptest.NewRecorder()
		httpApp.HTTPGetMultiID(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("status code for n=%s should be 400 but %d", n, w.Code)
		}
	}
}

func TestHTTPClientPathPrefix(t *testing.T) {
	app, err := katsubushi.New(81)
	if err != nil {
		t.Fatal(err)
	}
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	go app.RunHTTPServer(t.Context(), &katsubushi.Config{HTTPListener: listener, HTTPPathPrefix: "v1/"})
	select {
	case <-app.Ready():
	case <-time.After(5 * time.Second):
		t.Fatal("the app must become ready by RunHTTPServer")
	}

	u := fmt.Sprintf("http://%s", listener.Addr())
	client, err := katsubushi.NewHTTPClient([]string{u}, "v1/")
	if err != nil {
		t.Fatal(err)
	}
	id, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("failed to fetch id from the prefixed path: %v", err)
	}
	if id == 0 {
		t.Fatal("id should not be 0")
	}
	ids, err := client.FetchMulti(context.Background(), 3)
	if err != nil {
		t.Fatalf("failed to fetch ids from the prefixed path: %v", err)
	}
	if len(ids) != 3 {
		t.Fatalf("ids should contain 3 elements: %v", ids)
	}
}

func TestHTTPClientURLIsolation(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/id", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" {
			t.Errorf("/id should be requested without a query: %s", r.URL.RawQuery)
		}
		fmt.Fprint(w, "42")
	})
	mux.HandleFunc("/ids", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "1\n2\n3")
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()

	client, err := katsubushi.NewHTTPClient([]string{ts.URL}, "")
	if err != nil {
		t.Fatal(err)
	}

	// FetchMulti sets a query parameter. It must not leak into the following Fetch.
	if _, err := client.FetchMulti(context.Background(), 3); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Fetch(context.Background()); err != nil {
		t.Fatal(err)
	}

	// Concurrent calls must not race on the shared URLs (verified by -race).
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, err := client.Fetch(context.Background()); err != nil {
				t.Error(err)
			}
			if _, err := client.FetchMulti(context.Background(), 3); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}

func TestHTTPFailover(t *testing.T) {
	// The first server always fails; the client must fail over to the healthy one.
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := fmt.Sprintf("http://localhost:%d", httpPort)

	client, err := katsubushi.NewHTTPClient([]string{bad.URL, good}, "")
	if err != nil {
		t.Fatal(err)
	}

	id, err := client.Fetch(context.Background())
	if err != nil {
		t.Fatalf("Fetch should fail over to the healthy server: %v", err)
	}
	if id == 0 {
		t.Fatal("id should not be 0")
	}

	ids, err := client.FetchMulti(context.Background(), 10)
	if err != nil {
		t.Fatalf("FetchMulti should fail over to the healthy server: %v", err)
	}
	if len(ids) != 10 {
		t.Fatalf("ids should contain 10 elements: %v", ids)
	}
}

func TestHTTPAllServersFail(t *testing.T) {
	// When every server fails, an error must be returned (not a zero value).
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	bad2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad2.Close()

	client, err := katsubushi.NewHTTPClient([]string{bad.URL, bad2.URL}, "")
	if err != nil {
		t.Fatal(err)
	}

	if id, err := client.Fetch(context.Background()); err == nil {
		t.Fatalf("Fetch should return an error when all servers fail, got id=%d", id)
	} else if !strings.Contains(err.Error(), bad.URL) || !strings.Contains(err.Error(), bad2.URL) {
		t.Errorf("error should contain failures of all servers: %v", err)
	} else if strings.Contains(err.Error(), "no servers available") {
		t.Errorf("error should not say no servers available when servers are configured: %v", err)
	}
	if ids, err := client.FetchMulti(context.Background(), 10); err == nil {
		t.Fatalf("FetchMulti should return an error when all servers fail, got ids=%v", ids)
	} else if !strings.Contains(err.Error(), bad.URL) || !strings.Contains(err.Error(), bad2.URL) {
		t.Errorf("error should contain failures of all servers: %v", err)
	}
}

func TestHTTPClientNoServers(t *testing.T) {
	client, err := katsubushi.NewHTTPClient([]string{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Fetch(context.Background()); err == nil || !strings.Contains(err.Error(), "no servers available") {
		t.Errorf("Fetch should fail with no servers available: %v", err)
	}
	if _, err := client.FetchMulti(context.Background(), 10); err == nil || !strings.Contains(err.Error(), "no servers available") {
		t.Errorf("FetchMulti should fail with no servers available: %v", err)
	}
}

func BenchmarkHTTPClientFetch(b *testing.B) {
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		u := fmt.Sprintf("http://localhost:%d", httpPort)
		c, _ := katsubushi.NewHTTPClient([]string{u}, "")
		for pb.Next() {
			id, err := c.Fetch(context.Background())
			if err != nil {
				b.Fatal(err)
			}
			if id == 0 {
				b.Error("could not fetch id > 0")
			}
		}
	})
}
