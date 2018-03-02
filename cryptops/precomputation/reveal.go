////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package precomputation

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Reveal implements the Reveal phase of the precomputation. It removes the
// cypher keys from the message and recipient cypher text, revealing the
// private keys for the round
type Reveal struct{}

// KeysReveal holds the keys used by the Reveal operation
type KeysReveal struct {
	// Private cypher key for all messages in the round
	// Generated in Generation phase
	Z *cyclic.Int
}

// Pre-allocate memory and arrange key objects for Precomputation Reveal phase
func (r Reveal) Build(g *cyclic.Group, face interface{}) *services.DispatchBuilder {

	// The empty interface should be castable to a Round
	round := face.(*globals.Round)

	// Allocate messages for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &PrecomputationSlot{
			Slot: i,
			MessagePrecomputation:     cyclic.NewMaxInt(),
			RecipientIDPrecomputation: cyclic.NewMaxInt(),
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Prepare the correct keys
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysReveal{Z: round.Z}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{
		BatchSize: round.BatchSize,
		Keys:      &keys,
		Output:    &om,
		G:         g,
	}

	return &db
}

// Input: Partial message cypher text, from Encrypt Phase
//        Partial recipient ID cypher text, from Permute Phase
// This phase removes the homomorphic encryption from these two quantities.
func (r Reveal) Run(g *cyclic.Group, in, out *PrecomputationSlot,
	keys *KeysReveal) services.Slot {

	// Eq 15.11 Root by cypher key to remove one layer of homomorphic
	// encryption from partially encrypted message cypher text.
	g.RootCoprime(in.MessagePrecomputation, keys.Z, out.MessagePrecomputation)

	// Eq 15.13 Root by cypher key to remove one layer of homomorphic
	// encryption from partially encrypted recipient ID cypher text.
	g.RootCoprime(in.RecipientIDPrecomputation, keys.Z,
		out.RecipientIDPrecomputation)

	out.MessageCypher = in.MessageCypher
	out.RecipientIDCypher = in.RecipientIDCypher

	return out
}