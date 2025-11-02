package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// WireGuardKeyPair represents a WireGuard Curve25519 key pair
type WireGuardKeyPair struct {
	PublicKey  [32]byte
	PrivateKey [32]byte
}

// GenerateWireGuardKeyPair generates a new WireGuard key pair
func GenerateWireGuardKeyPair() (*WireGuardKeyPair, error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Clamp the private key as per WireGuard spec
	privateKey[0] &= 248
	privateKey[31] &= 127
	privateKey[31] |= 64

	// Compute public key
	var publicKey [32]byte
	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return &WireGuardKeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}, nil
}

// PublicKeyString returns the base64-encoded public key
func (k *WireGuardKeyPair) PublicKeyString() string {
	return base64.StdEncoding.EncodeToString(k.PublicKey[:])
}

// PrivateKeyString returns the base64-encoded private key
func (k *WireGuardKeyPair) PrivateKeyString() string {
	return base64.StdEncoding.EncodeToString(k.PrivateKey[:])
}

// ParseWireGuardPublicKey parses a base64-encoded WireGuard public key
func ParseWireGuardPublicKey(encoded string) ([32]byte, error) {
	var key [32]byte
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return key, fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(data) != 32 {
		return key, fmt.Errorf("invalid public key size: got %d, want 32", len(data))
	}

	copy(key[:], data)
	return key, nil
}

// ParseWireGuardPrivateKey parses a base64-encoded WireGuard private key
func ParseWireGuardPrivateKey(encoded string) ([32]byte, error) {
	var key [32]byte
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return key, fmt.Errorf("failed to decode private key: %w", err)
	}

	if len(data) != 32 {
		return key, fmt.Errorf("invalid private key size: got %d, want 32", len(data))
	}

	copy(key[:], data)
	return key, nil
}
