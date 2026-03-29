package time

import (
	"fmt"
	"sync/atomic"
	"time"
)

// RealTimeProvider uses real system time.
type RealTimeProvider struct{}

// NewRealTimeProvider creates a new RealTimeProvider.
func NewRealTimeProvider() *RealTimeProvider {
	return &RealTimeProvider{}
}

// Now returns the current system time.
func (p *RealTimeProvider) Now() time.Time {
	return time.Now()
}

// GenerateID generates a unique ID using current nanosecond timestamp.
func (p *RealTimeProvider) GenerateID() string {
	return fmt.Sprintf("%d", nextMonotonicUnixNanoID())
}

var lastGeneratedID int64

func nextMonotonicUnixNanoID() int64 {
	for {
		now := time.Now().UnixNano()
		last := atomic.LoadInt64(&lastGeneratedID)
		if now <= last {
			now = last + 1
		}
		if atomic.CompareAndSwapInt64(&lastGeneratedID, last, now) {
			return now
		}
	}
}

// FixedTimeProvider returns a fixed time (for tests).
type FixedTimeProvider struct {
	fixedTime time.Time
	idCounter int
}

// NewFixedTimeProvider creates a new FixedTimeProvider with a fixed time.
func NewFixedTimeProvider(t time.Time) *FixedTimeProvider {
	return &FixedTimeProvider{
		fixedTime: t,
		idCounter: 0,
	}
}

// Now returns the fixed time.
func (p *FixedTimeProvider) Now() time.Time {
	return p.fixedTime
}

// GenerateID generates deterministic test IDs.
func (p *FixedTimeProvider) GenerateID() string {
	p.idCounter++
	return fmt.Sprintf("test-id-%d", p.idCounter)
}

// SetTime updates the fixed time (useful for sequential tests).
func (p *FixedTimeProvider) SetTime(t time.Time) {
	p.fixedTime = t
}
