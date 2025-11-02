package crypto

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Ed25519KeyPair represents an Ed25519 public/private key pair
type Ed25519KeyPair struct {
	PublicKey  ed25519.PublicKey
	PrivateKey ed25519.PrivateKey
}

// GenerateEd25519KeyPair generates a new Ed25519 key pair
func GenerateEd25519KeyPair() (*Ed25519KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ed25519 key pair: %w", err)
	}

	return &Ed25519KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// PublicKeyString returns the base64-encoded public key
func (k *Ed25519KeyPair) PublicKeyString() string {
	return base64.StdEncoding.EncodeToString(k.PublicKey)
}

// PrivateKeyString returns the base64-encoded private key
func (k *Ed25519KeyPair) PrivateKeyString() string {
	return base64.StdEncoding.EncodeToString(k.PrivateKey)
}

// Sign signs a message with the private key
func (k *Ed25519KeyPair) Sign(message []byte) []byte {
	return ed25519.Sign(k.PrivateKey, message)
}

// Verify verifies a signature with the public key
func (k *Ed25519KeyPair) Verify(message, signature []byte) bool {
	return ed25519.Verify(k.PublicKey, message, signature)
}

// ParseEd25519PublicKey parses a base64-encoded public key
func ParseEd25519PublicKey(encoded string) (ed25519.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(data) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: got %d, want %d", len(data), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(data), nil
}

// ParseEd25519PrivateKey parses a base64-encoded private key
func ParseEd25519PrivateKey(encoded string) (ed25519.PrivateKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(data) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size: got %d, want %d", len(data), ed25519.PrivateKeySize)
	}

	return ed25519.PrivateKey(data), nil
}
