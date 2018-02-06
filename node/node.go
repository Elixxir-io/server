// Package node contains the initialization and main loop of a cMix server.
package node

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"strconv"

	"gitlab.com/privategrity/comms/mixserver"
	"gitlab.com/privategrity/server/cryptops"
	"gitlab.com/privategrity/server/cryptops/precomputation"
	"gitlab.com/privategrity/server/cryptops/realtime"
	"gitlab.com/privategrity/server/globals"
	"gitlab.com/privategrity/server/io"
	"gitlab.com/privategrity/server/services"
)

// StartServer reads configuration options and starts the cMix server
func StartServer(serverIndex int) {
	viper.Debug()
	jww.INFO.Printf("Log Filename: %v\n", viper.GetString("logPath"))
	jww.INFO.Printf("Config Filename: %v\n\n", viper.ConfigFileUsed())

	// Get all servers
	servers := getServers()
	// Determine what the next server leap is
	nextIndex := serverIndex + 1
	if serverIndex == len(servers)-1 {
		nextIndex = 0
	}
	io.NextServer = "localhost:" + servers[nextIndex]
	localServer := "localhost:" + servers[serverIndex]

	// Start mix servers on localServer
	jww.INFO.Printf("Starting server on %v\n", localServer)
	// Initialize GlobalRoundMap
	globals.GlobalRoundMap = globals.NewRoundMap()
	// Kick off Comms server
	go mixserver.StartServer(localServer, io.ServerImpl{Rounds: &globals.GlobalRoundMap})
	// Kick off a Round TODO Better parameters
	NewRound("Test", 5)
}

// getServers pulls a string slice of server ports from the config file and
// verifies that the ports are valid.
func getServers() []string {
	servers := viper.GetStringSlice("servers")
	if servers == nil {
		jww.ERROR.Println("No servers listed in config file.")
	}
	for i := range servers {
		temp, err := strconv.Atoi(servers[i])
		// catch non-int ports
		if err != nil {
			jww.ERROR.Println("Non-integer server ports in config file")
		}
		// Catch invalid ports
		if temp > 65535 || temp < 0 {
			jww.ERROR.Printf("Port %v listed in the config file is not a "+
				"valid port\n", temp)
		}
		// Catch reserved ports
		if temp < 1024 {
			jww.WARN.Printf("Port %v is a reserved port, superuser privilege"+
				" may be required.\n", temp)
		}
	}
	return servers
}

// Kicks off a new round in CMIX
func NewRound(roundId string, batchSize uint64) {
	// Create a new Round
	round := globals.NewRound(batchSize)
	// Add round to the GlobalRoundMap
	globals.GlobalRoundMap.AddRound(roundId, round)

	// Create the controller for PrecompDecrypt
	precompDecryptController := services.DispatchCryptop(globals.Grp,
		precomputation.Decrypt{}, nil, nil, round)
	round.AddChannel(globals.PRECOMP_DECRYPT, precompDecryptController.InChannel)
	// Kick off PrecompDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompDecryptController.OutChannel, io.PrecompDecryptHandler{})

	// Create the dispatch controller for PrecompPermute
	precompPermuteController := services.DispatchCryptop(globals.Grp,
		precomputation.Permute{}, nil, nil, round)
	// Hook up the dispatcher's input to the round
	round.AddChannel(globals.PRECOMP_PERMUTE,
		precompPermuteController.InChannel)
	// Create the message reorganizer for PrecompPermute
	precompPermuteReorganizer := services.NewSlotReorganizer(
		precompPermuteController.OutChannel, nil, batchSize)
	// Kick off PrecompPermute Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		precompPermuteReorganizer.OutChannel,
		io.PrecompPermuteHandler{})

	// Create the controller for Keygen
	keygenController := services.DispatchCryptop(globals.Grp,
		cryptops.GenerateClientKey{}, nil, nil, round)

	// Create the controller for RealtimeDecrypt
	realtimeDecryptController := services.DispatchCryptop(globals.Grp,
		realtime.Decrypt{}, keygenController.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_DECRYPT, keygenController.InChannel)
	// Kick off RealtimeDecrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimeDecryptController.OutChannel, io.RealtimeDecryptHandler{})

	// Create the controller for RealtimeEncrypt
	realtimeEncryptController := services.DispatchCryptop(globals.Grp,
		realtime.Encrypt{}, keygenController.OutChannel, nil, round)
	// Add the InChannel from the keygen controller to round
	round.AddChannel(globals.REAL_ENCRYPT, keygenController.InChannel)
	// Kick off RealtimeEncrypt Transmission Handler
	services.BatchTransmissionDispatch(roundId, batchSize,
		realtimeEncryptController.OutChannel, io.RealtimeEncryptHandler{})
}
