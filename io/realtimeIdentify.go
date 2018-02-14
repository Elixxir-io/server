package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/comms/mixserver/message"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
	"strconv"
)

// Blank struct for implementing services.BatchTransmission
type RealtimeIdentifyHandler struct{}

// TransmissionHandler for RealtimeIdentifyMessages
func (h RealtimeIdentifyHandler) Handler(
	roundId string, batchSize uint64, slots []*services.Slot) {
	jww.INFO.Println("Beginning RealtimeEncrypt Phase...")
	// Create the RealtimeEncryptMessage
	msg := &pb.RealtimeEncryptMessage{
		RoundID: roundId,
		Slots:   make([]*pb.RealtimeEncryptSlot, batchSize),
	}

	// Get round
	round := globals.GlobalRoundMap.GetRound(roundId)

	// Iterate over the input slots
	for i := range slots {
		out := (*slots[i]).(*realtime.SlotIdentify)
		// Convert to RealtimeEncryptSlot
		rId, _ := strconv.ParseUint(out.EncryptedRecipientID.Text(10), 10, 64)
		msgSlot := &pb.RealtimeEncryptSlot{
			Slot:             out.Slot,
			RecipientID:      rId,
			EncryptedMessage: round.LastNode.EncryptedMessage[i].Bytes(),
		}

		// Append the RealtimeEncryptSlot to the RealtimeEncryptMessage
		msg.Slots[i] = msgSlot
	}

	// Send the first RealtimeEncrypt Message
	jww.DEBUG.Printf("Sending RealtimeEncrypt Message to %v...", NextServer)
	message.SendRealtimeEncrypt(NextServer, msg)
}
