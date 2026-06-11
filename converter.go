package katsubushi

import "time"

// ToTime returns the time when id was generated.
func ToTime(id uint64) time.Time {
	return defaultSpec.toTime(id)
}

// ToID returns the minimum id which will be generated at time t.
func ToID(t time.Time) uint64 {
	return defaultSpec.toID(t)
}

// Dump returns the structure of id.
func Dump(id uint64) (t time.Time, workerID uint64, sequence uint64) {
	return defaultSpec.dump(id)
}

// ToTimeJSSafe returns the time when id in the JS-safe format was generated.
func ToTimeJSSafe(id uint64) time.Time {
	return jsSafeSpec.toTime(id)
}

// ToIDJSSafe returns the minimum id in the JS-safe format which will be generated at time t.
func ToIDJSSafe(t time.Time) uint64 {
	return jsSafeSpec.toID(t)
}

// DumpJSSafe returns the structure of id in the JS-safe format.
func DumpJSSafe(id uint64) (t time.Time, workerID uint64, sequence uint64) {
	return jsSafeSpec.dump(id)
}

func (s idSpec) toTime(id uint64) time.Time {
	ts := id >> s.timestampShift()
	d := time.Duration(int64(ts) * int64(s.timestampUnit))
	return Epoch.Add(d)
}

func (s idSpec) toID(t time.Time) uint64 {
	d := t.Sub(Epoch)
	ts := uint64(d.Nanoseconds()) / uint64(s.timestampUnit)
	return ts << s.timestampShift()
}

func (s idSpec) dump(id uint64) (t time.Time, workerID uint64, sequence uint64) {
	workerID = (id >> s.sequenceBits) & s.workerIDMask()
	sequence = id & uint64(s.sequenceMask())
	return s.toTime(id), workerID, sequence
}
