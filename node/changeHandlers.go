////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

// ChangeHandlers contains the logic for every state within the state machine

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/internal/phase"
	"gitlab.com/elixxir/server/internal/round"
	"gitlab.com/elixxir/server/internal/state"
	"gitlab.com/elixxir/server/io"
	"gitlab.com/elixxir/server/permissioning"
	"strings"
	"sync"
	"time"
)

// Number of hard-coded users to create
var numDemoUsers = int(256)

func Dummy(from current.Activity) error {
	return nil
}

// NotStarted is the beginning state of state machine. Enters waiting upon successful completion
func NotStarted(instance *internal.Instance) error {
	// Start comms network
	ourDef := instance.GetDefinition()
	network := instance.GetNetwork()

	jww.INFO.Printf("Loading certificates from disk")
	// Get the Server and Gateway certificates from file, if they exist
	certsExist, serverCert, gwCert := getCertificates(ourDef.ServerCertPath,
		ourDef.GatewayCertPath)

	// If the certificates were retrieved from file, so do not need to register
	if certsExist {
		jww.INFO.Printf("Certificates found!")
	} else {
		jww.INFO.Printf("Certificates not found, registering with permissioning!")
		// Connect to the Permissioning Server without authentication
		permHost, err := network.AddHost(&id.Permissioning,
			// instance.GetPermissioningAddress,
			ourDef.Permissioning.Address,
			ourDef.Permissioning.TlsCert,
			true,
			false)
		if err != nil {
			return errors.Errorf("Unable to connect to registration server: %+v", err)
		}

		// Blocking call: begin Node registration
		err = permissioning.RegisterNode(ourDef, network, permHost)
		if err != nil {
			if strings.Contains(err.Error(), "Node with registration code") && strings.Contains(err.Error(), "has already been registered") {
				jww.WARN.Println("Node is already registered, continuing to retrieve the signed cert from the NDF. Attempting re-registration is NOT secure")
			} else {
				return errors.Errorf("Failed to register node: %+v", err)
			}

		}
		// Disconnect the old Permissioning server to enable authentication
		permHost.Disconnect()
	}

	// Connect to the Permissioning Server with authentication enabled
	// the server does not have a signed cert, but the pemrissionign has its cert,
	// reverse authetnication on conenctiosn just use the public key inside certs,
	// not the entire key chain, so even through the server does have a signed
	// cert, it can reverse auth with permissioning, allowing it to get the
	// full NDF
	// do this even if you have the certs to ensure the permissioning server is
	// ready for servers to connect to it
	permHost, err := network.AddHost(&id.Permissioning,
		ourDef.Permissioning.Address, ourDef.Permissioning.TlsCert, true, true)
	if err != nil {
		return errors.Errorf("Unable to connect to registration server: %+v", err)
	}

	// Retry polling until an ndf is returned
	err = errors.Errorf(ndf.NO_NDF)

	pollDelay := 1 * time.Second

	for err != nil && (strings.Contains(err.Error(), ndf.NO_NDF)) {
		time.After(pollDelay)

		var permResponse *mixmessages.PermissionPollResponse
		// Blocking call: Request ndf from permissioning
		permResponse, err = permissioning.PollPermissioning(permHost, instance, current.NOT_STARTED)
		if err == nil {
			//find certs in NDF if they are nto already had
			if !certsExist {
				serverCert, gwCert, err = permissioning.FindSelfInNdf(ourDef,
					instance.GetConsensus().GetFullNdf().Get())
				if err != nil {
					//if certs are not in NDF, redo the poll
					continue
				}
			}

			err = permissioning.UpdateNDf(permResponse, instance)
		}
	}

	// Check for unexpected errors (ie errors from polling other than NO_NDF)
	if err != nil {
		return errors.Errorf("Failed to get ndf: %+v", err)
	}

	// Do not need to get the Server and Gateway certificates if they were
	// already retrieved from file
	if ourDef.WriteToFile && !certsExist {
		jww.INFO.Printf("Writing certificates to disk")
		// Save the retrieved certificates to file
		writeCertificates(ourDef, serverCert, gwCert)
		jww.INFO.Printf("Certificates written to disk")
	} else if !certsExist && !ourDef.WriteToFile {
		jww.WARN.Printf("Not writing certs to file because no file found")
	}

	cmixGrp := instance.GetConsensus().GetCmixGroup()
	//populate the dummy precanned users
	jww.INFO.Printf("Adding dummy users to registry")
	userDatabase := instance.GetUserRegistry()
	PopulateDummyUsers(userDatabase, cmixGrp)

	//Add a dummy user for gateway
	dummy := userDatabase.NewUser(cmixGrp)
	dummy.ID = &id.DummyUser
	dummy.BaseKey = cmixGrp.NewIntFromBytes((*dummy.ID)[:])
	dummy.IsRegistered = true
	userDatabase.UpsertUser(dummy)

	jww.INFO.Printf("Waiting on communication from gateway to continue")
	// Atomically denote that gateway is ready for polling
	instance.SetGatewayAsReady()

	// Receive signal that indicates that gateway is ready for polling
	instance.GetGatewayFirstTime().Receive()

	jww.INFO.Printf("Communication form gateway recieved")
	jww.INFO.Printf("Restarting network interface to begin operation")
	// Restart the network with these signed certs
	err = instance.RestartNetwork(io.NewImplementation, serverCert, gwCert)
	if err != nil {
		return errors.Errorf("Unable to restart network with new certificates: %+v", err)
	}

	// HACK HACK HACK
	// FIXME: we should not be coupling connections and server objects
	// Technically the servers can fail to bind for up to
	// a couple minutes (depending on operating system), but
	// in practice 10 seconds works
	time.Sleep(10 * time.Second)

	// Once done with notStarted transition into waiting
	go func() {
		// Ensure that instance is in not started prior to transition
		cur, err := instance.GetStateMachine().WaitFor(1*time.Second, current.NOT_STARTED)
		if cur != current.NOT_STARTED || err != nil {
			roundErr := errors.Errorf("Server never transitioned to %v state: %+v", current.NOT_STARTED, err)
			instance.ReportRoundFailure(roundErr, instance.GetID(), nil)
		}

		// if error passed in go to error
		if instance.GetRecoveredError() != nil {
			ok, err := instance.GetStateMachine().Update(current.ERROR)
			if !ok || err != nil {
				roundErr := errors.Errorf("Unable to transition to %v state: %+v", current.ERROR, err)
				instance.ReportRoundFailure(roundErr, instance.GetID(), nil)
			}
		} else {
			// Transition state machine into waiting state
			ok, err := instance.GetStateMachine().Update(current.WAITING)
			if !ok || err != nil {
				roundErr := errors.Errorf("Unable to transition to %v state: %+v", current.WAITING, err)
				instance.ReportRoundFailure(roundErr, instance.GetID(), nil)
			}
		}

		// Periodically re-poll permissioning
		// fixme we need to review the performance implications and possibly make this programmable
		ticker := time.NewTicker(50 * time.Millisecond)
		for range ticker.C {
			err := permissioning.Poll(instance)
			if err != nil {
				// If we receive an error polling here, panic this thread
				roundErr := errors.Errorf("Received error polling for permisioning: %+v", err)
				instance.ReportRoundFailure(roundErr, instance.GetID(), nil)
			}
		}
	}()

	return nil
}

