package enrollment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/netf/safeedge/pkg/crypto"
)

// DeviceIdentity stores the device's cryptographic identity
type DeviceIdentity struct {
	DeviceID           string `json:"device_id"`
	PublicKey          string `json:"public_key"`
	PrivateKey         string `json:"private_key"`
	WireguardPublicKey string `json:"wireguard_public_key"`
	WireguardPrivateKey string `json:"wireguard_private_key"`
	WireguardIP        string `json:"wireguard_ip"`
	ControlPlaneURL    string `json:"control_plane_url"`
}

// EnrollRequest is the request payload for device enrollment
type EnrollRequest struct {
	Token              string `json:"token"`
	PublicKey          string `json:"public_key"`
	WireguardPublicKey string `json:"wireguard_public_key"`
	Platform           string `json:"platform"`
	AgentVersion       string `json:"agent_version"`
	SiteTag            string `json:"site_tag,omitempty"`
}

// EnrollResponse is the response from the enrollment endpoint
type EnrollResponse struct {
	DeviceID    string `json:"device_id"`
	WireguardIP string `json:"wireguard_ip"`
	Status      string `json:"status"`
}

// Enroll enrolls the device with the control plane
func Enroll(controlPlaneURL, token, siteTag, identityPath string) (*DeviceIdentity, error) {
	// Generate Ed25519 key pair for device identity
	ed25519Keys, err := crypto.GenerateEd25519KeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Ed25519 keys: %w", err)
	}

	// Generate WireGuard key pair for tunneling
	wgKeys, err := crypto.GenerateWireGuardKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate WireGuard keys: %w", err)
	}

	// Get platform info
	platform := getPlatform()

	// Prepare enrollment request
	enrollReq := EnrollRequest{
		Token:              token,
		PublicKey:          ed25519Keys.PublicKeyString(),
		WireguardPublicKey: wgKeys.PublicKeyString(),
		Platform:           platform,
		AgentVersion:       "0.1.0",
		SiteTag:            siteTag,
	}

	// Make HTTP request to enrollment endpoint
	reqBody, err := json.Marshal(enrollReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(
		controlPlaneURL+"/v1/enrollments",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("enrollment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("enrollment failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var enrollResp EnrollResponse
	if err := json.NewDecoder(resp.Body).Decode(&enrollResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create device identity
	identity := &DeviceIdentity{
		DeviceID:            enrollResp.DeviceID,
		PublicKey:           ed25519Keys.PublicKeyString(),
		PrivateKey:          ed25519Keys.PrivateKeyString(),
		WireguardPublicKey:  wgKeys.PublicKeyString(),
		WireguardPrivateKey: wgKeys.PrivateKeyString(),
		WireguardIP:         enrollResp.WireguardIP,
		ControlPlaneURL:     controlPlaneURL,
	}

	// Save identity to file
	if err := saveIdentity(identity, identityPath); err != nil {
		return nil, fmt.Errorf("failed to save identity: %w", err)
	}

	return identity, nil
}

// LoadIdentity loads the device identity from file
func LoadIdentity(path string) (*DeviceIdentity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read identity file: %w", err)
	}

	var identity DeviceIdentity
	if err := json.Unmarshal(data, &identity); err != nil {
		return nil, fmt.Errorf("failed to parse identity: %w", err)
	}

	return &identity, nil
}

// saveIdentity saves the device identity to file
func saveIdentity(identity *DeviceIdentity, path string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal identity
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal identity: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write identity file: %w", err)
	}

	return nil
}

// getPlatform returns the platform string
func getPlatform() string {
	return fmt.Sprintf("%s-%s", os.Getenv("GOOS"), os.Getenv("GOARCH"))
}
