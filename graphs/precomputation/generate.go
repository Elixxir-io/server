////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Generate phase
// Generate phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// GenerateStream holds the inputs for the Generate operation
type GenerateStream struct {
	Grp *cyclic.Group

	// RNG
	rngConstructor csprng.SourceConstructor

	// Phase Keys
	R *cyclic.IntBuffer
	S *cyclic.IntBuffer
	U *cyclic.IntBuffer
	V *cyclic.IntBuffer

	// Share keys for each phase
	YR *cyclic.IntBuffer
	YS *cyclic.IntBuffer
	YU *cyclic.IntBuffer
	YV *cyclic.IntBuffer
}

// GetName returns the name of this op
func (gs *GenerateStream) GetName() string {
	return "PrecompGenerateStream"
}

// Link maps the round data to the Generate Stream data structure (the input)
func (gs *GenerateStream) Link(grp *cyclic.Group, batchSize uint32, source ...interface{}) {
	roundBuffer := source[0].(*round.Buffer)
	rngConstructor := source[2].(func() csprng.Source)

	gs.LinkGenerateStream(grp, batchSize, roundBuffer, rngConstructor)
}

// Link maps the round data to the Generate Stream data structure (the input)
func (gs *GenerateStream) LinkGenerateStream(grp *cyclic.Group, batchSize uint32, roundBuffer *round.Buffer,
	rngConstructor csprng.SourceConstructor) {

	gs.Grp = grp

	gs.rngConstructor = rngConstructor

	// Phase keys
	gs.R = roundBuffer.R.GetSubBuffer(0, batchSize)
	gs.S = roundBuffer.S.GetSubBuffer(0, batchSize)
	gs.U = roundBuffer.U.GetSubBuffer(0, batchSize)
	gs.V = roundBuffer.V.GetSubBuffer(0, batchSize)

	// Share keys
	gs.YR = roundBuffer.Y_R.GetSubBuffer(0, batchSize)
	gs.YS = roundBuffer.Y_S.GetSubBuffer(0, batchSize)
	gs.YU = roundBuffer.Y_U.GetSubBuffer(0, batchSize)
	gs.YV = roundBuffer.Y_V.GetSubBuffer(0, batchSize)
}

type GenerateSubstreamInterface interface {
	GetGenerateSubStream() *GenerateStream
}

// getSubStream implements reveal interface to return stream object
func (gs *GenerateStream) GetGenerateSubStream() *GenerateStream {
	return gs
}

// Input function pulls things from the mixmessage
func (gs *GenerateStream) Input(index uint32, slot *mixmessages.Slot) error {
	if index >= uint32(gs.R.Len()) {
		return services.ErrOutsideOfBatch
	}
	return nil
}

// Output returns an empty cMixSlot message
func (gs *GenerateStream) Output(index uint32) *mixmessages.Slot {
	return &mixmessages.Slot{}
}

// Generate implements cryptops.Generate for precomputation
var Generate = services.Module{
	// Generates key pairs R, S, U, and V
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop,
		chunk services.Chunk) error {
		gssi, ok := streamInput.(GenerateSubstreamInterface)
		generate, ok2 := cryptop.(cryptops.GeneratePrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		gs := gssi.GetGenerateSubStream()

		rng := gs.rngConstructor()

		for i := chunk.Begin(); i < chunk.End(); i++ {
			errors := []error{
				generate(gs.Grp, gs.R.Get(i), gs.YR.Get(i), rng),
				generate(gs.Grp, gs.S.Get(i), gs.YS.Get(i), rng),
				generate(gs.Grp, gs.U.Get(i), gs.YU.Get(i), rng),
				generate(gs.Grp, gs.V.Get(i), gs.YV.Get(i), rng),
			}
			for _, err := range errors {
				if err != nil {
					jww.CRITICAL.Panicf(err.Error())
				}
			}
		}
		return nil
	},
	Cryptop:    cryptops.Generate,
	NumThreads: services.AutoNumThreads,
	InputSize:  services.AutoInputSize,
	Name:       "Generate",
}

// InitGenerateGraph initializes a new generate graph
func InitGenerateGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("PrecompGenerate", &GenerateStream{})

	generate := Generate.DeepCopy()

	g.First(generate)
	g.Last(generate)

	return g
}
