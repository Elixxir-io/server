////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"testing"
	"gitlab.com/privategrity/crypto/id"
)

func TestPeel(t *testing.T) {
	// NOTE: Does not test correctness

	test := 6
	pass := 0

	bs := uint64(3)

	round := globals.NewRound(bs)

	rng := cyclic.NewRandom(cyclic.NewInt(5), cyclic.NewInt(1000))

	grp := cyclic.NewGroup(cyclic.NewInt(101), cyclic.NewInt(27), cyclic.NewInt(97), rng)

	recipientIds := [3]*id.UserID{
		id.NewUserIDFromUint(5, t),
		id.NewUserIDFromUint(7, t),
		id.NewUserIDFromUint(9, t),
	}

	var im []services.Slot

	im = append(im, &Slot{
		Slot:      uint64(0),
		CurrentID: recipientIds[0],
		Message:   cyclic.NewInt(int64(39))})

	im = append(im, &Slot{
		Slot:      uint64(1),
		CurrentID: recipientIds[1],
		Message:   cyclic.NewInt(int64(86))})

	im = append(im, &Slot{
		Slot:      uint64(2),
		CurrentID: recipientIds[2],
		Message:   cyclic.NewInt(int64(66))})

	// Set the keys
	round.LastNode.MessagePrecomputation = make([]*cyclic.Int, round.BatchSize)
	round.LastNode.MessagePrecomputation[0] = cyclic.NewInt(77)
	round.LastNode.MessagePrecomputation[1] = cyclic.NewInt(93)
	round.LastNode.MessagePrecomputation[2] = cyclic.NewInt(47)

	expected := [][]*cyclic.Int{
		{cyclic.NewInt(74)},
		{cyclic.NewInt(19)},
		{cyclic.NewInt(72)},
	}

	dc := services.DispatchCryptop(&grp, Peel{}, nil, nil, round)

	for i := uint64(0); i < bs; i++ {
		dc.InChannel <- &(im[i])
		rtn := <-dc.OutChannel

		result := expected[i]

		rtnXtc := (*rtn).(*Slot)

		// Test EncryptedMessage results
		for j := 0; j < 1; j++ {
			if result[j].Cmp(rtnXtc.Message) != 0 {
				t.Errorf("Test of RealtimePeel's EncryptedMessage output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtnXtc.Message.Text(10))
			} else {
				pass++
			}
		}

		// Test RecipientID pass through
		if recipientIds[i] != rtnXtc.CurrentID {
			t.Errorf("Test of RealtimePeel's RecipientID ouput failed on index %v.  Expected: %v Received: %v ",
				i, recipientIds[i], rtnXtc.CurrentID)
		} else {
			pass++
		}
	}

	println("Realtime Peel", pass, "out of", test, "tests passed.")

}