func Waiting(from current.Activity) error {
	// start waiting process
	return nil
}

// Precomputing does various business logic to prep for the start of a new round
func Precomputing(instance *internal.Instance, newRoundTimeout time.Duration) error {
	// Add round.queue to instance, get that here and use it to get new round
	// start pre-precomputation
	roundInfo, err := instance.GetCreateRoundQueue().Receive()
	if err != nil {
		jww.TRACE.Printf("Error with create round queue: %+v", err)
	}

	roundID := roundInfo.GetRoundId()
	topology := roundInfo.GetTopology()
	// Extract topology from RoundInfo
	nodeIDs, err := id.NewIDListFromBytes(topology)
	if err != nil {
		return errors.Errorf("Unable to convert topology into a node list: %+v", err)
	}

	// fixme: this panics on error, external comm should not be able to crash server
	circuit := connect.NewCircuit(nodeIDs)

	for i := 0; i < circuit.Len(); i++ {
		nodeId := circuit.GetNodeAtIndex(i)
		ourHost, ok := instance.GetNetwork().GetHost(nodeId)
		if !ok {
			return errors.Errorf("Host not available for node %s in round", circuit.GetNodeAtIndex(i))
		}
		circuit.AddHost(ourHost)
	}

	//Build the components of the round
	phases, phaseResponses := NewRoundComponents(
		instance.GetGraphGenerator(),
		circuit,
		instance.GetID(),
		instance,
		roundInfo.GetBatchSize(),
		newRoundTimeout, instance.GetStreamPool(),
		instance.GetDisableStreaming())

	var override = func() {
		phaseOverrides := instance.GetPhaseOverrides()
		for toOverride, or := range phaseOverrides {
			phases[toOverride] = or
		}
	}
	if instance.GetOverrideRound() != -1 {
		if instance.GetOverrideRound() == int(roundID) {
			override()
		}
	} else {
		override()
	}

	//Build the round
	rnd, err := round.New(
		instance.GetConsensus().GetCmixGroup(),
		instance.GetUserRegistry(),
		roundID, phases, phaseResponses,
		circuit,
		instance.GetID(),
		roundInfo.GetBatchSize(),
		instance.GetRngStreamGen(),
		instance.GetStreamPool(),
		instance.GetIP(),
		GetDefaultPanicHanlder(instance, &roundID))
	if err != nil {
		return errors.WithMessage(err, "Failed to create new round")
	}

	//Add the round to the manager
	instance.GetRoundManager().AddRound(rnd)
	jww.INFO.Printf("[%+v]: RID %d CreateNewRound COMPLETE", instance,
		roundID)

	if circuit.IsFirstNode(instance.GetID()) {
		err := StartLocalPrecomp(instance, roundID)
		if err != nil {
			return errors.WithMessage(err, "Failed to TransmitCreateNewRound")
		}
	}

	return nil
}

