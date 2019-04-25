package phase

//The state a phase is in
type State uint32

const (
	//Initialized: Data structures for the phase have been created but it is not ready to run
	Initialized State = iota
	//Available: Next phase to run according to round but no input has been received
	Available
	//Queued: Next phase to run according to round and input has been received but it
	// has not begun execution by resource manager
	Queued
	//Running: Next phase to run according to round and input has been received and it
	// is being executed by resource manager
	Running
	//Finished: Phase is finished
	Finished
	// End of const block item: holds number of constants
	NumStates
)

//Array used to get the Phase Names for Printing
var stateStrings = [NumStates]string{"Initialized",
	"Available", "Queued", "Running", "Finished"}

// Adheres to the Stringer interface to return the name of the phase type
func (s State) String() string {
	return stateStrings[s]
}