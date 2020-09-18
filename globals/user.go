///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package globals

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/nonce"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"strconv"
	"sync"
)

const MaxSalts = 300

var ErrNonexistantUser = errors.New("user not found in user registry")
var errTooManySalts = "user %v must rekey, has stored too many salts"
var ErrSaltIncorrectLength = errors.New("salt of incorrect length, must be 256 bits")
var ErrUserIDTooShort = errors.New("User id length too short")

// Globally initiated User ID counter
var idCounter = uint64(1)

// Interface for User Registry operations
type UserRegistry interface {
	NewUser(grp *cyclic.Group) *User
	DeleteUser(id *id.ID)
	GetUser(id *id.ID, grp *cyclic.Group) (user *User, err error)
	UpsertUser(user *User)
	CountUsers() int
	InsertSalt(user *id.ID, salt []byte) error
}

// Structure implementing the UserRegistry Interface with an underlying sync.Map
type UserMap sync.Map

// Structure representing a User in the system
type User struct {
	ID           *id.ID
	HUID         []byte
	BaseKey      *cyclic.Int
	RsaPublicKey *rsa.PublicKey
	Nonce        nonce.Nonce

	IsRegistered bool

	salts [][]byte
	sync.Mutex
}

// DeepCopy creates a deep copy of a user and returns a pointer to the new copy
func (u *User) DeepCopy() *User {
	if u == nil {
		return nil
	}
	newUser := new(User)
	newUser.ID = u.ID
	newUser.BaseKey = u.BaseKey.DeepCopy()

	if u.RsaPublicKey != nil && u.RsaPublicKey.PublicKey.N != nil {

		rsaPublicKey, err := rsa.LoadPublicKeyFromPem(rsa.
			CreatePublicKeyPem(u.RsaPublicKey))
		if err != nil {
			jww.ERROR.Printf("Unable to convert PEM to public key: %+v"+
				"\n  PEM: %v",
				errors.New(err.Error()), u.RsaPublicKey)
		}
		newUser.RsaPublicKey = rsaPublicKey
	}

	newUser.Nonce = nonce.Nonce{
		Value:      u.Nonce.Value,
		GenTime:    u.Nonce.GenTime,
		ExpiryTime: u.Nonce.ExpiryTime,
		TTL:        u.Nonce.TTL,
	}

	newUser.IsRegistered = u.IsRegistered
	copy(u.Nonce.Bytes(), newUser.Nonce.Bytes())
	return newUser
}

// NewUser creates a new User object with default fields and given address.
func (m *UserMap) NewUser(grp *cyclic.Group) *User {
	idCounter++
	usr := new(User)
	h := sha256.New()
	i := idCounter - 1

	// Generate user parameters
	usr.ID = new(id.ID)
	binary.BigEndian.PutUint64(usr.ID[:], i)
	usr.ID.SetType(id.User)

	h.Reset()
	h.Write([]byte(strconv.Itoa(int(40000 + i))))
	usr.BaseKey = grp.NewIntFromBytes(h.Sum(nil))
	usr.RsaPublicKey = nil

	usr.Nonce = *new(nonce.Nonce)

	return usr
}

// Inserts a unique salt into the salt table
// Returns true if successful, else false
func (m *UserMap) InsertSalt(id *id.ID, salt []byte) error {
	// If the number of salts for the given UserId
	// is greater than the maximum allowed, then reject

	userFace, ok := (*sync.Map)(m).Load(*id)
	if !ok {
		return ErrNonexistantUser
	}

	user := userFace.(*User)
	user.Lock()
	defer user.Unlock()

	if len(user.salts) >= MaxSalts {
		jww.ERROR.Printf("Unable to insert salt: Too many salts have already"+
			" been used for User %q", *id)
		return errors.New(fmt.Sprintf(errTooManySalts, id))
	}

	// Insert salt into the collection
	user.salts = append(user.salts, salt)
	return nil
}

// DeleteUser deletes a user with the given ID from userCollection.
func (m *UserMap) DeleteUser(id *id.ID) {
	// If key does not exist, do nothing
	(*sync.Map)(m).Delete(*id)
}

// GetUser returns a user with the given ID from userCollection
func (m *UserMap) GetUser(id *id.ID, grp *cyclic.Group) (*User, error) {
	var err error
	var userCopy *User

	u, ok := (*sync.Map)(m).Load(*id)
	if !ok {
		err = ErrNonexistantUser
	} else {
		user := u.(*User)
		user.Lock()
		userCopy = user.DeepCopy()
		user.Unlock()
	}
	return userCopy, err
}

// UpsertUser inserts given user into userCollection or update the user if it
// already exists (Upsert operation).
func (m *UserMap) UpsertUser(user *User) {
	(*sync.Map)(m).Store(*(user.ID), user)
}

// CountUsers returns a count of the users in userCollection.
func (m *UserMap) CountUsers() int {
	numUser := 0

	(*sync.Map)(m).Range(
		func(key, value interface{}) bool {
			numUser++
			return true
		},
	)

	return numUser
}
