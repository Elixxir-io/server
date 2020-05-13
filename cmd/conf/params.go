////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package conf

import (
	gorsa "crypto/rsa"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/crypto/xx"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/id/idf"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/server/internal"
	"gitlab.com/elixxir/server/services"
	"net"
	"runtime"
	"time"
)

// This object is used by the server instance.
// It should be constructed using a viper object
type Params struct {
	Index            int
	SkipReg          bool `yaml:"skipReg"`
	Verbose          bool
	KeepBuffers      bool
	UseGPU           bool
	DisableStreaming bool
	Groups           Groups
	RngScalingFactor uint `yaml:"rngScalingFactor"`
	GWConnTimeout    time.Duration
	ServerCertPath   string
	GatewayCertPath  string
	SignedCertPath   string

	Node          Node
	Database      Database
	Gateways      Gateways
	Permissioning Permissioning
	Metrics       Metrics
	GraphGen      GraphGen
}

// NewParams gets elements of the viper object
// and updates the params object. It returns params
// unless it fails to parse in which it case returns error
func NewParams(vip *viper.Viper) (*Params, error) {

	params := Params{}

	params.Index = vip.GetInt("index")

	params.Node.Paths.Idf = vip.GetString("node.paths.Idf")
	params.Node.Paths.Cert = vip.GetString("node.paths.cert")
	params.Node.Paths.Key = vip.GetString("node.paths.key")
	params.Node.Paths.Log = vip.GetString("node.paths.log")
	params.Node.Addresses = vip.GetStringSlice("node.addresses")

	params.Database.Name = vip.GetString("database.name")
	params.Database.Username = vip.GetString("database.username")
	params.Database.Password = vip.GetString("database.password")
	params.Database.Addresses = vip.GetStringSlice("database.addresses")

	params.Gateways.Paths.Cert = vip.GetString("gateways.paths.cert")
	params.Gateways.Addresses = vip.GetStringSlice("gateways.addresses")

	params.Permissioning.Paths.Cert = vip.GetString("permissioning.paths.cert")
	params.Permissioning.Address = vip.GetString("permissioning.address")
	params.Permissioning.RegistrationCode = vip.GetString("permissioning.registrationCode")

	params.ServerCertPath = vip.GetString("node.paths.cert")
	params.GatewayCertPath = vip.GetString("gateways.paths.cert")

	params.GraphGen.defaultNumTh = uint8(vip.GetUint("graphgen.defaultNumTh"))
	if params.GraphGen.defaultNumTh == 0 {
		params.GraphGen.defaultNumTh = uint8(runtime.NumCPU())
	}
	params.GraphGen.minInputSize = vip.GetUint32("graphgen.mininputsize")
	if params.GraphGen.minInputSize == 0 {
		params.GraphGen.minInputSize = 4
	}
	params.GraphGen.outputSize = vip.GetUint32("graphgen.outputsize")
	if params.GraphGen.outputSize == 0 {
		params.GraphGen.outputSize = 4
	}
	// This (outputThreshold) already defaulted to 0.0
	params.GraphGen.outputThreshold = float32(vip.GetFloat64("graphgen.outputthreshold"))

	params.SkipReg = vip.GetBool("skipReg")
	params.Verbose = vip.GetBool("verbose")
	params.KeepBuffers = vip.GetBool("keepBuffers")
	params.UseGPU = vip.GetBool("useGpu")
	params.RngScalingFactor = vip.GetUint("rngScalingFactor")

	params.SignedCertPath = vip.GetString("signedCertPath")

	// If RngScalingFactor is not set, then set default value
	if params.RngScalingFactor == 0 {
		params.RngScalingFactor = 10000
	}

	gwTimeoutMs := vip.GetUint64("GatewayConnectionTimeout")
	if gwTimeoutMs == 0 {
		params.GWConnTimeout = 289 * 365 * 24 * time.Hour
	} else {
		params.GWConnTimeout = time.Duration(gwTimeoutMs) * time.Millisecond
	}

	params.Groups.CMix = vip.GetStringMapString("groups.cmix")
	params.Groups.E2E = vip.GetStringMapString("groups.e2e")

	params.Metrics.Log = vip.GetString("metrics.log")

	return &params, nil
}

