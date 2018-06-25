package katsubushi

import "time"

// ToTime returns the time when id was generated.
func ToTime(id uint64) time.Time {
	ts := id >> (WorkerIDBits + SequenceBits)
	d := time.Duration(int64(ts) * int64(time.Millisecond))
	return Epoch.Add(d)
}

// ToID returns the minimum id which will be generated at time t.
func ToID(t time.Time) uint64 {
	d := t.Sub(Epoch)
	ts := uint64(d.Nanoseconds()) / uint64(time.Millisecond)
	return ts << (WorkerIDBits + SequenceBits)
}

// Dump returns the structure of id.
func Dump(id uint64) (t time.Time, workerID uint64, sequence uint64) {
	workerID = (id & (workerIDMask << SequenceBits)) >> SequenceBits
	sequence = id & sequenceMask
	return ToTime(id), workerID, sequence
}
