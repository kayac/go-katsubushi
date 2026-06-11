package katsubushi

import (
	"slices"
	"testing"
	"time"
)

const maxSafeInteger = uint64(9007199254740991) // Number.MAX_SAFE_INTEGER in JavaScript

func TestJSSafeMaxID(t *testing.T) {
	if uint64(JSSafeMaxID) != maxSafeInteger {
		t.Errorf("JSSafeMaxID %d must equal Number.MAX_SAFE_INTEGER %d", uint64(JSSafeMaxID), maxSafeInteger)
	}
}

func TestJSSafeInvalidWorkerID(t *testing.T) {
	// JSSafeWorkerIDBits = 6bits = 0~63
	if _, err := NewJSSafeGenerator(63); err != nil {
		t.Errorf("unexpected error: %s", err)
	}

	if _, err := NewJSSafeGenerator(64); err != ErrInvalidWorkerID {
		t.Errorf("invalid error for overranged workerID: %s", err)
	}
}

func TestJSSafeUniqueWorkerID(t *testing.T) {
	if _, err := NewJSSafeGenerator(10); err != nil {
		t.Fatalf("failed to create first generator: %s", err)
	}

	g, _ := NewJSSafeGenerator(10) // duplicate!!
	if g != nil {
		t.Fatalf("worker ID must be unique")
	}
}

func TestJSSafeWorkerIDPoolSeparatedByFormat(t *testing.T) {
	newGeneratorLock.Lock()
	workerIDPools[jsSafeSpec.name] = append(workerIDPools[jsSafeSpec.name], 59)
	newGeneratorLock.Unlock()
	t.Cleanup(func() {
		newGeneratorLock.Lock()
		defer newGeneratorLock.Unlock()
		workerIDPools[jsSafeSpec.name] = slices.DeleteFunc(workerIDPools[jsSafeSpec.name], func(id uint) bool {
			return id == 59
		})
	})

	if err := checkWorkerID(59, jsSafeSpec); err != ErrDuplicatedWorkerID {
		t.Errorf("worker ID 59 must be duplicated in the JS-safe format: %s", err)
	}
	// The same worker ID is still available in the default format.
	if err := checkWorkerID(59, defaultSpec); err != nil {
		t.Errorf("worker ID 59 must be available in the default format: %s", err)
	}
}

func TestJSSafeGenerateAnID(t *testing.T) {
	workerID := uint(1)

	g, err := NewJSSafeGenerator(workerID)
	if err != nil {
		t.Fatalf("failed to create new generator: %s", err)
	}

	now := time.Now()
	id, err := g.NextID()
	if err != nil {
		t.Fatalf("failed to generate id: %s", err)
	}

	t.Logf("id = %d", id)

	if id == 0 {
		t.Error("invalid id")
	}

	if id > maxSafeInteger {
		t.Errorf("id %d exceeds Number.MAX_SAFE_INTEGER %d", id, maxSafeInteger)
	}

	t.Log("restore timestamp")
	{
		ts := ToTimeJSSafe(id)
		if d := now.Sub(ts); d < 0 || d > 2*JSSafeTimestampUnit {
			t.Errorf("failed to restore timestamp: %s", ts)
		}
	}

	t.Log("restore worker ID")
	{
		_, wid, _ := DumpJSSafe(id)
		if uint(wid) != workerID {
			t.Errorf("failed to restore worker ID: %d", wid)
		}
	}
}

func TestJSSafeGenerateSomeIDs(t *testing.T) {
	g, err := NewJSSafeGenerator(2)
	if err != nil {
		t.Fatalf("failed to create new generator: %s", err)
	}
	ids := make(map[uint64]struct{}, 1000)

	var lastID uint64
	// 1000 IDs require sequence overflows (8bits = 256 IDs per 10ms tick)
	for range 1000 {
		id, err := g.NextID()
		if err != nil {
			t.Fatalf("failed to generate id: %s", err)
		}

		if _, exists := ids[id]; exists {
			t.Fatal("id duplicated!!")
		}

		if id <= lastID {
			t.Fatal("generated smaller id!!")
		}

		if id > maxSafeInteger {
			t.Fatalf("id %d exceeds Number.MAX_SAFE_INTEGER %d", id, maxSafeInteger)
		}

		ids[id] = struct{}{}
		lastID = id
	}

	t.Logf("%d ids are tested", len(ids))
}

func TestJSSafeAppIDFormat(t *testing.T) {
	app, err := NewJSSafe(3)
	if err != nil {
		t.Fatalf("failed to create app: %s", err)
	}
	if app.idFormat != "js-safe" {
		t.Errorf("unexpected idFormat: %s", app.idFormat)
	}

	app, err = New(getNextWorkerID())
	if err != nil {
		t.Fatalf("failed to create app: %s", err)
	}
	if app.idFormat != "default" {
		t.Errorf("unexpected idFormat: %s", app.idFormat)
	}
}

func TestJSSafeIDLifetime(t *testing.T) {
	// The JS-safe format must fit within MAX_SAFE_INTEGER
	// for at least 170 years from the epoch (2^39 * 10ms is about 174 years).
	maxFields := uint64(1<<(JSSafeWorkerIDBits+JSSafeSequenceBits)) - 1
	id := ToIDJSSafe(Epoch.AddDate(170, 0, 0)) | maxFields
	if id > maxSafeInteger {
		t.Errorf("id %d at 170 years after the epoch exceeds Number.MAX_SAFE_INTEGER", id)
	}
}
