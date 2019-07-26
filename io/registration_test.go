////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/server/globals"
	"gitlab.com/elixxir/server/server"
	"gitlab.com/elixxir/server/server/conf"
	"gitlab.com/elixxir/server/server/measure"
	"os"
	"testing"
	"time"
)

var serverInstance *server.Instance
var dsaParams = signature.GetDefaultDSAParams()
var pubKey *signature.DSAPublicKey
var privKey *signature.DSAPrivateKey
var regPrivKey *signature.DSAPrivateKey

func TestMain(m *testing.M) {
	grp := cyclic.NewGroup(dsaParams.GetP(), dsaParams.GetG(), dsaParams.GetQ())

	rng := csprng.NewSystemRNG()
	dsaParams := signature.CustomDSAParams(grp.GetP(), grp.GetQ(), grp.GetG())
	privKey = dsaParams.PrivateKeyGen(rng)
	pubKey = privKey.PublicKeyGen()
	pubKeyStr, _ := pubKey.PemEncode()
	regPrivKey = privKey

	nid := server.GenerateId()
	grps := initConfGroups(grp)

	params := conf.Params{
		Groups: grps,
		Node: conf.Node{
			Ids: []string{nid.String()},
		},
		Permissioning: conf.Permissioning{
			PublicKey: string(pubKeyStr),
		},
	}

	serverInstance = server.CreateServerInstance(&params, &globals.UserMap{},
		pubKey, privKey, measure.ResourceMonitor{})

	os.Exit(m.Run())
}

// TODO: How should the public key be retrieved?
// Is it from the Permissioning.Paths.Cert?
// Perhaps Paths object should get a GetPublicKey Method?
// Test request nonce
func TestRequestNonce(t *testing.T) {
	rng := csprng.NewSystemRNG()
	salt := cmix.NewSalt(rng, 32)

	hash := append(pubKey.GetKey().Bytes(), dsaParams.GetP().Bytes()...)
	hash = append(hash, dsaParams.GetQ().Bytes()...)
	hash = append(hash, dsaParams.GetG().Bytes()...)

	sign, err := regPrivKey.Sign(hash, rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data: %v", err.Error())
	}

	result, err2 := RequestNonce(serverInstance,
		salt,
		pubKey.GetKey().Bytes(),
		dsaParams.GetP().Bytes(),
		dsaParams.GetQ().Bytes(),
		dsaParams.GetG().Bytes(),
		hash,
		sign.R.Bytes(),
		sign.S.Bytes())

	if result == nil || err2 != nil {
		t.Errorf("Error in RequestNonce")
	}
}

// Test request nonce with invalid signature
func TestRequestNonce_BadSignature(t *testing.T) {
	rng := csprng.NewSystemRNG()
	salt := cmix.NewSalt(rng, 32)

	hash := append(pubKey.GetKey().Bytes(), dsaParams.GetP().Bytes()...)
	hash = append(hash, dsaParams.GetQ().Bytes()...)
	hash = append(hash, dsaParams.GetQ().Bytes()...)

	sign, err := regPrivKey.Sign(hash, rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	_, err2 := RequestNonce(serverInstance,
		salt,
		pubKey.GetKey().Bytes(),
		dsaParams.GetP().Bytes(),
		dsaParams.GetQ().Bytes(),
		dsaParams.GetG().Bytes(),
		hash,
		sign.R.Bytes(),
		sign.S.Bytes())

	if err2 == nil {
		t.Errorf("Expected error in RequestNonce")
	}
}

// Test confirm nonce
func TestConfirmNonce(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)
	serverInstance.GetUserRegistry().UpsertUser(user)

	rng := csprng.NewSystemRNG()
	user.PublicKey = pubKey

	sign, err := privKey.Sign(user.Nonce.Bytes(), rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	_, _, _, _, _, _, _, err2 := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), sign.R.Bytes(), sign.S.Bytes())
	if err2 != nil {
		t.Errorf("Error in ConfirmNonce")
	}
}

// Test confirm nonce that doesn't exist
func TestConfirmNonce_NonExistant(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)

	rng := csprng.NewSystemRNG()
	user.PublicKey = pubKey

	sign, err := privKey.Sign(user.Nonce.Bytes(), rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	_, _, _, _, _, _, _, err2 := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), sign.R.Bytes(), sign.S.Bytes())
	if err2 == nil {
		t.Errorf("ConfirmNonce: Expected unexistant nonce")
	}
}

// Test confirm nonce expired
func TestConfirmNonce_Expired(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce, _ = nonce.NewNonce(1)
	serverInstance.GetUserRegistry().UpsertUser(user)

	rng := csprng.NewSystemRNG()
	user.PublicKey = pubKey

	sign, err := privKey.Sign(user.Nonce.Bytes(), rng)
	if sign == nil || err != nil {
		t.Errorf("Error signing data")
	}

	// Wait for nonce to expire
	wait := time.After(time.Duration(2) * time.Second)
	select {
	case <-wait:
	}

	_, _, _, _, _, _, _, err2 := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), sign.R.Bytes(), sign.S.Bytes())
	if err2 == nil {
		t.Errorf("ConfirmNonce: Expected expired nonce")
	}
}

// Test confirm nonce with invalid signature
func TestConfirmNonce_BadSignature(t *testing.T) {
	user := serverInstance.GetUserRegistry().NewUser(serverInstance.GetGroup())
	user.Nonce, _ = nonce.NewNonce(nonce.RegistrationTTL)
	serverInstance.GetUserRegistry().UpsertUser(user)

	_, _, _, _, _, _, _, err := ConfirmRegistration(serverInstance,
		user.Nonce.Bytes(), make([]byte, 0),
		make([]byte, 0))
	if err == nil {
		t.Errorf("ConfirmNonce: Expected bad signature!")
	}
}

func initConfGroups(grp *cyclic.Group) conf.Groups {

	primeString := grp.GetP().TextVerbose(16, 0)
	smallprime := grp.GetQ().TextVerbose(16, 0)
	generator := grp.GetG().TextVerbose(16, 0)

	cmixMap := map[string]string{
		"prime":      primeString,
		"smallprime": smallprime,
		"generator":  generator,
	}

	grps := conf.Groups{
		CMix: cmixMap,
	}

	return grps
}
