// Package crypto provides access to secure encryption and signing methods
package crypto

import (
	"crypto/rand"
	"encoding/json"
	"strconv"

	"github.com/dchest/blake2b"
	triplesec "github.com/keybase/go-triplesec"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/nacl/box"

	"github.com/arigatomachine/cli/daemon/base64"
	"github.com/arigatomachine/cli/daemon/db"
	"github.com/arigatomachine/cli/daemon/envelope"
	"github.com/arigatomachine/cli/daemon/identity"
	"github.com/arigatomachine/cli/daemon/primitive"
	"github.com/arigatomachine/cli/daemon/session"
)

const (
	nonceSize = 16
	blakeSize = 16
)

// SignatureKeyPair is an ed25519/eddsa digital signature keypair.
// The private portion of the keypair is encrypted with triplesec.
//
// PNonce contains the nonce used when deriving the password used to encrypt
// the private portion.
type SignatureKeyPair struct {
	Public  ed25519.PublicKey
	Private []byte
	PNonce  []byte
}

// EncryptionKeyPair is a curve25519 encryption keypair.
// The private portion of the keypair is encrypted with triplesec.
//
// PNonce contains the nonce used when deriving the password used to encrypt
// the private portion.
type EncryptionKeyPair struct {
	Public  [32]byte
	Private []byte
	PNonce  []byte
}

// KeyPairs contains a signature and an encryption keypair for a user.
type KeyPairs struct {
	Signature  SignatureKeyPair
	Encryption EncryptionKeyPair
}

// Engine exposes methods to encrypt, unencrypt and sign values, using
// the logged in user's credentials.
type Engine struct {
	db   *db.DB
	sess session.Session
}

// NewEngine returns a new Engine
func NewEngine(sess session.Session, db *db.DB) *Engine {
	return &Engine{db: db, sess: sess}
}

// Seal encrypts the plaintext pt bytes with triplesec-v3 using a key derrived
// via blake2b from the user's master key and a nonce (returned).
func (e *Engine) Seal(pt []byte) ([]byte, []byte, error) {
	mk, err := e.unsealMasterKey()
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, nonceSize)
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, nil, err
	}

	dk := deriveKey(mk, nonce)
	ts := newTriplesec(dk)
	ct, err := ts.Encrypt(pt)

	return ct, nonce, err
}

// Unseal decrypts the ciphertext ct, encrypted with triplesec-v3, using the
// a key derrived via blake2b from the user's master key and the provided nonce.
func (e *Engine) Unseal(ct, nonce []byte) ([]byte, error) {
	mk, err := e.unsealMasterKey()
	if err != nil {
		return nil, err
	}

	dk := deriveKey(mk, nonce)
	ts := newTriplesec(dk)
	return ts.Decrypt(ct)
}

// GenerateKeyPairs generates and ed25519 signing key pair, and a curve25519
// encryption key pair for the user, encrypting the private keys in
// triplesec-v3 with the user's master key.
func (e *Engine) GenerateKeyPairs() (*KeyPairs, error) {
	kp := &KeyPairs{}

	pubSig, privSig, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	sealedSig, nonceSig, err := e.Seal(privSig)
	if err != nil {
		return nil, err
	}

	kp.Signature.Private = sealedSig
	kp.Signature.Public = pubSig
	kp.Signature.PNonce = nonceSig

	pubEnc, privEnc, err := box.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	sealedEnc, nonceEnc, err := e.Seal((*privEnc)[:])
	if err != nil {
		return nil, err
	}

	kp.Encryption.Private = sealedEnc
	kp.Encryption.Public = *pubEnc
	kp.Encryption.PNonce = nonceEnc

	return kp, nil
}

// Sign signs b bytes using the provided Sealed ed25519 keypair.
func (e *Engine) Sign(s SignatureKeyPair, b []byte) ([]byte, error) {
	pk, err := e.Unseal(s.Private, s.PNonce)
	if err != nil {
		return nil, err
	}

	return ed25519.Sign(pk, b), nil
}

// Verify verifies that sig is the correct signature for b given
// SignatureKeyPair s.
func (e *Engine) Verify(s SignatureKeyPair, b, sig []byte) bool {
	return ed25519.Verify(s.Public, b, sig)
}

// SignedEnvelope returns a new SignedEnvelope containing body
func (e *Engine) SignedEnvelope(body identity.Identifiable,
	sigID *identity.ID, sigKP *SignatureKeyPair) (*envelope.Signed,
	error) {

	b, err := json.Marshal(&body)
	if err != nil {
		return nil, err
	}

	s, err := e.Sign(*sigKP, append([]byte(strconv.Itoa(body.Version())), b...))
	if err != nil {
		return nil, err
	}

	sv := base64.Value(s)
	sig := primitive.Signature{
		PublicKeyID: sigID,
		Algorithm:   EdDSA,
		Value:       &sv,
	}

	id, err := identity.New(body, &sig)
	if err != nil {
		return nil, err
	}

	return &envelope.Signed{
		ID:        &id,
		Version:   1,
		Body:      body,
		Signature: sig,
	}, nil
}

// unsealMasterKey uses the scrypt stretched password to decrypt the master
// password, which is encrypted with triplesec-v3
func (e *Engine) unsealMasterKey() ([]byte, error) {
	ts := newTriplesec([]byte(e.sess.Passphrase()))
	self := envelope.Unsigned{}
	err := e.db.Get(e.sess.ID(), &self)
	if err != nil {
		return nil, err
	}

	mk, err := ts.Decrypt(*(self.Body.(*primitive.User).Master.Value))
	return mk, err
}

func newTriplesec(k []byte) *triplesec.Cipher {
	// err is only set when a salt is given. this won't happen, so
	// let's just panic.
	ts, err := triplesec.NewCipher(k, nil)
	if err != nil {
		panic(err)
	}

	return ts
}

// deriveKey Derives a single use key from the user's master key via blake2b
// and a nonce.
func deriveKey(mk, nonce []byte) []byte {
	h, err := blake2b.New(&blake2b.Config{Size: blakeSize, Key: nonce[:]})
	if err != nil {
		panic(err)
	}

	h.Sum(mk)
	return h.Sum(nil)
}