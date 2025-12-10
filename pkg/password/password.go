// Package password provides password hashing and verification using Argon2id.
// It also supports legacy bcrypt hashes for backward compatibility.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidHash         = errors.New("invalid hash format")
	ErrIncompatibleVersion = errors.New("incompatible argon2 version")
	ErrMismatchedPassword  = errors.New("passwords do not match")
)

// Argon2id parameters (OWASP recommended)
// https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html
const (
	argon2Time    = 1         // Number of iterations
	argon2Memory  = 64 * 1024 // 64 MB memory
	argon2Threads = 4         // Parallelism factor
	argon2KeyLen  = 32        // Output key length
	argon2SaltLen = 16        // Salt length
)

// Hash generates an Argon2id hash for the given password.
// Returns a string in the format: $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
func Hash(password string) (string, error) {
	salt := make([]byte, argon2SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)

	// Encode to standard format
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argon2Memory, argon2Time, argon2Threads, b64Salt, b64Hash)

	return encoded, nil
}

// Verify checks if a password matches the given hash.
// Supports both Argon2id and legacy bcrypt hashes.
func Verify(password, encodedHash string) error {
	// Check if it's a bcrypt hash (starts with $2a$, $2b$, or $2y$)
	if strings.HasPrefix(encodedHash, "$2") {
		return verifyBcrypt(password, encodedHash)
	}

	// Otherwise, assume Argon2id
	return verifyArgon2id(password, encodedHash)
}

// verifyBcrypt verifies a password against a bcrypt hash (legacy support)
func verifyBcrypt(password, hash string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrMismatchedPassword
	}
	return nil
}

// verifyArgon2id verifies a password against an Argon2id hash
func verifyArgon2id(password, encodedHash string) error {
	// Parse the hash
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 {
		return ErrInvalidHash
	}

	if parts[1] != "argon2id" {
		return ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return ErrInvalidHash
	}
	if version != argon2.Version {
		return ErrIncompatibleVersion
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return ErrInvalidHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return ErrInvalidHash
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return ErrInvalidHash
	}

	// Compute hash with same parameters
	computedHash := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedHash)))

	// Constant-time comparison
	if subtle.ConstantTimeCompare(expectedHash, computedHash) != 1 {
		return ErrMismatchedPassword
	}

	return nil
}

// NeedsRehash checks if a hash should be upgraded to Argon2id.
// Returns true for bcrypt hashes or Argon2id hashes with outdated parameters.
func NeedsRehash(encodedHash string) bool {
	// bcrypt hashes need rehashing
	if strings.HasPrefix(encodedHash, "$2") {
		return true
	}

	// Check if Argon2id parameters are current
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return true
	}

	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return true
	}

	// Rehash if parameters don't match current settings
	return memory != argon2Memory || time != argon2Time || threads != argon2Threads
}
