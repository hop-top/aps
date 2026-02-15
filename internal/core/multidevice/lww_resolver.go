package multidevice

import (
	"fmt"
	"time"
)

// LWWResolver implements last-write-wins conflict resolution. When two
// events conflict, the event with the latest timestamp wins. In the rare
// case of identical timestamps, the lexicographically greater device ID
// is used as a deterministic tiebreaker.
type LWWResolver struct{}

// NewLWWResolver creates a new last-write-wins resolver.
func NewLWWResolver() *LWWResolver {
	return &LWWResolver{}
}

// Resolve picks a winner among the conflicting events based on their
// timestamps. Returns a ConflictResolution describing the outcome.
func (r *LWWResolver) Resolve(conflict *Conflict) (*ConflictResolution, error) {
	if conflict == nil {
		return nil, fmt.Errorf("conflict must not be nil")
	}
	if len(conflict.Events) == 0 {
		return nil, fmt.Errorf("conflict has no events to resolve")
	}

	winner := conflict.Events[0]
	for _, event := range conflict.Events[1:] {
		if event.Timestamp.After(winner.Timestamp) {
			winner = event
		} else if event.Timestamp.Equal(winner.Timestamp) {
			// Deterministic tiebreaker: lexicographically greater device ID wins.
			if event.DeviceID > winner.DeviceID {
				winner = event
			}
		}
	}

	now := time.Now()
	resolution := &ConflictResolution{
		Strategy:    "lww",
		WinnerEvent: winner.ID,
		Result:      winner.Payload,
		ResolvedBy:  "auto",
	}

	conflict.Status = ConflictAutoResolved
	conflict.ResolvedAt = &now
	conflict.Resolution = resolution

	return resolution, nil
}
