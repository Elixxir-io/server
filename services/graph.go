package services

import (
	"gitlab.com/elixxir/server/globals"
	"math"
	"time"
)

// Should probably add more params to this like block ID, worker thread ID, etc
type ErrorCallback func(err error)

type Graph struct {
	callback    ErrorCallback
	modules     map[uint64]*Module
	firstModule *Module
	lastModule  *Module

	name string

	outputModule *Module

	idCount uint64

	stream Stream

	batchSize       uint32
	expandBatchSize uint32

	built  bool
	linked bool

	outputChannel IO_Notify
}

func NewGraph(name string, callback ErrorCallback, stream Stream) *Graph {
	var g Graph
	g.callback = callback
	g.modules = make(map[uint64]*Module)
	g.idCount = 0
	g.batchSize = 0
	g.expandBatchSize = 0

	g.name = name

	g.built = false
	g.linked = false

	g.stream = stream

	return &g
}

// This is too long of a function
func (g *Graph) Build(batchSize uint32) {
	//Checks graph is properly formatted
	g.checkGraph()

	//Find expanded batch size
	var integers []uint32

	for _, m := range g.modules {
		m.checkParameters(globals.MinSlotSize)
		if m.InputSize != INPUT_IS_BATCHSIZE {
			integers = append(integers, m.InputSize)
		}
	}

	integers = append(integers, globals.MinSlotSize)
	lcm := globals.LCM(integers)

	expandBatchSize := uint32(math.Ceil(float64(batchSize)/float64(lcm))) * lcm

	g.batchSize = batchSize
	g.expandBatchSize = expandBatchSize

	/*setup output*/
	g.outputModule = &Module{
		InputSize:    globals.MinSlotSize,
		inputModules: []*Module{g.lastModule},
		Name:         "Output",
		copy:         true,
	}

	g.lastModule.outputModules = append(g.lastModule.outputModules, g.outputModule)
	g.add(g.outputModule)

	/*build assignments*/
	for _, m := range g.modules {
		m.buildAssignments(expandBatchSize)
	}

	g.built = true

	//populate channels
	for _, m := range g.modules {
		m.open()
	}

	/*finish setting up output*/
	g.outputChannel = g.outputModule.input

	delete(g.modules, g.outputModule.id)
}

func (g *Graph) checkGraph() {
	//Check if graph has modules
	if len(g.modules) == 0 {
		panic("No modules in graph")
	}

	if g.firstModule == nil {
		panic("No first module")
	}

	if g.lastModule == nil {
		panic("No last module")
	}
}

func (g *Graph) Run() {
	if !g.built {
		panic("graph not built")
	}

	if !g.linked {
		panic("stream not linked and built")
	}

	for _, m := range g.modules {

		m.state.numTh = uint8(m.NumThreads)
		m.state.Init()

		for i := uint8(0); i < m.state.numTh; i++ {
			go dispatch(g, m, uint8(i))
		}
	}
}

func (g *Graph) Connect(a, b *Module) {

	g.add(a)
	g.add(b)

	a.outputModules = append(a.outputModules, b)
	b.inputModules = append(b.inputModules, a)
}

func (g *Graph) Link(source interface{}) {
	g.stream.Link(g.expandBatchSize, source)
	g.linked = true
}

func (g *Graph) First(f *Module) {
	g.add(f)
	g.firstModule = f
}

func (g *Graph) Last(l *Module) {
	g.add(l)
	g.lastModule = l
}

func (g *Graph) add(m *Module) {
	if !m.copy {
		panic("cannot build a graph with an original module, must use a copy")
	}
	m.used = true
	_, ok := g.modules[m.id]

	if !ok {
		g.idCount++
		m.id = g.idCount
		g.modules[m.id] = m
	}
}

func (g *Graph) GetStream() Stream {
	return g.stream
}

func (g *Graph) Send(sr Chunk) {

	srList := g.firstModule.assignmentList.PrimeOutputs(sr)

	for _, r := range srList {
		g.firstModule.input <- r
	}

	done := g.firstModule.assignmentList.DenoteCompleted(len(srList))

	if done {
		// FIXME: Perhaps not the correct place to close the channel.
		// Ideally, only the sender closes, and only if there's one sender.
		// Does commenting this fix the double close?
		// It does not.
		g.firstModule.closeInput()
	}
}

// Outputs from the last op in the graph get sent on this channel.
func (g *Graph) ChunkDoneChannel() IO_Notify {
	return g.outputChannel
}

func (g *Graph) GetExpandedBatchSize() uint32 {
	return g.expandBatchSize
}

func (g *Graph) GetBatchSize() uint32 {
	return g.batchSize
}

func (g *Graph) GetName() string {
	return g.name
}

// This doesn't quite seem robust
func (g *Graph) Kill() bool {
	success := true
	for _, m := range g.modules {
		success = success && m.state.Kill(time.Millisecond*10)
	}
	return success
}
