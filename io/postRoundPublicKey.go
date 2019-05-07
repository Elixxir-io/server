////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/mixmessages"
	comm "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/server/round"
	"gitlab.com/elixxir/server/services"
)

// TransmitRoundPublicKey sends the public key to every node
// in the round
func TransmitRoundPublicKey(pubKey *cyclic.Int, roundID id.Round,
	nal *services.NodeAddressList) error {

	// Create the message structure to send the messages
	roundPubKeyMsg := &mixmessages.RoundPublicKey{
		Round: &mixmessages.RoundInfo{
			ID: uint64(roundID),
		},
		Key: pubKey.Bytes(),
	}

	for _, recipient := range nal.GetAllNodesAddress() {
		// Make sure the comm doesn't return an Ack with an
		// error message
		ack, err := comm.SendPostRoundPublicKey(
			recipient.Address, recipient.Cert, roundPubKeyMsg)
		if ack != nil && ack.Error != "" {
			err = errors.Errorf("Remote Server Error: %s, %s",
				recipient.Address, ack.Error)
		}
		return err
	}

	return nil
}

// PostRoundPublicKey implements the server gRPC handler for
// posting a public key to the round
// Transmission handler
func PostRoundPublicKey(grp *cyclic.Group, roundBuff *round.Buffer, pk *mixmessages.RoundPublicKey) error {

	inside := grp.BytesInside(pk.GetKey())

	if !inside {
		return node.ErrOutsideOfGroup
	}

	grp.SetBytes(roundBuff.CypherPublicKey, pk.GetKey())

	return nil
}
