package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/netf/safeedge/internal/controlplane/database/generated"
)

type CreateEnrollmentTokenRequest struct {
	OrganizationID   string `json:"organization_id"`
	SiteTag          string `json:"site_tag,omitempty"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
	MaxUses          int    `json:"max_uses"`
}

type CreateEnrollmentTokenResponse struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	MaxUses   int       `json:"max_uses"`
}

func CreateEnrollmentToken(queries *generated.Queries, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateEnrollmentTokenRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode request", zap.Error(err))
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Generate random token
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			logger.Error("failed to generate token", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		token := hex.EncodeToString(tokenBytes)

		// Hash token for storage
		hash := sha256.Sum256([]byte(token))
		tokenHash := hex.EncodeToString(hash[:])

		orgID, err := uuid.Parse(req.OrganizationID)
		if err != nil {
			http.Error(w, "invalid organization ID", http.StatusBadRequest)
			return
		}

		expiresAt := time.Now().UTC().Add(time.Duration(req.ExpiresInSeconds) * time.Second)

		enrollmentToken, err := queries.CreateEnrollmentToken(r.Context(), generated.CreateEnrollmentTokenParams{
			OrganizationID: orgID,
			TokenHash:      tokenHash,
			SiteTag:        stringToText(req.SiteTag),
			ExpiresAt:      expiresAt,
			MaxUses:        int32(req.MaxUses),
		})
		if err != nil {
			logger.Error("failed to create enrollment token", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		logger.Info("enrollment token created",
			zap.String("token_id", enrollmentToken.ID.String()),
			zap.String("organization_id", orgID.String()),
		)

		resp := CreateEnrollmentTokenResponse{
			ID:        enrollmentToken.ID.String(),
			Token:     token, // Return unhashed token only once
			ExpiresAt: enrollmentToken.ExpiresAt,
			MaxUses:   int(enrollmentToken.MaxUses),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

type EnrollDeviceRequest struct {
	Token              string `json:"token"`
	PublicKey          string `json:"public_key"`
	WireguardPublicKey string `json:"wireguard_public_key"`
	Platform           string `json:"platform"`
	AgentVersion       string `json:"agent_version"`
	SiteTag            string `json:"site_tag,omitempty"`
}

type EnrollDeviceResponse struct {
	DeviceID    string `json:"device_id"`
	WireguardIP string `json:"wireguard_ip"`
	Status      string `json:"status"`
}

func EnrollDevice(queries *generated.Queries, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req EnrollDeviceRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to decode request", zap.Error(err))
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		// Hash the provided token
		hash := sha256.Sum256([]byte(req.Token))
		tokenHash := hex.EncodeToString(hash[:])

		// Verify enrollment token
		enrollmentToken, err := queries.GetEnrollmentTokenByHash(r.Context(), tokenHash)
		if err != nil {
			logger.Warn("invalid enrollment token", zap.String("token_hash", tokenHash))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Increment token usage
		if _, err := queries.IncrementTokenUsage(r.Context(), enrollmentToken.ID); err != nil {
			logger.Error("failed to increment token usage", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Allocate WireGuard IP from pool
		// For M0, use simple sequential allocation
		// In production, implement proper IPAM with IP pool management
		deviceCount, err := queries.CountDevicesByStatus(r.Context(), generated.CountDevicesByStatusParams{
			OrganizationID: enrollmentToken.OrganizationID,
			Status:         "ACTIVE",
		})
		if err != nil {
			logger.Error("failed to count devices", zap.Error(err))
			deviceCount = 0
		}

		// Allocate IP: 10.100.0.x where x = deviceCount + 2 (skip .0 and .1)
		wireguardIP := net.IPv4(10, 100, 0, byte(deviceCount+2))

		// Create device
		device, err := queries.CreateDevice(r.Context(), generated.CreateDeviceParams{
			OrganizationID:     enrollmentToken.OrganizationID,
			PublicKey:          req.PublicKey,
			WireguardPublicKey: req.WireguardPublicKey,
			WireguardIp:        wireguardIP,
			AgentVersion:       req.AgentVersion,
			Platform:           req.Platform,
			SiteTag:            stringToText(req.SiteTag),
		})
		if err != nil {
			logger.Error("failed to create device", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		logger.Info("device enrolled",
			zap.String("device_id", device.ID.String()),
			zap.String("platform", device.Platform),
		)

		resp := EnrollDeviceResponse{
			DeviceID:    device.ID.String(),
			WireguardIP: wireguardIP.String(),
			Status:      device.Status,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func stringToText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}
