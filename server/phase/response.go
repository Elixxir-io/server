package phase

import (
	"fmt"
)

// Response describes how the round should act when given an input coming
// from a specific phase.  The specific phase is looked up in the
// ResponseMap and the specifics are used to determine how to proceed
type ResponseMap map[string]Response

type Response interface {
	CheckState(State) bool
	GetPhaseLookup() Type
	GetReturnPhase() Type
	GetExpectedStates() []State
	fmt.Stringer
}

type response struct {
	phaseLookup    Type
	returnPhase    Type
	expectedStates []State
}

//NewResponse Builds a new CMIX phase response adhering to the Response interface
// The phase should be in one of the states in expectedStates for processing to
// continue.
func NewResponse(lookup, rtn Type, expectedStates ...State) Response {
	return response{phaseLookup: lookup, returnPhase: rtn, expectedStates: expectedStates}
}

//GetPhaseLookup Returns the phaseLookup
func (r response) GetPhaseLookup() Type {
	return r.phaseLookup
}

//GetReturnPhase returns the returnPhase
func (r response) GetReturnPhase() Type {
	return r.returnPhase
}

//GetExpectedStates returns the expected states as a slice
func (r response) GetExpectedStates() []State {
	return r.expectedStates
}

// CheckState returns true if the passed state is in
// the expected states list, otherwise it returns false
func (r response) CheckState(state State) bool {
	for _, expected := range r.expectedStates {
		if state == expected {
			return true
		}
	}

	return false
}

// String adheres to the stringer interface
func (r response) String() string {
	validStates := "{'"

	for _, s := range r.expectedStates[:len(r.expectedStates)-1] {
		validStates += s.String() + "', '"
	}

	validStates += r.expectedStates[len(r.expectedStates)-1].String() + "'}"

	return fmt.Sprintf("phase.Responce{phaseLookup: '%s', returnPhase:'%s', expectedStates: %s}",
		r.phaseLookup, r.returnPhase, validStates)
}
