////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package realtime

import (
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/services"
	"testing"
)

func TestEncrypt(t *testing.T) {
	// NOTE: Does not test correctness

	test := 6
	pass := 0

	grp := cyclic.NewGroup(large.NewInt(107), large.NewInt(23),
		large.NewInt(27))

	batchSize := uint64(3)

	round := globals.NewRound(batchSize, grp)

	recipientIds := [3]*id.User{
		id.NewUserFromUint(5, t),
		id.NewUserFromUint(7, t),
		id.NewUserFromUint(9, t),
	}

	associatedDatas := [3]*cyclic.Int{
		grp.NewInt(int64(42)),
		grp.NewInt(int64(84)),
		grp.NewInt(int64(106)),
	}

	var im []services.Slot

	im = append(im, &Slot{
		Slot:           uint64(0),
		CurrentID:      recipientIds[0],
		AssociatedData: associatedDatas[0],
		Message:        grp.NewInt(int64(39)),
		CurrentKey:     grp.NewInt(int64(65))})

	im = append(im, &Slot{
		Slot:           uint64(1),
		CurrentID:      recipientIds[1],
		AssociatedData: associatedDatas[1],
		Message:        grp.NewInt(int64(86)),
		CurrentKey:     grp.NewInt(int64(44))})

	im = append(im, &Slot{
		Slot:           uint64(2),
		CurrentID:      recipientIds[2],
		AssociatedData: associatedDatas[2],
		Message:        grp.NewInt(int64(66)),
		CurrentKey:     grp.NewInt(int64(94))})

	// Set the keys
	round.T[0] = grp.NewInt(52)
	round.T[1] = grp.NewInt(68)
	round.T[2] = grp.NewInt(11)

	expected := [][]*cyclic.Int{
		{grp.NewInt(103)},
		{grp.NewInt(84)},
		{grp.NewInt(85)},
	}

	dc := services.DispatchCryptop(grp, Encrypt{}, nil, nil, round)

	for i := uint64(0); i < batchSize; i++ {
		dc.InChannel <- &(im[i])
		rtn := <-dc.OutChannel

		result := expected[i]

		rtnXtc := (*rtn).(*Slot)

		// Test EncryptedMessage results
		for j := 0; j < 1; j++ {
			if result[j].Cmp(rtnXtc.Message) != 0 {
				t.Errorf("Test of RealtimeEncrypt's EncryptedMessage output "+
					"failed on index: %v on value: %v.  Expected: %v Received: %v ",
					i, j, result[j].Text(10), rtnXtc.Message.Text(10))
			} else {
				pass++
			}
		}

		// Test CurrentID pass through
		if recipientIds[i] != rtnXtc.CurrentID {
			t.Errorf("Test of RealtimeEncrypt's AssociatedData ouput failed on index %v.  Expected: %v Received: %v ",
				i, recipientIds[i], rtnXtc.CurrentID)
		} else {
			pass++
		}

		// Test AssociatedData pass through
		if associatedDatas[i].Cmp(rtnXtc.AssociatedData) != 0 {
			t.Errorf("Test of RealtimeEncrypt's AssociatedData ouput failed on index %v.  Expected: %v Received: %v ",
				i, associatedDatas[i], rtnXtc.AssociatedData)
		} else {
			pass++
		}

	}

	println("Realtime Encrypt", pass, "out of", test, "tests passed.")

}
