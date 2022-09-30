package katsubushi_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/kayac/go-katsubushi"
)

var httpApp *katsubushi.App

func init() {
	var err error
	httpApp, err = katsubushi.NewApp(katsubushi.Option{WorkerID: 80})
	if err != nil {
		panic(err)
	}
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
