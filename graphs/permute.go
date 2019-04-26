////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package graphs

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/services"
)

// PermuteIO used to convert input and output when streams are linked
type PermuteIO struct {
	Input  *cyclic.IntBuffer
	Output []*cyclic.Int
}

// ModifyGraphGeneratorForPermute makes a copy of the graph generator
// where the OutputThreshold=1.0
func ModifyGraphGeneratorForPermute(gc services.GraphGenerator) services.GraphGenerator {
	return services.NewGraphGenerator(
		gc.GetMinInputSize(),
		gc.GetErrorHandler(),
		gc.GetDefaultNumTh(),
		gc.GetOutputSize(),
		1.0,
	)
}
