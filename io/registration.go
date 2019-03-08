////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating callbacks for registration hooks into comms library

package io

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/server/globals"
)

// Hardcoded DSA keypair for server
var privateKey = signature.ReconstructPrivateKey(
	signature.ReconstructPublicKey(
		signature.CustomDSAParams(
			cyclic.NewIntFromString(
				"30750343950514704536276023434265839029124334317988505564167458417283780314176030679869465145210450450413094649435403726164543712514459220176167267968694610943796790691984406648079016027194069734403521689925779501546155263509657732722190558929364426285634110306795981486263696818928156868365918801033846152395120988973328170239511555007819315208193466977964692046974589973426914693220758498979084474450327743061468231312769193694594128506468508644709444756011318907131361086685125651727256800693418524876453637601836526956223240891296560607966150714035941109120141543878863536333163970787411472405470624101322171728131", 10),
			cyclic.NewIntFromString(
				"75366779259145636267238955202683430013889243239892807787552578959517735157673", 10),
			cyclic.NewIntFromString(
				"29667919687591409914198353837742991079572137554476937089226462133710218883583238403467727953145557913538061592837760340273170525977885886879016977171869688643211242760419053746626249016782063147630778758451949958024384329322511414912513275606170343593654169288530697432772128013431330953535873222785860719194909543905399989610243904698285592416610299341328629653018998580696813874407549733351733730510084753676877180166715437143799509520852354122201551692340784827336662695052209435932781802092383825725276790531653425331914659071667259354695851395276166582574309001529499804706818156092591623443728372224024189225106", 10)),
		cyclic.NewIntFromString(
			"6751039599981084352715746521578470573924749111592696444183240058203587823672266597448716585538776635351866636100189142338543003740859135570402563142388791085351103430637469941413147107451263270911888480904031352894404204606094280538078479517727143032612146908131878618531838224626012318942208820089356983532702653079068155917101802738732889637441531882430875152326668143323492991353471626561296287995660023840633217551835476608525515157813998133589057187869732055580270146493929537003654831419033277442346344225628526783241565038987922780244119906948207566372696999047730660583342188014173351953993741292228967269264", 10)),
	cyclic.NewIntFromString(
		"42550882829954778017125769706877006331096629065463203067508920951713430611837", 10))

// Hardcoded DSA public key for registration server
var registrationPublicKey = signature.ReconstructPublicKey(
	signature.CustomDSAParams(
		cyclic.NewIntFromString(
			"30949293278198593511477868896720820117970022649354083015698831869517631035497255717290731733400631568292419488068270853879446835896394792942527922345721315835087999713923029666645508890669758301919913914652337841732954845320338088953961062197174831408484541724769090600772212012008134779371969302277700742358184424756898924554750635004422565047329512179531783894820452271954825345153615082023708518818544566754998254339197976722237939663162344981456595477791349977541336343369833538285270028247959464556573828897858680537201463583254256851251853663730104874459703467620271776788559438303327562649293237879464353546437", 10),
		cyclic.NewIntFromString(
			"103130506967384723850031368742455125174573322543907594910489683099272678070181", 10),
		cyclic.NewIntFromString(
			"20053598158707091877705744803141780900125016173151622803763124415608912215142522243921671977218421094826664151968507433725949181351220293493915668675115092248247859418457735105445562876402161701317815381401409745196087605995639299122547051518472973100823597655482304240910147969228686025066097699321783719506892914592515223667969703243508512744436567354854972042844926515628991682907710290693211482390402107400664823680296900676390566282037425799361335175234814594516107241808232038880694090781684669671853894494601997160262275091955814872812564499939145091300077656207352736379090482976107073964793474015181883057731", 10)),
	cyclic.NewIntFromString(
		"26463486228990882175412961082117137129236932723218315229770483288885959850107032025589742791223153420660263350741272032644251726448226821113652426407751193610917557252606720692673230795937716224132057988622950839933054136803749612534401778283421126049147077636399968571534289591857614581770480134396036385489740402012223687706110013677953603225892166382503261546944881593648396200024140871260398637499841743509851955620853951655918063387917465276101165812190169399925746034743728536253929853239613918426568522563185456387365104515482472761127162554264087060221287919020535388758461876720271863904323144519014259335081", 10))

// Handle nonce request from Client
func (s ServerImpl) RequestNonce(salt, Y, P, Q, G,
	hash, R, S []byte) ([]byte, error) {

	// Concatenate Client public key byte slices
	data := make([]byte, 0)
	data = append(data, Y...)
	data = append(data, P...)
	data = append(data, Q...)
	data = append(data, G...)

	// Verify signed public key using hardcoded RegistrationServer public key
	valid := registrationPublicKey.Verify(data, signature.DSASignature{
		R: cyclic.NewIntFromBytes(R),
		S: cyclic.NewIntFromBytes(S),
	})

	if !valid {
		// Invalid signed Client public key, return an error
		jww.ERROR.Printf("Unable to verify signed public key!")
		return make([]byte, 0), errors.New("signed public key is invalid")
	}

	// Assemble Client public key
	userPublicKey := signature.ReconstructPublicKey(
		signature.CustomDSAParams(
			cyclic.NewIntFromBytes(P),
			cyclic.NewIntFromBytes(Q),
			cyclic.NewIntFromBytes(G)),
		cyclic.NewIntFromBytes(Y))

	// Generate UserID
	userId := registration.GenUserID(userPublicKey, salt)

	// Generate a nonce with a timestamp
	userNonce := nonce.NewNonce(nonce.RegistrationTTL)

	// Store user information in the database
	newUser := globals.Users.NewUser()
	newUser.Nonce = userNonce
	newUser.ID = userId
	newUser.PublicKey = userPublicKey
	globals.Users.UpsertUser(newUser)

	// Return nonce to Client with empty error field
	return userNonce.Bytes(), nil
}

// Handle confirmation of nonce from Client
func (s ServerImpl) ConfirmNonce(hash, R,
	S []byte) ([]byte, []byte, []byte, error) {

	// Verify signed nonce using Client public key (from Step 7a),
	//  ensuring TTL has not expired
	// If valid signed nonce:
	//     Update user database entry to indicate successful registration
	//     Use hardcoded Server keypair to sign Client public key
	//     Return signed Client public key to Client with empty error field

	// If invalid signed nonce:
	//     Return empty public key to Client with relevant error

	return make([]byte, 0), make([]byte, 0), make([]byte, 0), nil
}