// Create a new Definition object from the Params object
func (p *Params) ConvertToDefinition() (*internal.Definition, error) {

	def := &internal.Definition{}

	def.Flags.KeepBuffers = p.KeepBuffers
	def.Flags.SkipReg = p.SkipReg
	def.Flags.Verbose = p.Verbose
	def.Flags.UseGPU = p.UseGPU
	def.GwConnTimeout = p.GWConnTimeout

	var tlsCert, tlsKey []byte
	var err error

	if p.Node.Paths.Cert != "" {
		tlsCert, err = utils.ReadFile(p.Node.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Cert: %+v", err)
		}
	}

	if p.Node.Paths.Key != "" {
		tlsKey, err = utils.ReadFile(p.Node.Paths.Key)

		if err != nil {
			jww.FATAL.Panicf("Could not load TLS Key: %+v", err)
		}
	}

	_, port, err := net.SplitHostPort(p.Node.Addresses[p.Index])
	if err != nil {
		jww.FATAL.Panicf("Unable to obtain port from address: %+v",
			errors.New(err.Error()))
	}
	def.Address = fmt.Sprintf("0.0.0.0:%s", port)
	def.TlsCert = tlsCert
	def.TlsKey = tlsKey
	def.LogPath = p.Node.Paths.Log
	def.MetricLogPath = p.Metrics.Log

	// Only def values if params is set
	if p.SignedCertPath != "" {
		def.WriteToFile = true
		def.ServerCertPath = p.SignedCertPath
		def.GatewayCertPath = p.GatewayCertPath + "-definition"
	}

	def.Gateway.Address = p.Gateways.Addresses[p.Index]
	var GwTlsCerts []byte

	if p.Gateways.Paths.Cert != "" {
		GwTlsCerts, err = utils.ReadFile(p.Gateways.Paths.Cert)
		if err != nil {
			jww.FATAL.Panicf("Could not load gateway TLS Cert: %+v", err)
		}
	}

	def.Gateway.TlsCert = GwTlsCerts

	var PermTlsCert []byte

	if p.Permissioning.Paths.Cert != "" {
		PermTlsCert, err = utils.ReadFile(p.Permissioning.Paths.Cert)

		if err != nil {
			jww.FATAL.Panicf("Could not load permissioning TLS Cert: %+v", err)
		}
	}

	def.DisableStreaming = p.DisableStreaming

	//Set the node's private/public key
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey

	if p.Node.Paths.Cert != "" || p.Node.Paths.Key != "" {
		// Get the node's TLS cert
		tlsCertPEM, err := utils.ReadFile(p.Node.Paths.Cert)
		if err != nil {
			jww.FATAL.Panicf("Could not read tls cert file: %v", err)
		}

		//Get the RSA key out of the TLS cert
		tlsCert, err := tls.LoadCertificate(string(tlsCertPEM))
		if err != nil {
			jww.FATAL.Panicf("Could not decode tls cert file into a"+
				" tls cert: %v", err)
		}

		publicKey = &rsa.PublicKey{PublicKey: *tlsCert.PublicKey.(*gorsa.PublicKey)}

		// Get the node's TLS Key
		tlsKeyPEM, err := utils.ReadFile(p.Node.Paths.Key)
		if err != nil {
			jww.FATAL.Panicf("Could not read tls key file: %v", err)
		}

		privateKey, err = rsa.LoadPrivateKeyFromPem(tlsKeyPEM)
		if err != nil {
			jww.FATAL.Panicf("Could not decode tls key from file: %+v",
				err)
		}
	}

	def.PublicKey = publicKey
	def.PrivateKey = privateKey

	// Check if the IDF exists
	if p.Node.Paths.Idf != "" && utils.Exists(p.Node.Paths.Idf) {
		// If the IDF exists, then get the ID and save it
		_, newID, err2 := idf.UnloadIDF(p.Node.Paths.Idf)
		if err2 != nil {
			return nil, errors.Errorf("Could not unload IDF: %+v", err2)
		}

		def.ID = newID
	} else {
		// If the IDF does not exist, then generate a new ID, save it to an IDF,
		// and save the ID to the definition

		// Generate a random 256-bit number for the salt
		salt := cmix.NewSalt(csprng.NewSystemRNG(), 32)

		// Generate new ID
		newID, err2 := xx.NewID(def.PublicKey, salt[:32], id.Node)
		if err2 != nil {
			return nil, errors.Errorf("Failed to create new ID: %+v", err2)
		}

		// Save new ID to file
		err2 = idf.LoadIDF(p.Node.Paths.Idf, salt, newID)
		if err2 != nil {
			return nil, errors.Errorf("Failed to save new ID to file: %+v",
				err2)
		}

		def.ID = newID
	}

	def.Gateway.ID = def.ID.DeepCopy()
	def.Gateway.ID.SetType(id.Gateway)

	def.Permissioning.TlsCert = PermTlsCert
	def.Permissioning.Address = p.Permissioning.Address
	def.Permissioning.RegistrationCode = p.Permissioning.RegistrationCode
	if len(def.Permissioning.TlsCert) > 0 {
		permCert, err := tls.LoadCertificate(string(def.Permissioning.TlsCert))
		if err != nil {
			jww.FATAL.Panicf("Could not decode permissioning tls cert file "+
				"into a tls cert: %v", err)
		}

		def.Permissioning.PublicKey = &rsa.PublicKey{PublicKey: *permCert.PublicKey.(*gorsa.PublicKey)}
	}

	//
	ourNdf := createNdf(def, p)
	def.FullNDF = ourNdf
	def.PartialNDF = ourNdf

	PanicHandler := func(g, m string, err error) {
		jww.FATAL.Panicf(fmt.Sprintf("Error in module %s of graph %s: %+v", g,
			m, err))
	}

	def.GraphGenerator = services.NewGraphGenerator(p.GraphGen.minInputSize, PanicHandler,
		p.GraphGen.defaultNumTh, p.GraphGen.outputSize, p.GraphGen.outputThreshold)

	return def, nil
}

