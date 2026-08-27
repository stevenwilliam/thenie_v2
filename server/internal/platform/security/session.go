package security

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
)

// SessionTokenBytes is the entropy behind a session token. 32 bytes is well
// past the point where guessing is the attack anyone would try.
const SessionTokenBytes = 32

var ErrTokenMalformed = errors.New("security: session token malformed")

// NewSessionToken returns a URL-safe token and the hash to store.
//
// Only the HASH is persisted. The token exists in the customer's cookie and
// nowhere else, so a leaked database backup contains no usable session — the
// same reason passwords are hashed, applied to the credential that actually
// rides on every request.
func NewSessionToken() (token, hash string, err error) {
	b := make([]byte, SessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("security: session token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, HashSessionToken(token), nil
}

// HashSessionToken returns the stored form of a token.
//
// Plain SHA-256, not argon2: this is a 256-bit random value, not a
// human-chosen password, so there is no dictionary to slow down and adding a
// work factor would only slow down every authenticated request.
func HashSessionToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
