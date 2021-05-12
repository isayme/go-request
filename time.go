package request

import "time"

func toMs(d time.Duration) int64 {
	return int64(d / time.Millisecond)
}
