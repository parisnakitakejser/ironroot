package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"strconv"

	"golang.org/x/crypto/pbkdf2"
)

const (
	pbkdf2Iterations = 100000
	saltSize         = 16
)

// EncryptPrivateKeyGCM encrypts private key DER bytes using PBKDF2 and AES-256-GCM.
func EncryptPrivateKeyGCM(der []byte, password string) (*pem.Block, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("generate salt: %w", err)
	}

	// Derive a 32-byte (256-bit) key
	key := pbkdf2.Key([]byte(password), salt, pbkdf2Iterations, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	// Seal DER bytes with AES-GCM
	ciphertext := gcm.Seal(nil, nonce, der, nil)

	return &pem.Block{
		Type: "ENCRYPTED PRIVATE KEY",
		Headers: map[string]string{
			"Salt":  hex.EncodeToString(salt),
			"Iter":  strconv.Itoa(pbkdf2Iterations),
			"Nonce": hex.EncodeToString(nonce),
		},
		Bytes: ciphertext,
	}, nil
}

// DecryptPrivateKeyGCM decrypts private key DER bytes using PBKDF2 and AES-256-GCM.
func DecryptPrivateKeyGCM(block *pem.Block, password string) ([]byte, error) {
	saltHex := block.Headers["Salt"]
	iterStr := block.Headers["Iter"]
	nonceHex := block.Headers["Nonce"]

	if saltHex == "" || iterStr == "" || nonceHex == "" {
		return nil, errors.New("missing cryptographic headers in encrypted PEM block")
	}

	salt, err := hex.DecodeString(saltHex)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}

	iter, err := strconv.Atoi(iterStr)
	if err != nil {
		return nil, fmt.Errorf("parse iterations: %w", err)
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}

	key := pbkdf2.Key([]byte(password), salt, iter, 32, sha256.New)

	aesBlock, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(aesBlock)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	der, err := gcm.Open(nil, nonce, block.Bytes, nil)
	if err != nil {
		return nil, errors.New("failed to decrypt private key (check password)")
	}

	return der, nil
}

// IsEncryptedPEMBlockGCM checks if the block is encrypted using our GCM method.
func IsEncryptedPEMBlockGCM(block *pem.Block) bool {
	return block.Type == "ENCRYPTED PRIVATE KEY" && block.Headers["Salt"] != ""
}
