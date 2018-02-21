package io

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/privategrity/comms/mixmessages"
	"gitlab.com/privategrity/crypto/cyclic"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/services"
)

type ReceiveMessageHandler struct{}

// Serves as the batch queue
// TODO better batch logic, we should convert this to a queue or channel
var msgCounter uint64 = 0
var msgQueue = make([]*services.Slot, globals.BatchSize)

// Reception handler for ReceiveMessageFromClient
func (s ServerImpl) ReceiveMessageFromClient(msg *pb.CmixMessage) {
	jww.DEBUG.Printf("Received message from client: %v...", msg)

	// Verify message fields are within the global cyclic group
	recipientId := cyclic.NewIntFromBytes(msg.RecipientID)
	messagePayload := cyclic.NewIntFromBytes(msg.MessagePayload)
	if globals.Grp.Inside(recipientId) && globals.Grp.Inside(messagePayload) {
		// Convert message to a Slot
		inputMsg := services.Slot(&realtime.SlotDecryptOut{
			Slot:                 msgCounter,
			SenderID:             1,
			EncryptedMessage:     messagePayload,
			EncryptedRecipientID: recipientId,
		})
		// Append the message to the batch queue
		msgQueue[msgCounter] = &inputMsg
		msgCounter += 1
	} else {
		jww.ERROR.Printf("Received message is not in the group: MsgPayload %v RecipientID %v",
			messagePayload.Text(10), recipientId.Text(10))
	}

	// Once the batch is filled
	if msgCounter == globals.BatchSize {
		roundId := globals.GetNextWaitingRoundID()
		jww.DEBUG.Printf("Beginning round %s...", roundId)
		// Pass the batch queue into Realtime
		StartRealtime(msgQueue, roundId, globals.BatchSize)

		// Reset the batch queue
		msgCounter = 0
		msgQueue = make([]*services.Slot, globals.BatchSize)
		// Begin a new round and start precomputation
		BeginNewRound(Servers)
	}
}

// Begin Realtime with a new batch of slots
func StartRealtime(slots []*services.Slot, roundId string, batchSize uint64) {
	jww.INFO.Println("Beginning RealtimeDecrypt Phase...")
	kickoffDecryptHandler(roundId, batchSize, slots)
}