func Standby(from current.Activity) error {
	// start standby process
	return nil

}

// Realtime checks if we are in the correct phase
func Realtime(instance *internal.Instance) error {
	// Get new realtime round info from queue
	roundInfo, err := instance.GetRealtimeRoundQueue().Receive()
	if err != nil {
		return errors.Errorf("Unable to receive from RealtimeRoundQueue: %+v", err)
	}

	// Get our round
	ourRound, err := instance.GetRoundManager().GetRound(roundInfo.GetRoundId())
	if err != nil {
		return errors.Errorf("Unable to get round from round info: %+v", err)
	}

	// Check for correct phase in round
	if ourRound.GetCurrentPhase().GetType() != phase.RealDecrypt {
		return errors.Errorf("Not in correct phase. Expected phase: %+v. "+
			"Current phase: %+v", phase.RealDecrypt, ourRound.GetCurrentPhase())
	}

	if ourRound.GetTopology().IsFirstNode(instance.GetID()) {
		err = instance.GetRequestNewBatchQueue().Send(roundInfo)
		if err != nil {
			return errors.Errorf("Unable to send to RequestNewBatch queue: %+v", err)
		}
	}

	return nil
}

func Completed(from current.Activity) error {
	// start completed
	return nil
}

func Error(instance *internal.Instance) error {
	//If the error state was recovered from a restart, exit.
	if instance.GetRecoveredError() != nil {
		return nil
	}

	// Check for error message on server instance
	msg := instance.GetRoundError()
	if msg == nil {
		jww.FATAL.Panic("No error found on instance")
	}

	nid, err := id.Unmarshal(msg.NodeId)
	if err != nil {
		return errors.WithMessage(err, "Failed to get node id from error")
	}

	wg := sync.WaitGroup{}
	// If the error originated with us, send broadcast to other nodes
	if nid.Cmp(instance.GetID()) {
		r, err := instance.GetRoundManager().GetRound(id.Round(msg.Id))
		if err != nil {
			return errors.WithMessage(err, "Failed to get round id")
		}
		top := r.GetTopology()
		for i := 0; i < top.Len(); i++ {
			n := top.GetNodeAtIndex(i)
			wg.Add(1)
			go func() {
				// Don't need to send back to self
				if !instance.GetID().Cmp(n) {
					h, ok := instance.GetNetwork().GetHost(n)
					if !ok {
						jww.ERROR.Printf("Could not get host for node %s", n.String())
					}

					_, err := instance.SendRoundError(h, msg)
					if err != nil {
						err := errors.WithMessagef(err, "Failed to send error to node %s", n.String())
						jww.ERROR.Printf(err.Error())
					}
				}
				wg.Done()
			}()
		}
	}

	// Wait until the error messages are sent, or timeout after 3 minutes
	timeoutCh := make(chan struct{})
	timeout := time.NewTimer(3 * time.Minute)

	go func() {
		wg.Wait()
		timeoutCh <- struct{}{}
	}()

	select {
	case <-timeoutCh:
	case <-timeout.C:
	}

	b, err := proto.Marshal(msg)
	if err != nil {
		return errors.WithMessage(err, "Failed to marshal message into bytes")
	}

	err = utils.WriteFile(instance.GetRecoveredErrorFilePath(), b, 0644, 0644)
	if err != nil {
		return errors.WithMessage(err, "Failed to write error to file")
	}

	err = instance.GetResourceQueue().Kill(5 * time.Second)
	if err != nil {
		return errors.WithMessage(err, "Resource queue kill timed out")
	}

	instance.GetPanicWrapper()(fmt.Sprintf("Error encountered - closing server & writing error to file %s",
		instance.GetRecoveredErrorFilePath()))
	return nil
}

