package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/scrypt"
)

const (
	// Prefix to identify encrypted data and version
	encryptionPrefix = "ENCv1:"

	// Parameters for scrypt KDF
	_scryptN = 1 << 15 // CPU/memory cost parameter (32768)
	r        = 8       // block size
	p        = 1       // parallelization
	keyLen   = 32      // key length for AES-256

	// Sizes for salt and AES-GCM nonce
	saltSize  = 16
	nonceSize = 12
)

// Encrypt takes plaintext and a password, and returns a base64-encoded string
// with a versioned prefix. It uses scrypt for key derivation and AES-GCM
// for authenticated encryption.
func Encrypt(data, password string) (string, error) {
	// Generate random salt
	salt := make([]byte, saltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	// Derive key
	key, err := scrypt.Key([]byte(password), salt, _scryptN, r, p, keyLen)
	if err != nil {
		return "", err
	}

	// Create AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate nonce
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Seal (encrypt + authenticate)
	ciphertext := gcm.Seal(nil, nonce, []byte(data), nil)

	// Concatenate salt|nonce|ciphertext
	var payload []byte
	payload = append(payload, salt...)
	payload = append(payload, nonce...)
	payload = append(payload, ciphertext...)

	// Base64 encode and add prefix
	return encryptionPrefix + base64.StdEncoding.EncodeToString(payload), nil
}

// Decrypt takes a base64-encoded, versioned encrypted string and password,
// then returns the decrypted plaintext or an error.
func Decrypt(data, password string) (string, error) {
	if !IsEncrypted(data) {
		return "", errors.New("data is not in recognized encrypted format")
	}
	// Remove prefix and decode
	b64 := strings.TrimPrefix(data, encryptionPrefix)
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return "", err
	}

	// Extract salt, nonce, and ciphertext
	if len(raw) < saltSize+nonceSize {
		return "", errors.New("ciphertext too short")
	}
	salt := raw[:saltSize]
	nonce := raw[saltSize : saltSize+nonceSize]
	ciphertext := raw[saltSize+nonceSize:]

	// Derive key
	key, err := scrypt.Key([]byte(password), salt, _scryptN, r, p, keyLen)
	if err != nil {
		return "", err
	}

	// Create AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Open (verify and decrypt)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// IsEncrypted checks whether the given string has the expected encrypted prefix.
func IsEncrypted(data string) bool {
	return strings.HasPrefix(data, encryptionPrefix)
}
