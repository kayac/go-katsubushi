package katsubushi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/kayac/go-katsubushi"
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
	for i := 0; i < 10; i++ {
		id, err := client.Fetch()
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
	for i := 0; i < 10; i++ {
		ids, err := client.FetchMulti(10)
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

func BenchmarkHTTPClientFetch(b *testing.B) {
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		u := fmt.Sprintf("http://localhost:%d", httpPort)
		c, _ := katsubushi.NewHTTPClient([]string{u}, "")
		for pb.Next() {
			id, err := c.Fetch()
			if err != nil {
				b.Fatal(err)
			}
			if id == 0 {
				b.Error("could not fetch id > 0")
			}
		}
	})
}
