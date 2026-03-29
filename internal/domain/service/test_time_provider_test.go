package service

import (
	"fmt"
	"time"
)

type testTimeProvider struct {
	now       time.Time
	idCounter int
}

func newTestTimeProviderAt(t time.Time) *testTimeProvider {
	return &testTimeProvider{now: t}
}

func (p *testTimeProvider) Now() time.Time {
	return p.now
}

func (p *testTimeProvider) GenerateID() string {
	p.idCounter++
	return fmt.Sprintf("test-id-%d", p.idCounter)
}
