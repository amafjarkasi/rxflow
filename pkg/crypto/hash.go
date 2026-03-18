package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

var (
	ErrEmptySalt = errors.New("salt cannot be empty")
)

// Hasher provides keyed HMAC hashing for patient identifiers
// This fulfills the "Patient hash salting" requirement
type Hasher struct {
	salt []byte
}

// NewHasher initializes a new keyed HMAC hasher
func NewHasher(salt []byte) (*Hasher, error) {
	if len(salt) == 0 {
		return nil, ErrEmptySalt
	}
	return &Hasher{salt: salt}, nil
}

// HashPatientData creates a deterministic, salted hash of patient identifiers
func (h *Hasher) HashPatientData(identifier, dob, name string) string {
	mac := hmac.New(sha256.New, h.salt)

	// Canonicalize data before hashing
	mac.Write([]byte(identifier))
	mac.Write([]byte("|"))
	mac.Write([]byte(dob))
	mac.Write([]byte("|"))
	mac.Write([]byte(name))

	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyPatientData checks if the provided data matches a given hash
func (h *Hasher) VerifyPatientData(identifier, dob, name, expectedHash string) bool {
	computedHash := h.HashPatientData(identifier, dob, name)

	// Use constant-time comparison to prevent timing attacks
	return hmac.Equal([]byte(computedHash), []byte(expectedHash))
}
