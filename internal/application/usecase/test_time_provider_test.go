package usecase

import (
	"fmt"
	"sync/atomic"
	"time"
)

type testTimeProvider struct {
	now      time.Time
	sequence int64
}

func newTestTimeProviderAt(t time.Time) *testTimeProvider {
	return &testTimeProvider{now: t}
}

func newTestMonotonicTimeProvider() *testTimeProvider {
	return &testTimeProvider{now: time.Now()}
}

func (p *testTimeProvider) Now() time.Time {
	return p.now
}

func (p *testTimeProvider) GenerateID() string {
	id := atomic.AddInt64(&p.sequence, 1)
	return fmt.Sprintf("test-id-%d", id)
}
