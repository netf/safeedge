package rest

import (
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/netf/safeedge/internal/controlplane/database/generated"
	"github.com/netf/safeedge/internal/controlplane/server/rest/handlers"
)

func RegisterRoutes(router chi.Router, queries *generated.Queries, logger *zap.Logger) {
	// API version prefix
	router.Route("/v1", func(r chi.Router) {
		// Enrollment
		r.Post("/enrollment-tokens", handlers.CreateEnrollmentToken(queries, logger))
		r.Post("/enrollments", handlers.EnrollDevice(queries, logger))

		// Devices
		r.Get("/devices", handlers.ListDevices(queries, logger))
		r.Get("/devices/{id}", handlers.GetDevice(queries, logger))
		r.Post("/devices/{id}/suspend", handlers.SuspendDevice(queries, logger))
		r.Post("/devices/{id}/reactivate", handlers.ReactivateDevice(queries, logger))

		// Access Sessions
		r.Post("/access-sessions", handlers.CreateAccessSession(queries, logger))
		r.Delete("/access-sessions/{id}", handlers.TerminateAccessSession(queries, logger))

		// Artifacts
		r.Post("/artifacts", handlers.CreateArtifact(queries, logger))
		r.Get("/artifacts/{id}", handlers.GetArtifact(queries, logger))

		// Rollouts
		r.Post("/rollouts", handlers.CreateRollout(queries, logger))
		r.Get("/rollouts/{id}", handlers.GetRollout(queries, logger))
		r.Post("/rollouts/{id}/start", handlers.StartRollout(queries, logger))
		r.Post("/rollouts/{id}/abort", handlers.AbortRollout(queries, logger))

		// Audit Logs
		r.Get("/audit-logs", handlers.ListAuditLogs(queries, logger))
	})
}
