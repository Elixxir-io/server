////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package realtime implements the realtime cryptographic phases of the cMix
// protocol as detailed in the cMix technical doc. To decrypt messages, the
// system goes through five phases, which are Decrypt, Permute, Identify,
// Encrypt, and Peel.
//
// The Decrypt phase removes the encryption added by the Client while
// simultaneously encrypting the message with unpermuted internode keys.
//
// The Permute phase mixes the slots, discarding information regarding who
// the sender is, while encrypting with permuted internode keys.
//
// The Identify phase fully decrypts all internode keys from the recipient.
//
// The Encrypt phase encrypts for the recipient.
//
// The peel phase removes the internode keys.
package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

// Decrypt phase completely removes the encryption added by the sending client,
// while adding in the First Unpermuted Internode Keys.  Becasue the unpermutted
// keys are added simultaniously, no entropy is lost.
type Decrypt struct{}

// SlotDecryptIn is used to pass external data into Decrypt
type SlotDecryptIn struct {
	//Slot Number of the Data
	Slot uint64
	// ID of the sending client (Pass through)
	CurrentID uint64
	// Message Encrypted with some Transmission Keys and some First Unpermuted
	// Internode Message Keys.
	Message *cyclic.Int
	// Recipient Encrypted with some Transmission Keys and some Unpermuted
	// Internode Recipient Keys.
	EncryptedRecipient *cyclic.Int
	// Next Ratchet of the sender's Transmission key
	CurrentKey *cyclic.Int
}

// SlotDecryptOut is used to pass the results out of Decrypt
type SlotDecryptOut struct {
	//Slot Number of the Data
	Slot uint64
	// ID of the sending client(Pass through)
	CurrentID uint64
	// Message Encrypted with a Transmission Key removed and a First Unpermuted
	// Internode Message Key added.
	Message *cyclic.Int
	// Recipient Encrypted with a Transmission Key removed and an Unpermuted
	// Internode Recipient Key added.
	EncryptedRecipient *cyclic.Int
}

// SlotID Returns the Slot number
func (e *SlotDecryptIn) SlotID() uint64 {
	return e.Slot
}

// ID of the user for keygen
func (e *SlotDecryptIn) UserID() uint64 {
	return e.CurrentID
}

// Cyclic int to place the key in
func (e *SlotDecryptIn) Key() *cyclic.Int {
	return e.CurrentKey
}

// Returns the KeyType
func (e *SlotDecryptIn) GetKeyType() cryptops.KeyType {
	return cryptops.TRANSMISSION
}

// SlotID Returns the Slot number
func (e *SlotDecryptOut) SlotID() uint64 {
	return e.Slot
}

// KeysDecrypt holds the keys used by the Decrypt Operation
type KeysDecrypt struct {
	// First Unpermuted Internode Message Key
	R *cyclic.Int
	// Unpermuted Internode Recipient Key
	U *cyclic.Int
}

// Allocated memory and arranges key objects for the Realtime Decrypt Phase
func (d Decrypt) Build(g *cyclic.Group,
	face interface{}) *services.DispatchBuilder {

	// Get round from the empty interface
	round := face.(*globals.Round)

	// Allocate Memory for output
	om := make([]services.Slot, round.BatchSize)

	for i := uint64(0); i < round.BatchSize; i++ {
		om[i] = &SlotDecryptOut{
			Slot:               i,
			Message:            cyclic.NewMaxInt(),
			EncryptedRecipient: cyclic.NewMaxInt(),
			CurrentID:          0,
		}
	}

	keys := make([]services.NodeKeys, round.BatchSize)

	// Link the keys for decryption
	for i := uint64(0); i < round.BatchSize; i++ {
		keySlc := &KeysDecrypt{
			R: round.R[i],
			U: round.U[i],
		}
		keys[i] = keySlc
	}

	db := services.DispatchBuilder{BatchSize: round.BatchSize,
		Keys: &keys, Output: &om, G: g}

	return &db

}

// Removes the encryption added by the Client while simultaneously
// encrypting the message with unpermuted internode keys.
func (d Decrypt) Run(g *cyclic.Group, in *SlotDecryptIn,
	out *SlotDecryptOut, keys *KeysDecrypt) services.Slot {

	// Create Temporary variable
	tmp := cyclic.NewMaxInt()

	// Eq 3.1: Modulo Multiplies the First Unpermuted Internode Message Key together
	// with with Transmission key before modulo multiplying into the
	// EncryptedMessage
	g.Mul(in.CurrentKey, keys.R, tmp)
	g.Mul(in.Message, tmp, out.Message)

	// Eq 3.3: Modulo Multiplies the Unpermuted Internode Recipient Key together
	// with with Transmission key before modulo multiplying into the
	// EncryptedRecipient
	g.Mul(in.CurrentKey, keys.U, tmp)
	g.Mul(in.EncryptedRecipient, tmp, out.EncryptedRecipient)

	// Pass through SenderID
	out.CurrentID = in.CurrentID
	return out

}
