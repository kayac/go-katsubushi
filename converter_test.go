package katsubushi_test

import (
	"testing"
	"time"

	"github.com/kayac/go-katsubushi/v2"
)

func TestConvertFixed(t *testing.T) {
	t1 := time.Unix(1465276650, 0)
	id := katsubushi.ToID(t1)
	if id != 189608755200000000 {
		t.Error("unexpected id", id)
	}

	t2 := katsubushi.ToTime(id)
	if !t1.Equal(t2) {
		t.Error("roundtrip failed")
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
		t.Error("roundtrip failed")
	}
}

func TestConvertNow(t *testing.T) {
	t1 := time.Now().In(time.UTC)
	id := katsubushi.ToID(t1)
	t2 := katsubushi.ToTime(id)
	f := "2006-01-02T15:04:05.000"
	if t1.Format(f) != t2.Format(f) {
		t.Error("roundtrip failed", t1, t2)
	}
}

func TestDump(t *testing.T) {
	testCases := []struct {
		id  uint64
		ts  time.Time
		wid uint64
		seq uint64
	}{
		{
			354101311794212865,
			time.Date(2017, 9, 4, 3, 12, 11, 615000000, time.UTC),
			999,
			1,
		},
		{
			354103658909954052,
			time.Date(2017, 9, 4, 3, 21, 31, 211000000, time.UTC),
			999,
			4,
		},
	}
	for _, tc := range testCases {
		ts, wid, seq := katsubushi.Dump(tc.id)
		if ts != tc.ts {
			t.Errorf("%d timestamp is not expected. got %s expected %s", tc.id, ts, tc.ts)
		}
		if wid != tc.wid {
			t.Errorf("%d workerID is not expected. got %d expected %d", tc.id, wid, tc.wid)
		}
		if seq != tc.seq {
			t.Errorf("%d sequence is not expected. got %d expected %d", tc.id, seq, tc.seq)
		}
	}
}
