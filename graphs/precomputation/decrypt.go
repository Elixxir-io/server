package precomputation

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cryptops"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/node"
	"gitlab.com/elixxir/server/services"
)

// This file implements the Graph for the Precomputation Decrypt phase
// Decrypt phase transforms first unpermuted internode keys
// and partial cypher texts into the data that the permute phase needs

// Stream holding data containing keys and inputs used by decrypt
type DecryptStream struct {
	Grp             *cyclic.Group
	PublicCypherKey *cyclic.Int

	//Link to round object
	R *cyclic.IntBuffer
	U *cyclic.IntBuffer

	Y_R *cyclic.IntBuffer
	Y_U *cyclic.IntBuffer

	//Unique to stream
	KeysMsg   *cyclic.IntBuffer
	CypherMsg *cyclic.IntBuffer
	KeysAD    *cyclic.IntBuffer
	CypherAD  *cyclic.IntBuffer
}

func (s *DecryptStream) GetName() string {
	return "PrecompDecryptStream"
}

func (s *DecryptStream) Link(batchSize uint32, source interface{}) {
	round := source.(*node.RoundBuffer)

	s.Grp = round.Grp
	s.PublicCypherKey = round.CypherPublicKey

	s.R = round.R.GetSubBuffer(0, batchSize)
	s.U = round.U.GetSubBuffer(0, batchSize)
	s.Y_R = round.Y_R.GetSubBuffer(0, batchSize)
	s.Y_U = round.Y_U.GetSubBuffer(0, batchSize)

	s.KeysMsg = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.CypherMsg = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.KeysAD = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
	s.CypherAD = s.Grp.NewIntBuffer(batchSize, s.Grp.NewInt(1))
}

func (s *DecryptStream) Input(index uint32, slot *mixmessages.CmixSlot) error {

	if index >= uint32(s.KeysMsg.Len()) {
		return node.ErrOutsideOfBatch
	}

	if !s.Grp.BytesInside(slot.EncryptedMessageKeys, slot.PartialMessageCypherText,
		slot.EncryptedAssociatedDataKeys, slot.PartialAssociatedDataCypherText) {
		return node.ErrOutsideOfGroup
	}

	s.Grp.SetBytes(s.KeysMsg.Get(index), slot.EncryptedMessageKeys)
	s.Grp.SetBytes(s.KeysAD.Get(index), slot.EncryptedAssociatedDataKeys)
	s.Grp.SetBytes(s.CypherMsg.Get(index), slot.PartialMessageCypherText)
	s.Grp.SetBytes(s.CypherAD.Get(index), slot.PartialAssociatedDataCypherText)
	return nil
}

func (s *DecryptStream) Output(index uint32) *mixmessages.CmixSlot {

	return &mixmessages.CmixSlot{
		EncryptedMessageKeys:            s.KeysMsg.Get(index).Bytes(),
		EncryptedAssociatedDataKeys:     s.KeysAD.Get(index).Bytes(),
		PartialMessageCypherText:        s.CypherMsg.Get(index).Bytes(),
		PartialAssociatedDataCypherText: s.CypherAD.Get(index).Bytes(),
	}
}

//Sole module in Precomputation Decrypt implementing cryptops.Elgamal
var DecryptElgamal = services.Module{
	// Multiplies in own Encrypted Keys and Partial Cypher Texts
	Adapt: func(streamInput services.Stream, cryptop cryptops.Cryptop, chunk services.Chunk) error {
		s, ok := streamInput.(*DecryptStream)
		elgamal, ok2 := cryptop.(cryptops.ElGamalPrototype)

		if !ok || !ok2 {
			return services.InvalidTypeAssert
		}

		for i := chunk.Begin(); i < chunk.End(); i++ {
			//Execute elgamal on the keys for the Message
			elgamal(s.Grp, s.R.Get(i), s.Y_R.Get(i), s.PublicCypherKey, s.KeysMsg.Get(i), s.CypherMsg.Get(i))
			//Execute elgamal on the keys for the Associated Data
			elgamal(s.Grp, s.U.Get(i), s.Y_U.Get(i), s.PublicCypherKey, s.KeysAD.Get(i), s.CypherAD.Get(i))
		}
		return nil
	},
	Cryptop:    cryptops.ElGamal,
	NumThreads: 5,
	InputSize:  services.AUTO_INPUTSIZE,
	Name:       "DecryptElgamal",
}

//Called to initialize the graph. Conforms to graphs.Initialize function type
func InitDecryptGraph(gc services.GraphGenerator) *services.Graph {
	g := gc.NewGraph("PrecompDecrypt", &DecryptStream{})

	decryptElgamal := DecryptElgamal.DeepCopy()

	g.First(decryptElgamal)
	g.Last(decryptElgamal)

	return g
}