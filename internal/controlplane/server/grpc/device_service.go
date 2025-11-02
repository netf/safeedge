package grpc

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/netf/safeedge/api/proto/gen"
	"github.com/netf/safeedge/internal/controlplane/database/generated"
)

type DeviceService struct {
	pb.UnimplementedDeviceServiceServer
	queries *generated.Queries
	logger  *zap.Logger

	// Active device streams
	mu      sync.RWMutex
	streams map[string]pb.DeviceService_DeviceStreamServer
}

func NewDeviceService(queries *generated.Queries, logger *zap.Logger) *DeviceService {
	return &DeviceService{
		queries: queries,
		logger:  logger,
		streams: make(map[string]pb.DeviceService_DeviceStreamServer),
	}
}

func (s *DeviceService) Register(server *grpc.Server) {
	pb.RegisterDeviceServiceServer(server, s)
}

func (s *DeviceService) DeviceStream(stream pb.DeviceService_DeviceStreamServer) error {
	var deviceID string
	ctx := stream.Context()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("device stream context done", zap.String("device_id", deviceID))
			s.removeStream(deviceID)
			return ctx.Err()
		default:
		}

		msg, err := stream.Recv()
		if err == io.EOF {
			s.logger.Info("device stream closed", zap.String("device_id", deviceID))
			s.removeStream(deviceID)
			return nil
		}
		if err != nil {
			s.logger.Error("error receiving from device stream",
				zap.String("device_id", deviceID),
				zap.Error(err),
			)
			s.removeStream(deviceID)
			return status.Errorf(codes.Internal, "receive error: %v", err)
		}

		// Handle different message types
		switch payload := msg.Payload.(type) {
		case *pb.DeviceMessage_Heartbeat:
			if err := s.handleHeartbeat(ctx, stream, payload.Heartbeat); err != nil {
				s.logger.Error("heartbeat error",
					zap.String("device_id", payload.Heartbeat.DeviceId),
					zap.Error(err),
				)
				return err
			}
			deviceID = payload.Heartbeat.DeviceId
			s.addStream(deviceID, stream)

		case *pb.DeviceMessage_Health:
			if err := s.handleHealthReport(ctx, payload.Health); err != nil {
				s.logger.Error("health report error",
					zap.String("device_id", payload.Health.DeviceId),
					zap.Error(err),
				)
			}

		case *pb.DeviceMessage_UpdateAck:
			if err := s.handleUpdateAck(ctx, payload.UpdateAck); err != nil {
				s.logger.Error("update ack error",
					zap.String("device_id", payload.UpdateAck.DeviceId),
					zap.Error(err),
				)
			}

		default:
			s.logger.Warn("unknown message type")
		}
	}
}

func (s *DeviceService) handleHeartbeat(ctx context.Context, stream pb.DeviceService_DeviceStreamServer, hb *pb.HeartbeatRequest) error {
	deviceUUID, err := uuid.Parse(hb.DeviceId)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid device ID: %v", err)
	}

	// Update last_seen_at in database
	_, err = s.queries.UpdateDeviceHeartbeat(ctx, deviceUUID)
	if err != nil {
		s.logger.Warn("failed to update device heartbeat",
			zap.String("device_id", hb.DeviceId),
			zap.Error(err),
		)
	}

	s.logger.Debug("heartbeat received",
		zap.String("device_id", hb.DeviceId),
		zap.String("agent_version", hb.AgentVersion),
	)

	// Send heartbeat acknowledgment
	ack := &pb.ControlMessage{
		Payload: &pb.ControlMessage_HeartbeatAck{
			HeartbeatAck: &pb.HeartbeatAck{
				Timestamp: hb.Timestamp,
			},
		},
	}

	return stream.Send(ack)
}

func (s *DeviceService) handleHealthReport(ctx context.Context, health *pb.HealthReport) error {
	s.logger.Info("health report received",
		zap.String("device_id", health.DeviceId),
		zap.String("rollout_id", health.RolloutId),
		zap.Bool("healthy", health.Healthy),
	)

	// TODO: Update rollout_device_status table
	// This will be implemented when we add the rollout service

	return nil
}

func (s *DeviceService) handleUpdateAck(ctx context.Context, ack *pb.UpdateAck) error {
	s.logger.Info("update ack received",
		zap.String("device_id", ack.DeviceId),
		zap.String("rollout_id", ack.RolloutId),
		zap.String("status", ack.Status.String()),
	)

	// TODO: Update rollout_device_status table
	// This will be implemented when we add the rollout service

	return nil
}

func (s *DeviceService) addStream(deviceID string, stream pb.DeviceService_DeviceStreamServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.streams[deviceID] = stream
	s.logger.Info("device stream registered", zap.String("device_id", deviceID))
}

func (s *DeviceService) removeStream(deviceID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.streams, deviceID)
	s.logger.Info("device stream removed", zap.String("device_id", deviceID))
}

// SendUpdateNotification sends an update notification to a specific device
func (s *DeviceService) SendUpdateNotification(deviceID string, update *pb.UpdateNotification) error {
	s.mu.RLock()
	stream, ok := s.streams[deviceID]
	s.mu.RUnlock()

	if !ok {
		return status.Errorf(codes.NotFound, "device %s not connected", deviceID)
	}

	msg := &pb.ControlMessage{
		Payload: &pb.ControlMessage_Update{
			Update: update,
		},
	}

	return stream.Send(msg)
}

// SendRollbackRequest sends a rollback request to a specific device
func (s *DeviceService) SendRollbackRequest(deviceID string, rollback *pb.RollbackRequest) error {
	s.mu.RLock()
	stream, ok := s.streams[deviceID]
	s.mu.RUnlock()

	if !ok {
		return status.Errorf(codes.NotFound, "device %s not connected", deviceID)
	}

	msg := &pb.ControlMessage{
		Payload: &pb.ControlMessage_Rollback{
			Rollback: rollback,
		},
	}

	return stream.Send(msg)
}
