package katsubushi

import "time"

func ToTime(id uint64) time.Time {
	ts := id>>(WorkerIDBits+SequenceBits) + TimestampSince
	sec := ts / 1000
	msec := ts % 1000
	return time.Unix(int64(sec), int64(msec)*int64(time.Millisecond))
}

func ToID(t time.Time) uint64 {
	ts := uint64(t.UnixNano() / int64(time.Millisecond))
	return (ts - TimestampSince) << (WorkerIDBits + SequenceBits)
}
