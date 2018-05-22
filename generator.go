package katsubushi

import (
	"errors"
	"sync"
	"time"
)

var nowFunc = time.Now

// Epoch is katsubushi epoch time (2015-01-01 00:00:00 UTC)
// Generated ID includes elapsed time from Epoch.
var Epoch = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

// for bitshift
const (
	WorkerIDBits = 10
	SequenceBits = 12
	workerIDMask = -1 ^ (-1 << WorkerIDBits)
	sequenseMask = -1 ^ (-1 << SequenceBits)
)

var workerIDPool = []uint{}
var newGeneratorLock sync.Mutex

// errors
var (
	ErrInvalidWorkerID    = errors.New("invalid worker id")
	ErrDuplicatedWorkerID = errors.New("duplicated worker")
)

func checkWorkerID(id uint) error {
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

// Generator is an interface to generate unique ID.
type Generator interface {
	NextID() (uint64, error)
	WorkerID() uint
}

type generator struct {
	workerID      uint
	lastTimestamp uint64
	sequence      uint
	lock          sync.Mutex
	startedAt     time.Time
	offset        time.Duration
}

// NewGenerator returns new generator.
func NewGenerator(workerID uint) (Generator, error) {
	// To keep worker ID be unique.
	newGeneratorLock.Lock()
	defer newGeneratorLock.Unlock()

	if err := checkWorkerID(workerID); err != nil {
		return nil, err
	}

	// save as already used
	workerIDPool = append(workerIDPool, workerID)

	now := nowFunc()
	return &generator{
		workerID:  workerID,
		startedAt: now,
		offset:    now.Sub(Epoch),
	}, nil
}

func (g *generator) WorkerID() uint {
	return g.workerID
}

// NextID generate new ID.
func (g *generator) NextID() (uint64, error) {
	g.lock.Lock()
	defer g.lock.Unlock()

	ts := g.timestamp()

	// for rewind of server clock
	if ts < g.lastTimestamp {
		return 0, errors.New("system clock was rollbacked")
	}

	if ts == g.lastTimestamp {
		g.sequence = (g.sequence + 1) & sequenseMask
		if g.sequence == 0 {
			// overflow
			ts = g.waitUntilNextTick(ts)
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = ts

	return (g.lastTimestamp << (WorkerIDBits + SequenceBits)) | (uint64(g.workerID) << SequenceBits) | (uint64(g.sequence)), nil
}

func (g *generator) timestamp() uint64 {
	d := nowFunc().Sub(g.startedAt) + g.offset
	return uint64(d.Nanoseconds()) / uint64(time.Millisecond)
}

func (g *generator) waitUntilNextTick(ts uint64) uint64 {
	next := g.timestamp()

	for next <= ts {
		next = g.timestamp()
		time.Sleep(50 * time.Nanosecond)
	}

	return next
}
