package multidevice

import "sync"

// VectorClock tracks causal relationships between events across devices.
// Each device maintains a logical counter; comparing two clocks reveals
// whether one happened before the other or they are concurrent.
type VectorClock struct {
	Clocks map[string]int64 `json:"clocks"` // deviceID -> logical timestamp
	mu     sync.RWMutex
}

// NewVectorClock creates a new empty vector clock.
func NewVectorClock() *VectorClock {
	return &VectorClock{
		Clocks: make(map[string]int64),
	}
}

// Increment advances the clock for a given device by one tick.
func (vc *VectorClock) Increment(deviceID string) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.Clocks[deviceID]++
}

// Get returns the current logical timestamp for a device. Returns 0 if the
// device has no entry in the clock.
func (vc *VectorClock) Get(deviceID string) int64 {
	vc.mu.RLock()
	defer vc.mu.RUnlock()
	return vc.Clocks[deviceID]
}

// Set sets the logical timestamp for a device.
func (vc *VectorClock) Set(deviceID string, ts int64) {
	vc.mu.Lock()
	defer vc.mu.Unlock()
	vc.Clocks[deviceID] = ts
}

// Merge combines two vector clocks by taking the maximum value of each
// component. This is used when a device receives events from another device.
func (vc *VectorClock) Merge(other *VectorClock) {
	if other == nil {
		return
	}
	vc.mu.Lock()
	defer vc.mu.Unlock()
	other.mu.RLock()
	defer other.mu.RUnlock()

	for deviceID, otherTS := range other.Clocks {
		if otherTS > vc.Clocks[deviceID] {
			vc.Clocks[deviceID] = otherTS
		}
	}
}

// Compare returns the causal relationship between two vector clocks:
//
//	"before"     - a happened before b
//	"after"      - a happened after b
//	"concurrent" - a and b are causally independent (conflict possible)
//	"equal"      - a and b are identical
func Compare(a, b *VectorClock) string {
	if a == nil || b == nil {
		return "concurrent"
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Collect all device IDs from both clocks.
	devices := make(map[string]struct{})
	for d := range a.Clocks {
		devices[d] = struct{}{}
	}
	for d := range b.Clocks {
		devices[d] = struct{}{}
	}

	aBeforeB := true // all a[d] <= b[d]
	bBeforeA := true // all b[d] <= a[d]
	equal := true

	for d := range devices {
		av := a.Clocks[d]
		bv := b.Clocks[d]
		if av != bv {
			equal = false
		}
		if av > bv {
			aBeforeB = false
		}
		if bv > av {
			bBeforeA = false
		}
	}

	if equal {
		return "equal"
	}
	if aBeforeB {
		return "before"
	}
	if bBeforeA {
		return "after"
	}
	return "concurrent"
}

// Copy returns a deep copy of the vector clock.
func (vc *VectorClock) Copy() *VectorClock {
	vc.mu.RLock()
	defer vc.mu.RUnlock()

	clone := &VectorClock{
		Clocks: make(map[string]int64, len(vc.Clocks)),
	}
	for k, v := range vc.Clocks {
		clone.Clocks[k] = v
	}
	return clone
}
