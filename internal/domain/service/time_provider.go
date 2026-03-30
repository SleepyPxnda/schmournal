package service

import "time"

// TimeProvider abstracts time-related operations needed by domain/application logic.
// Infrastructure provides concrete implementations (real time, fixed time for tests).
type TimeProvider interface {
	Now() time.Time
	GenerateID() string
}
