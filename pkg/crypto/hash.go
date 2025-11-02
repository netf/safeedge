package crypto

import (
	"encoding/hex"
	"fmt"
	"io"

	"github.com/zeebo/blake3"
)

// BLAKE3Hash computes the BLAKE3 hash of data
func BLAKE3Hash(data []byte) string {
	hash := blake3.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// BLAKE3HashReader computes the BLAKE3 hash of data from a reader
func BLAKE3HashReader(r io.Reader) (string, error) {
	hasher := blake3.New()
	if _, err := io.Copy(hasher, r); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	hash := hasher.Sum(nil)
	return hex.EncodeToString(hash), nil
}

// VerifyBLAKE3 verifies that data matches the expected hash
func VerifyBLAKE3(data []byte, expectedHash string) bool {
	actualHash := BLAKE3Hash(data)
	return actualHash == expectedHash
}
