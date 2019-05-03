////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package services

import (
	jww "github.com/spf13/jwalterweatherman"
)

// Should probably add more params to this like block ID, worker thread ID, etc
type ErrorCallback func(graph, module string, err error)

type GraphGenerator struct {
	minInputSize    uint32
	errorHandler    ErrorCallback
	defaultNumTh    uint8
	outputSize      uint32
	outputThreshold float32
}

func NewGraphGenerator(minInputSize uint32, errorHandler ErrorCallback, defaultNumTh uint8, outputSize uint32, outputThreshold float32) GraphGenerator {
	if defaultNumTh > MAX_THREADS {
		jww.FATAL.Panicf("Max threads per module is 64, "+
			"cannot default to %v threads", defaultNumTh)
	}
	if defaultNumTh == 0 {
		jww.FATAL.Panicf("Cannot default to zero threads")
	}

	if minInputSize == 0 {
		jww.FATAL.Panicf("Minimum input size must be greater than zero")
	}

	if outputSize == AutoOutputSize {
		outputSize = minInputSize
	}

	if outputThreshold < 0.0 || outputThreshold > 1.0 {
		jww.FATAL.Panicf("Output Threshold must be between 0.0 and 1."+
			"0: recieved: %v", outputThreshold)
	}

	return GraphGenerator{
		minInputSize:    minInputSize,
		errorHandler:    errorHandler,
		defaultNumTh:    defaultNumTh,
		outputSize:      outputSize,
		outputThreshold: outputThreshold,
	}
}

func (gc GraphGenerator) GetMinInputSize() uint32 {
	return gc.minInputSize
}

func (gc GraphGenerator) GetDefaultNumTh() uint8 {
	return gc.defaultNumTh
}

func (gc GraphGenerator) GetErrorHandler() ErrorCallback {
	return gc.errorHandler
}

func (gc GraphGenerator) GetOutputSize() uint32 {
	return gc.outputSize
}

func (gc GraphGenerator) GetOutputThreshold() float32 {
	return gc.outputThreshold
}

func (gc GraphGenerator) NewGraph(name string, stream Stream) *Graph {

	var g Graph
	g.generator = gc
	g.modules = make(map[uint64]*Module)
	g.idCount = 0
	g.batchSize = 0
	g.expandBatchSize = 0

	g.name = name

	g.built = false
	g.linked = false

	g.stream = stream

	g.sentInputs = new(uint32)

	g.outputSize = gc.outputSize
	g.outputThreshold = gc.outputThreshold

	return &g
}
