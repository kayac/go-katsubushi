package katsubushi

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var nowFunc = time.Now

// TimestampSince is offset from epoch time origin as millseconds.
// It indicates 2015-01-01 00:00:00 UTC
const TimestampSince = uint64(1420070400 * 1000)

// for bitshift
const (
	WorkerIDBits = 10
	SequenceBits = 12
)

const (
	workerIDMask = -1 ^ (-1 << WorkerIDBits)
	sequenseMask = -1 ^ (-1 << SequenceBits)
)

var workerIDPool = []uint32{}
var newGeneratorLock sync.Mutex

// errors
var (
	ErrInvalidWorkerID    = errors.New("invalid worker id")
	ErrDuplicatedWorkerID = errors.New("duplicated worker")
)

func checkWorkerID(id uint32) error {
	if workerIDMask < id {
		return ErrInvalidWorkerID
	}

	for _, otherID := range workerIDPool {
		if id == otherID {
			return ErrDuplicatedWorkerID
		}
	}

	return nil
}

// Generator is generater for unique ID.
type Generator struct {
	WorkerID      uint32
	lastTimestamp uint64
	sequence      uint32
	lock          sync.Mutex
}

// NewGenerator returns new generator.
func NewGenerator(workerID uint32) (*Generator, error) {
	// To keep worker ID be unique.
	newGeneratorLock.Lock()
	defer newGeneratorLock.Unlock()

	if err := checkWorkerID(workerID); err != nil {
		return nil, err
	}

	// save as already used
	workerIDPool = append(workerIDPool, workerID)

	g := Generator{
		WorkerID: workerID,
	}

	return &g, nil
}

// NextID generate new ID.
func (g *Generator) NextID() (uint64, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	ts := g.timestamp()

	// for rewind of server clock
	if ts < g.lastTimestamp {
		return 0, fmt.Errorf("going to past!! your ntp service seems to be wrong")
	}

	if ts == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & sequenseMask

		if g.sequence == 0 {
			ts = g.waitUntilNextTick(ts)
		}
	} else {
		g.sequence = 0
	}

	g.lastTimestamp = ts

	return g.currentID(), nil
}

func (g *Generator) currentID() uint64 {
	return (g.lastTimestamp << (WorkerIDBits + SequenceBits)) | (uint64(g.WorkerID) << SequenceBits) | (uint64(g.sequence))
}

func (g *Generator) timestamp() uint64 {
	return uint64(nowFunc().UnixNano())/uint64(time.Millisecond) - TimestampSince
}

func (g *Generator) waitUntilNextTick(ts uint64) uint64 {
	next := g.timestamp()

	for next <= ts {
		next = g.timestamp()
		time.Sleep(50 * time.Nanosecond)
	}

	return next
}
