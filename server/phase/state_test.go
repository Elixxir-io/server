package phase

import (
	"sync/atomic"
	"testing"
)

// Proves that a single phase can be taken through all of its states
// using the provided utility methods
func TestPhaseStateIncrement(t *testing.T) {
	g := NewStateGroup()
	index, state := g.newState()
	p := Phase{
		tYpe:                g.GetCurrentPhase(),
		state:               state,
		stateIndex:          index,
		stateGroup:          g,
	}

	atomic.StoreUint32(state, uint32(Available))
	expected := Available
    if atomic.LoadUint32(state) != uint32(expected) {
    	t.Errorf("State was %v, but should have been %v",
    		State(atomic.LoadUint32(state)), expected)
	}
	p.IncrementPhaseToQueued()
	expected = Queued
	if g.GetState(index) != expected {
		t.Errorf("State was %v, but should have been %v",
			State(atomic.LoadUint32(state)), expected)
	}
	p.IncrementPhaseToRunning()
	expected = Running
	if g.GetState(index) != expected {
		t.Errorf("State was %v, but should have been %v",
			State(atomic.LoadUint32(state)), expected)
	}
	p.Finish()
	expected = Finished
	if g.GetState(index) != expected {
		t.Errorf("State was %v, but should have been %v",
			State(atomic.LoadUint32(state)), expected)
	}
}
