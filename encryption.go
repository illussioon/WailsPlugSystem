package wailsplugs

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

var encryptedPayloadMagic = []byte("WAILSPLUGS-ENC-1")

const encryptedPayloadNonceSize = 12

// EncryptPayload encrypts a package payload with AES-256-GCM. The key must be
// exactly 32 bytes. The returned envelope contains only a format marker,
// nonce, and authenticated ciphertext; it is not a ZIP archive.
func EncryptPayload(payload, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: AES-256 key must be exactly 32 bytes", ErrDecryption)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: cipher initialization failed", ErrDecryption)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, encryptedPayloadNonceSize)
	if err != nil {
		return nil, fmt.Errorf("%w: GCM initialization failed", ErrDecryption)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: nonce generation failed", ErrDecryption)
	}
	ciphertext := gcm.Seal(nil, nonce, payload, encryptedPayloadMagic)
	result := make([]byte, 0, len(encryptedPayloadMagic)+len(nonce)+len(ciphertext))
	result = append(result, encryptedPayloadMagic...)
	result = append(result, nonce...)
	result = append(result, ciphertext...)
	return result, nil
}

// DecryptPayload authenticates and decrypts an AES-256-GCM package payload.
func DecryptPayload(envelope, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("%w: AES-256 key must be exactly 32 bytes", ErrDecryption)
	}
	minimum := len(encryptedPayloadMagic) + encryptedPayloadNonceSize
	if len(envelope) < minimum {
		return nil, fmt.Errorf("%w: encrypted envelope is truncated", ErrDecryption)
	}
	if string(envelope[:len(encryptedPayloadMagic)]) != string(encryptedPayloadMagic) {
		return nil, fmt.Errorf("%w: unsupported envelope", ErrDecryption)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("%w: cipher initialization failed", ErrDecryption)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, encryptedPayloadNonceSize)
	if err != nil {
		return nil, fmt.Errorf("%w: GCM initialization failed", ErrDecryption)
	}
	nonceStart := len(encryptedPayloadMagic)
	nonceEnd := nonceStart + gcm.NonceSize()
	plaintext, err := gcm.Open(nil, envelope[nonceStart:nonceEnd], envelope[nonceEnd:], encryptedPayloadMagic)
	if err != nil {
		return nil, fmt.Errorf("%w: authentication failed", ErrDecryption)
	}
	return plaintext, nil
}
