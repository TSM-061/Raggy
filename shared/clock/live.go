package clock

import "time"

type LiveClock struct{}

func (*LiveClock) Now() time.Time {
	return time.Now()
}
