package node

import (
	"gitlab.com/privategrity/crypto/cyclic"
)

// LastNode contains precomputations held only by the last node
type LastNode struct {
	// Message Decryption key, AKA PiRST_Inv
	MessagePrecomputation []*cyclic.Int
	// Recipient ID Decryption Key, AKA PiUV_Inv
	RecipientPrecomputation []*cyclic.Int
}

// Round contains the keys and permutations for a given message batch
type Round struct {
	R            []*cyclic.Int // First unpermuted internode message key
	S            []*cyclic.Int // Permuted internode message key
	T            []*cyclic.Int // Second unpermuted internode message key
	V            []*cyclic.Int // Unpermuted internode recipient key
	U            []*cyclic.Int // Permuted *cyclic.Internode receipient key
	R_INV        []*cyclic.Int // First Inverse unpermuted internode message key
	S_INV        []*cyclic.Int // Permuted Inverse internode message key
	T_INV        []*cyclic.Int // Second Inverse unpermuted internode message key
	V_INV        []*cyclic.Int // Unpermuted Inverse internode recipient key
	U_INV        []*cyclic.Int // Permuted Inverse *cyclic.Internode receipient key
	Permutations []uint64      // Permutation array, messages at index i become
	// messages at index Permutations[i]
	CypherPublicKey *cyclic.Int // Global Cypher Key
	Z               *cyclic.Int // This node's Cypher Key
	// Private keys for the above
	Y_R []*cyclic.Int
	Y_S []*cyclic.Int
	Y_T []*cyclic.Int
	Y_V []*cyclic.Int
	Y_U []*cyclic.Int

	// Variables only carried by the last node
	LastNode

	BatchSize uint64
}

// Grp is the cyclic group that all operations are done within
var Grp *cyclic.Group

// Rounds is a mapping of session identifiers to round structures
var Rounds map[string]*Round

var TestArray = [2]float32{.03, .02}

// NewRound constructs an empty round for a given batch size, with all
// numbers being initialized to 0.
func NewRound(batchSize uint64) *Round {
	NR := Round{
		R: make([]*cyclic.Int, batchSize),
		S: make([]*cyclic.Int, batchSize),
		T: make([]*cyclic.Int, batchSize),
		V: make([]*cyclic.Int, batchSize),
		U: make([]*cyclic.Int, batchSize),

		R_INV: make([]*cyclic.Int, batchSize),
		S_INV: make([]*cyclic.Int, batchSize),
		T_INV: make([]*cyclic.Int, batchSize),
		V_INV: make([]*cyclic.Int, batchSize),
		U_INV: make([]*cyclic.Int, batchSize),

		CypherPublicKey: cyclic.NewInt(0),
		Z:               cyclic.NewInt(0),

		Permutations: make([]uint64, batchSize),

		Y_R: make([]*cyclic.Int, batchSize),
		Y_S: make([]*cyclic.Int, batchSize),
		Y_T: make([]*cyclic.Int, batchSize),
		Y_V: make([]*cyclic.Int, batchSize),
		Y_U: make([]*cyclic.Int, batchSize),

		BatchSize: batchSize}

	NR.CypherPublicKey.SetBytes(cyclic.Max4kBitInt)
	NR.Z.SetBytes(cyclic.Max4kBitInt)

	for i := uint64(0); i < batchSize; i++ {
		NR.R[i] = cyclic.NewMaxInt()
		NR.S[i] = cyclic.NewMaxInt()
		NR.T[i] = cyclic.NewMaxInt()
		NR.V[i] = cyclic.NewMaxInt()
		NR.U[i] = cyclic.NewMaxInt()

		NR.R_INV[i] = cyclic.NewMaxInt()
		NR.S_INV[i] = cyclic.NewMaxInt()
		NR.T_INV[i] = cyclic.NewMaxInt()
		NR.V_INV[i] = cyclic.NewMaxInt()
		NR.U_INV[i] = cyclic.NewMaxInt()

		NR.Y_R[i] = cyclic.NewMaxInt()
		NR.Y_S[i] = cyclic.NewMaxInt()
		NR.Y_T[i] = cyclic.NewMaxInt()
		NR.Y_V[i] = cyclic.NewMaxInt()
		NR.Y_U[i] = cyclic.NewMaxInt()

		NR.Permutations[i] = i

		NR.LastNode.MessagePrecomputation = nil
		NR.LastNode.RecipientPrecomputation = nil
	}

	return &NR
}
