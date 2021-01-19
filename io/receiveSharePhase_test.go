///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////
package io

import (
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/testUtil"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/large"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"testing"
)

func TestStartSharePhase(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	ri := &mixmessages.RoundInfo{ID: uint64(roundID)}
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	err = signature.Sign(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	err = ReceiveStartSharePhase(ri, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

// Happy path (no final key logic in this test)
func TestSharePhaseRound(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	// Fill with extra ID to avoid final Key generation codepath for this test
	mockId := id.NewIdFromBytes([]byte("test"), t)
	t.Logf("mockID: %v", mockId)
	topology := connect.NewCircuit([]*id.ID{instance.GetID(), mockId})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	mockHost, _ := connect.NewHost(mockId, nodeAddr, cert, connect.GetDefaultHostParams())

	topology.AddHost(nodeHost)
	topology.AddHost(mockHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	ri := &mixmessages.RoundInfo{ID: uint64(roundID)}
	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	// Get the previous node host for proper auth validation
	testHost := topology.GetHostAtIndex(1)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	instance.GetPhaseShareMachine().Update(state.STARTED)

	err = signature.Sign(ri, instance.GetPrivKey())
	if err != nil {
		t.Errorf("couldn't sign info message: %+v", err)
	}

	// Generate a share to send
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}
	// Manually fudge participant list so
	// our instance is not in the list
	piece.Participants = [][]byte{mockId.Bytes()}

	err = ReceiveSharePhasePiece(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

// Test implicit call ReceiveFinalKey through
// a ReceiveSharePhasePiece call
func TestSharePhaseRound_FinalKey(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	mockPhaseShare := testUtil.InitMockPhase(t)
	mockPhaseShare.Ptype = phase.PrecompShare

	tagKey := mockPhaseShare.GetType().String()
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhaseShare.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhaseShare.GetType()},
	)

	mockPhaseDecrypt := testUtil.InitMockPhase(t)
	mockPhaseDecrypt.Ptype = phase.PrecompDecrypt

	tagKey = mockPhaseDecrypt.GetType().String()
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhaseDecrypt.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhaseDecrypt.GetType()},
	)

	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})
	responseMap[phase.PrecompShare.String()+"Verification"] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase, mockPhaseDecrypt},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	params := connect.GetDefaultHostParams()
	params.MaxRetries = 0

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	// Generate a mock message
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}

	ok, err := instance.GetPhaseShareMachine().Update(state.STARTED)
	if !ok || err != nil {
		t.Errorf("Trouble updating phase machine for test: %v", err)
	}

	err = ReceiveSharePhasePiece(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	// Check that the key has been modified in the round
	expectedKey := grp.NewIntFromBytes(piece.Piece)
	receivedKey := rnd.GetBuffer().CypherPublicKey
	if expectedKey.Cmp(receivedKey) != 0 {
		t.Errorf("Final key did not match expected."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedKey.Bytes(), receivedKey.Bytes())
	}
}

// Unit test
func TestReceiveFinalKey(t *testing.T) {
	roundID := id.Round(0)
	grp := cyclic.NewGroup(large.NewIntFromString(primeString, 16),
		large.NewInt(2))

	instance, nodeAddr := mockInstance(t, mockSharePhaseImpl)
	topology := connect.NewCircuit([]*id.ID{instance.GetID()})

	cert, _ := utils.ReadFile(testkeys.GetNodeCertPath())
	nodeHost, _ := connect.NewHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	topology.AddHost(nodeHost)
	_, err := instance.GetNetwork().AddHost(instance.GetID(), nodeAddr, cert, connect.GetDefaultHostParams())
	if err != nil {
		t.Errorf("Failed to add host to instance: %v", err)
	}

	// Build responses for checks and for phase transition
	mockPhase := testUtil.InitMockPhase(t)
	responseMap := make(phase.ResponseMap)
	mockPhaseShare := testUtil.InitMockPhase(t)
	mockPhaseShare.Ptype = phase.PrecompShare

	tagKey := mockPhaseShare.GetType().String()
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhaseShare.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhaseShare.GetType()},
	)

	mockPhaseDecrypt := testUtil.InitMockPhase(t)
	mockPhaseDecrypt.Ptype = phase.PrecompDecrypt

	tagKey = mockPhaseDecrypt.GetType().String()
	responseMap[tagKey] = phase.NewResponse(
		phase.ResponseDefinition{
			PhaseAtSource:  mockPhaseDecrypt.GetType(),
			ExpectedStates: []phase.State{phase.Active},
			PhaseToExecute: mockPhaseDecrypt.GetType()},
	)

	responseMap[phase.PrecompShare.String()] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})
	responseMap[phase.PrecompShare.String()+"Verification"] =
		phase.NewResponse(phase.ResponseDefinition{mockPhase.GetType(),
			[]phase.State{phase.Active}, mockPhase.GetType()})

	rnd, err := round.New(grp, &globals.UserMap{}, roundID, []phase.Phase{mockPhase, mockPhaseDecrypt},
		responseMap, topology, topology.GetNodeAtIndex(0), 3,
		instance.GetRngStreamGen(), nil, "0.0.0.0", nil, nil)
	if err != nil {
		t.Errorf("Failed to create new round: %+v", err)
	}
	instance.GetRoundManager().AddRound(rnd)

	testHost := topology.GetHostAtIndex(0)
	auth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}

	// Generate a mock message
	msg := &pb.SharePiece{
		Piece:        grp.GetG().Bytes(),
		Participants: make([][]byte, 0),
		RoundID:      uint64(rnd.GetID()),
	}
	piece, err := generateShare(msg, grp, rnd, instance)
	if err != nil {
		t.Errorf("Could not generate a mock share: %v", err)
	}

	ok, err := instance.GetPhaseShareMachine().Update(state.STARTED)
	if !ok || err != nil {
		t.Errorf("Trouble updating phase machine for test: %v", err)
	}

	err = ReceiveFinalKey(piece, auth, instance)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	// Check that the key has been modified in the round
	expectedKey := grp.NewIntFromBytes(piece.Piece)
	receivedKey := rnd.GetBuffer().CypherPublicKey
	if expectedKey.Cmp(receivedKey) != 0 {
		t.Errorf("Final key did not match expected."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedKey.Bytes(), receivedKey.Bytes())
	}

}

func mockSharePhaseImpl(instance *internal.Instance) *node.Implementation {
	impl := node.NewImplementation()
	impl.Functions.SharePhaseRound = func(sharedPiece *mixmessages.SharePiece,
		auth *connect.Auth) error {
		return nil
	}
	impl.Functions.StartSharePhase = func(ri *mixmessages.RoundInfo, auth *connect.Auth) error {
		return nil
	}
	impl.Functions.ShareFinalKey = func(sharedPiece *mixmessages.SharePiece, auth *connect.Auth) error {
		return ReceiveFinalKey(sharedPiece, auth, instance)
	}

	return impl
}