func Crash(from current.Activity) error {
	// start error
	return nil
}

// NewStateChanges creates a state table with dummy functions
func NewStateChanges() [current.NUM_STATES]state.Change {
	// Create the state change function table
	var stateChanges [current.NUM_STATES]state.Change

	stateChanges[current.NOT_STARTED] = Dummy
	stateChanges[current.WAITING] = Dummy
	stateChanges[current.PRECOMPUTING] = Dummy
	stateChanges[current.STANDBY] = Dummy
	stateChanges[current.REALTIME] = Dummy
	stateChanges[current.COMPLETED] = Dummy
	stateChanges[current.ERROR] = Dummy
	stateChanges[current.CRASH] = Dummy

	return stateChanges
}

// getCertificates retrieves the Server and Gateway certificates from the path
// in the definition. If one ore more certificate is not found, then that
// certificate is returned as an empty string and the function returns false.
func getCertificates(serverPath, gatewayPath string) (bool, string, string) {
	var serverCert, gatewayCert []byte
	var err error

	// Check if the Server certificate files exist
	serverCertExists := utils.FileExists(serverPath)

	if serverCertExists {
		// Load the Server certificate from file
		serverCert, err = utils.ReadFile(serverPath)
		if err != nil {
			jww.DEBUG.Printf("Received Server's certificate from file %s: %v",
				serverPath, err)
		}
	}

	// Check if the Gateway certificate files exist
	gatewayCertExists := utils.FileExists(gatewayPath)

	if gatewayCertExists {
		// Load the Gateway certificate from file
		gatewayCert, err = utils.ReadFile(gatewayPath)
		if err != nil {
			jww.DEBUG.Printf("Received Gateway's certificate from file: %s: %v",
				gatewayPath, err)
		}
	}

	return serverCertExists && gatewayCertExists,
		string(serverCert),
		string(gatewayCert)
}

// writeCertificates writes the Server and Gateway certificates to the paths
// in the definition. If either file fails to save, then it panics.
func writeCertificates(def *internal.Definition, serverCert, gatewayCert string) {

	// Write the Server certificate to specified path
	err := utils.WriteFile(def.ServerCertPath, []byte(serverCert),
		utils.FilePerms, utils.DirPerms)
	if err != nil {
		jww.FATAL.Panicf("Error writing Server certificate to path "+
			"%s: %v", def.ServerCertPath, err)
	}

	// Write the Gateway certificate to specified path
	err = utils.WriteFile(def.GatewayCertPath, []byte(gatewayCert),
		utils.FilePerms, utils.DirPerms)
	if err != nil {
		jww.FATAL.Panicf("Error writing Gateway certificate to path "+
			"%s: %v", def.GatewayCertPath, err)
	}
}

// Create dummy users to be manually inserted into the database
func PopulateDummyUsers(ur globals.UserRegistry, grp *cyclic.Group) {
	// Deterministically create named users for demo
	for i := 1; i < numDemoUsers; i++ {
		u := ur.NewUser(grp)
		u.IsRegistered = true
		ur.UpsertUser(u)
	}
	return
}
