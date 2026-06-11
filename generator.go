package katsubushi

import (
	"errors"
	"slices"
	"sync"
	"time"
)

var nowFunc = time.Now
var nowMutex sync.RWMutex

func setNowFunc(f func() time.Time) {
	nowMutex.Lock()
	defer nowMutex.Unlock()
	nowFunc = f
}

func now() time.Time {
	nowMutex.RLock()
	defer nowMutex.RUnlock()
	return nowFunc()
}

// Epoch is katsubushi epoch time (2015-01-01 00:00:00 UTC)
// Generated ID includes elapsed time from Epoch.
var Epoch = time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)

// for bitshift
const (
	WorkerIDBits = 10
	SequenceBits = 12
)

// Bit layout of the JS-safe ID format.
// IDs in this format fit within 2^53-1 (Number.MAX_SAFE_INTEGER in JavaScript),
// so they are not corrupted even when handled as a double precision float.
const (
	JSSafeWorkerIDBits  = 6
	JSSafeSequenceBits  = 8
	JSSafeTimestampUnit = 10 * time.Millisecond

	// JSSafeMaxID is the maximum value of IDs in the JS-safe format,
	// which equals Number.MAX_SAFE_INTEGER in JavaScript.
	// IDs in the default format always exceed this value in practice,
	// so it can be used to determine the format of an ID.
	JSSafeMaxID = 1<<53 - 1
)

// idSpec defines a bit layout of generated IDs.
type idSpec struct {
	name          string
	workerIDBits  uint
	sequenceBits  uint
	timestampUnit time.Duration
}

func (s idSpec) workerIDMask() uint64 {
	return 1<<s.workerIDBits - 1
}

func (s idSpec) sequenceMask() uint {
	return 1<<s.sequenceBits - 1
}

func (s idSpec) timestampShift() uint {
	return s.workerIDBits + s.sequenceBits
}

var (
	defaultSpec = idSpec{
		name:          "default",
		workerIDBits:  WorkerIDBits,
		sequenceBits:  SequenceBits,
		timestampUnit: time.Millisecond,
	}
	jsSafeSpec = idSpec{
		name:          "js-safe",
		workerIDBits:  JSSafeWorkerIDBits,
		sequenceBits:  JSSafeSequenceBits,
		timestampUnit: JSSafeTimestampUnit,
	}
)

// workerIDPools keeps worker IDs in use, by format name.
// Worker IDs must be unique within a format.
var workerIDPools = map[string][]uint{}
var newGeneratorLock sync.Mutex

// errors
var (
	ErrInvalidWorkerID    = errors.New("invalid worker id")
	ErrDuplicatedWorkerID = errors.New("duplicated worker")
)

func checkWorkerID(id uint, spec idSpec) error {
	if uint(spec.workerIDMask()) < id {
		return ErrInvalidWorkerID
	}

	if slices.Contains(workerIDPools[spec.name], id) {
		return ErrDuplicatedWorkerID
	}

	return nil
}

// Generator is an interface to generate unique ID.
type Generator interface {
	NextID() (uint64, error)
	WorkerID() uint
}

type generator struct {
	spec          idSpec
	workerID      uint
	lastTimestamp uint64
	sequence      uint
	lock          sync.Mutex
	startedAt     time.Time
	offset        time.Duration
}

// NewGenerator returns new generator.
func NewGenerator(workerID uint) (Generator, error) {
	return newGenerator(workerID, defaultSpec)
}

// NewJSSafeGenerator returns new generator which generates IDs in the JS-safe format.
// Generated IDs fit within 2^53-1 (Number.MAX_SAFE_INTEGER in JavaScript).
// IDs in the JS-safe format are not compatible with IDs in the default format,
// so do not mix both formats in a service.
func NewJSSafeGenerator(workerID uint) (Generator, error) {
	return newGenerator(workerID, jsSafeSpec)
}

func newGenerator(workerID uint, spec idSpec) (Generator, error) {
	// To keep worker ID be unique.
	newGeneratorLock.Lock()
	defer newGeneratorLock.Unlock()

	if err := checkWorkerID(workerID, spec); err != nil {
		return nil, err
	}

	// save as already used
	workerIDPools[spec.name] = append(workerIDPools[spec.name], workerID)

	n := now()
	return &generator{
		spec:      spec,
		workerID:  workerID,
		startedAt: n,
		offset:    n.Sub(Epoch),
	}, nil
}

func (g *generator) WorkerID() uint {
	return g.workerID
}

func (g *generator) formatName() string {
	return g.spec.name
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
		g.sequence = (g.sequence + 1) & g.spec.sequenceMask()
		if g.sequence == 0 {
			// overflow
			ts = g.waitUntilNextTick(ts)
		}
	} else {
		g.sequence = 0
	}
	g.lastTimestamp = ts

	return (g.lastTimestamp << g.spec.timestampShift()) | (uint64(g.workerID) << g.spec.sequenceBits) | (uint64(g.sequence)), nil
}

func (g *generator) timestamp() uint64 {
	d := now().Sub(g.startedAt) + g.offset
	return uint64(d.Nanoseconds()) / uint64(g.spec.timestampUnit)
}

func (g *generator) waitUntilNextTick(ts uint64) uint64 {
	// sleep for 1/100 of the timestamp unit not to burn CPU.
	// e.g. 10us for 1ms unit, 100us for 10ms unit.
	interval := g.spec.timestampUnit / 100
	next := g.timestamp()

	for next <= ts {
		time.Sleep(interval)
		next = g.timestamp()
	}

	return next
}
