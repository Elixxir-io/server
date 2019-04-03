////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package precomputation

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

// Not sure if the input data represents real data accurately
// Expected data was generated by the cryptop.
// Right now this tests for regression, not correctness.
func TestPrecompDecrypt(t *testing.T) {
	grp := cyclic.NewGroup(large.NewInt(117), large.NewInt(7), large.NewInt(5))

	globals.Clear(t)
	globals.SetGroup(&grp)

	batchSize := uint64(3)
	round := globals.NewRound(batchSize, &grp)

	round.CypherPublicKey = grp.NewInt(13)

	var im []services.Slot

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(0),
		MessageCypher:                grp.NewInt(12),
		AssociatedDataCypher:         grp.NewInt(7),
		MessagePrecomputation:        grp.NewInt(3),
		AssociatedDataPrecomputation: grp.NewInt(8),
	})

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(1),
		MessageCypher:                grp.NewInt(2),
		AssociatedDataCypher:         grp.NewInt(4),
		MessagePrecomputation:        grp.NewInt(22),
		AssociatedDataPrecomputation: grp.NewInt(16),
	})

	im = append(im, &PrecomputationSlot{
		Slot:                         uint64(2),
		MessageCypher:                grp.NewInt(14),
		AssociatedDataCypher:         grp.NewInt(99),
		MessagePrecomputation:        grp.NewInt(96),
		AssociatedDataPrecomputation: grp.NewInt(5),
	})

	round.R_INV[0] = grp.NewInt(5)
	round.U_INV[0] = grp.NewInt(9)
	round.Y_R[0] = grp.NewInt(15)
	round.Y_U[0] = grp.NewInt(2)

	round.R_INV[1] = grp.NewInt(8)
	round.U_INV[1] = grp.NewInt(1)
	round.Y_R[1] = grp.NewInt(13)
	round.Y_U[1] = grp.NewInt(6)

	round.R_INV[2] = grp.NewInt(38)
	round.U_INV[2] = grp.NewInt(100)
	round.Y_R[2] = grp.NewInt(44)
	round.Y_U[2] = grp.NewInt(32)

	expected := [][]*cyclic.Int{{
		grp.NewInt(105), grp.NewInt(45),
		grp.NewInt(39), grp.NewInt(65),
	}, {
		grp.NewInt(112), grp.NewInt(22),
		grp.NewInt(52), grp.NewInt(52),
	}, {
		grp.NewInt(49), grp.NewInt(99),
		grp.NewInt(78), grp.NewInt(26),
	}}

	dispatch := services.DispatchCryptop(
		&grp, Decrypt{}, nil, nil, round)

	for i := 0; i < len(im); i++ {

		dispatch.InChannel <- &(im[i])
		actual := <-dispatch.OutChannel

		act := (*actual).(*PrecomputationSlot)

		expectedVal := expected[i]

		if act.MessageCypher.Cmp(expectedVal[0]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Message"+
				"Cypher Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expectedVal[0].Text(10), act.MessageCypher.Text(10))
		}

		if act.AssociatedDataCypher.Cmp(expectedVal[1]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Recipient"+
				"Keys Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expectedVal[1].Text(10), act.AssociatedDataCypher.Text(10))
		}

		if act.MessagePrecomputation.Cmp(expectedVal[2]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Message"+
				"Cypher Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expectedVal[2].Text(10), act.MessagePrecomputation.Text(10))
		}

		if act.AssociatedDataPrecomputation.Cmp(expectedVal[3]) != 0 {
			t.Errorf("Test of Precomputation Decrypt's cryptop failed Recipient"+
				"Cypher Test on index: %v\n\tExpected: %#v\n\tActual:   %#v", i,
				expectedVal[3].Text(10), act.AssociatedDataPrecomputation.Text(10))
		}
	}
}
