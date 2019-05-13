////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io impl.go implements server utility functions needed to work
// with the comms library
package node

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/server"
	"time"
)

// NewImplementation creates a new implementation of the server.
// When a function is added to comms, you'll need to point to it here.
func NewImplementation(instance *server.Instance) *node.Implementation {

	impl := node.NewImplementation()

	impl.Functions.RoundtripPing = func(*mixmessages.TimePing) {}
	impl.Functions.GetServerMetrics = func(*mixmessages.ServerMetrics) {}
	impl.Functions.CreateNewRound = func(message *mixmessages.RoundInfo) {}

	// impl.Functions.StartRealtime =

	impl.Functions.GetRoundBufferInfo = func() (int, error) { return 0, nil }

	impl.Functions.PostPhase = func(batch *mixmessages.Batch) {
		PostPhaseFunc(batch, instance)
	}

	impl.Functions.PostRoundPublicKey = func(pk *mixmessages.RoundPublicKey) {
		PostRoundPublicKeyFunc(instance, pk, impl)
	}

	// impl.Functions.PostPrecompResult =

	//impl.Functions.RequestNonce =

	//impl.Functions.ConfirmRegistration =

	impl.Functions.GetRoundBufferInfo = func() (int, error) {
		return io.GetRoundBufferInfo(instance.GetCompletedPrecomps(), time.Second)
	}

	impl.Functions.GetCompletedBatch = func() (batch *mixmessages.Batch, e error) {
		return io.GetCompletedBatch(instance.GetCompletedBatchQueue(), time.Second)
	}

	//impl.Functions.PostRoundPublicKey =

	impl.Functions.RequestNonce = func(salt, Y, P, Q, G, hash, R, S []byte) ([]byte, error) {
		return io.RequestNonce(instance, salt, Y, P, Q, G, hash, R, S)
	}

	impl.Functions.ConfirmRegistration = func(hash, R, S []byte) ([]byte, []byte, []byte,
		[]byte, []byte, []byte, []byte, error) {
		return io.ConfirmRegistration(instance, hash, R, S)
	}

	//impl.Functions.PostPrecompResult =

	return impl
}
