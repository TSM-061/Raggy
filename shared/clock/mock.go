package clock

import "time"

// MockClock is a Clock implementation for use in tests,
// returning a fixed time that can be set freely.
type MockClock struct {
	CurrentTime time.Time
}

func (m *MockClock) Now() time.Time {
	return m.CurrentTime
}