// createNdf is a helper function which builds a network ndf based off of the
//  server.Definition
func createNdf(def *internal.Definition, params *Params) *ndf.NetworkDefinition {
	// Build our node
	ourNode := ndf.Node{
		ID:             def.ID.Marshal(),
		Address:        def.Address,
		TlsCertificate: string(def.TlsCert),
	}

	// Build our gateway
	ourGateway := ndf.Gateway{
		ID:             def.Gateway.ID.Marshal(),
		Address:        def.Gateway.Address,
		TlsCertificate: string(def.Gateway.TlsCert),
	}

	// Build the perm server
	ourPerm := ndf.Registration{
		Address:        def.Permissioning.Address,
		TlsCertificate: string(def.Permissioning.TlsCert),
	}

	// Build the group
	cmixGrp := toNdfGroup(params.Groups.CMix)
	e2eGrp := toNdfGroup(params.Groups.E2E)

	networkDef := &ndf.NetworkDefinition{
		Timestamp:    time.Time{},
		Gateways:     []ndf.Gateway{ourGateway},
		Nodes:        []ndf.Node{ourNode},
		Registration: ourPerm,
		Notification: ndf.Notification{},
		UDB:          ndf.UDB{ID: id.UDB.Marshal()},
		E2E:          e2eGrp,
		CMIX:         cmixGrp,
	}

	return networkDef

}

// todo: docstring
func toNdfGroup(grp map[string]string) ndf.Group {
	pStr, pOk := grp["prime"]
	gStr, gOk := grp["generator"]

	if !gOk || !pOk {
		jww.FATAL.Panicf("Invalid Group Config "+
			"(prime: %v, generator: %v",
			pOk, gOk)
	}

	return ndf.Group{
		Prime:     pStr,
		Generator: gStr,
	}
}
