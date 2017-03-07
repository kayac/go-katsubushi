package katsubushi_test

import (
	"testing"
	"time"

	"github.com/kayac/go-katsubushi"
)

func TestConvertFixed(t *testing.T) {
	t1 := time.Unix(1465276650, 0)
	id := katsubushi.ToID(t1)
	if id != 189608755200000000 {
		t.Error("unexpected id", id)
	}

	t2 := katsubushi.ToTime(id)
	if !t1.Equal(t2) {
		t.Error("roudtrip failed")
	}
}

func TestConvertFixedSub(t *testing.T) {
	t1 := time.Unix(1465276650, 777000000)
	id := katsubushi.ToID(t1)
	if id != 189608758458974208 {
		t.Error("unexpected id", id)
	}

	t2 := katsubushi.ToTime(id)
	if !t1.Equal(t2) {
		t.Error("roudtrip failed")
	}
}

func TestConvertNow(t *testing.T) {
	t1 := time.Now()
	id := katsubushi.ToID(t1)
	t2 := katsubushi.ToTime(id)
	f := "2006-01-02T15:04:05.000"
	if t1.Format(f) != t2.Format(f) {
		t.Error("roudtrip failed", t1, t2)
	}
}
