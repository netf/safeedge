package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/netf/safeedge/internal/controlplane/database/generated"
)

func ListDevices(queries *generated.Queries, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Get organization ID from JWT auth
		orgID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

		devices, err := queries.ListDevices(r.Context(), generated.ListDevicesParams{
			OrganizationID: orgID,
			Limit:          100,
			Offset:         0,
		})
		if err != nil {
			logger.Error("failed to list devices", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(devices)
	}
}

func GetDevice(queries *generated.Queries, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid device ID", http.StatusBadRequest)
			return
		}

		device, err := queries.GetDevice(r.Context(), deviceID)
		if err != nil {
			logger.Error("failed to get device", zap.Error(err))
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(device)
	}
}

func SuspendDevice(queries *generated.Queries, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid device ID", http.StatusBadRequest)
			return
		}

		device, err := queries.UpdateDeviceStatus(r.Context(), generated.UpdateDeviceStatusParams{
			ID:     deviceID,
			Status: "SUSPENDED",
		})
		if err != nil {
			logger.Error("failed to suspend device", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(device)
	}
}

func ReactivateDevice(queries *generated.Queries, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		deviceID, err := uuid.Parse(chi.URLParam(r, "id"))
		if err != nil {
			http.Error(w, "invalid device ID", http.StatusBadRequest)
			return
		}

		device, err := queries.UpdateDeviceStatus(r.Context(), generated.UpdateDeviceStatusParams{
			ID:     deviceID,
			Status: "ACTIVE",
		})
		if err != nil {
			logger.Error("failed to reactivate device", zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(device)
	}
}
