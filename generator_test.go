package katsubushi

import (
	"testing"
	"time"
)

var nextWorkerID uint32

func getNextWorkerID() uint32 {
	nextWorkerID++
	return nextWorkerID
}

func TestInvalidWorkerID(t *testing.T) {
	// workerIDMask = 10bits = 0~1023
	if _, err := NewGenerator(1023); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if _, err := NewGenerator(1024); err != ErrInvalidWorkerID {
		t.Errorf("invalid error for overranged workerID: %s", err)
	}
}

func TestUniqueWorkerID(t *testing.T) {
	mayBeDup := getNextWorkerID()

	_, err := NewGenerator(mayBeDup)
	if err != nil {
		t.Fatalf("failed to create first generator: %s", err)
	}

	_, err = NewGenerator(getNextWorkerID())
	if err != nil {
		t.Fatalf("failed to create second generator: %s", err)
	}

	g, _ := NewGenerator(mayBeDup) // duplicate!!
	if g != nil {
		t.Fatalf("worker ID must be unique")
	}
}

func TestGenerateAnID(t *testing.T) {
	workerID := getNextWorkerID()

	g, err := NewGenerator(workerID)
	if err != nil {
		t.Fatalf("failed to create new generator: %s", err)
	}

	var id uint64
	now := time.Now()

	t.Log("generate")
	{
		ident, err := g.NextID()
		if err != nil {
			t.Fatalf("failed to generate id: %s", err)
		}

		if id < 0 {
			t.Error("invalid id")
		}

		id = ident
	}

	t.Logf("id = %d", id)

	t.Log("restore timestamp")
	{
		ts := (id & 0x7FFFFFFFFFC00000 >> (WorkerIDBits + SequenceBits)) + TimestampSince
		nowMsec := uint64(now.UnixNano()) / uint64(time.Millisecond)

		// To avoid failure would cause by timestamp on execution.
		if nowMsec != ts && ts != nowMsec+1 {
			t.Errorf("failed to restore timestamp: %d", ts)
		}
	}

	t.Log("restore worker ID")
	{
		wid := uint32(id & 0x3FF000 >> SequenceBits)
		if wid != workerID {
			t.Errorf("failed to restore worker ID: %d", wid)
		}
	}
}

func TestGenerateSomeIDs(t *testing.T) {
	g, _ := NewGenerator(getNextWorkerID())
	ids := []uint64{}

	for i := 0; i < 1000; i++ {
		id, err := g.NextID()
		if err != nil {
			t.Fatalf("failed to generate id: %s", err)
		}

		for _, otherID := range ids {
			if otherID == id {
				t.Fatal("id duplicated!!")
			}
		}

		if l := len(ids); 0 < l && id < ids[l-1] {
			t.Fatal("generated smaller id!!")
		}

		ids = append(ids, id)
	}

	t.Logf("%d ids are tested", len(ids))
}

func TestClockRollback(t *testing.T) {
	g, _ := NewGenerator(getNextWorkerID())
	_, err := g.NextID()
	if err != nil {
		t.Fatalf("failed to generate id: %s", err)
	}

	// サーバーの時計が巻き戻った想定
	nowFunc = func() time.Time {
		return time.Now().Add(-10 * time.Minute)
	}

	_, err = g.NextID()
	if err == nil {
		t.Fatalf("when server clock rollback, generater must return error")
	}

	t.Log(err)
}

func BenchmarkGenerateID(b *testing.B) {
	g, _ := NewGenerator(getNextWorkerID())
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		g.NextID()
	}
}
