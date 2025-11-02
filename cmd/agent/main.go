package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/netf/safeedge/api/proto/gen"
)

var (
	controlPlaneURL string
	deviceID        string
	logLevel        string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "safeedge-agent",
		Short: "SafeEdge device agent",
		Long:  "Zero-trust device agent for SafeEdge fleet management platform",
	}

	runCmd := &cobra.Command{
		Use:   "run",
		Short: "Run the agent",
		RunE:  runAgent,
	}

	enrollCmd := &cobra.Command{
		Use:   "enroll",
		Short: "Enroll this device with the control plane",
		RunE:  enrollDevice,
	}

	runCmd.Flags().StringVar(&controlPlaneURL, "control-plane", getEnv("CONTROL_PLANE_URL", "localhost:9090"), "Control plane gRPC address")
	runCmd.Flags().StringVar(&deviceID, "device-id", getEnv("DEVICE_ID", ""), "Device ID (required)")
	runCmd.Flags().StringVar(&logLevel, "log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")

	enrollCmd.Flags().StringVar(&controlPlaneURL, "control-plane", getEnv("CONTROL_PLANE_URL", "http://localhost:8080"), "Control plane HTTP address")

	rootCmd.AddCommand(runCmd, enrollCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func runAgent(cmd *cobra.Command, args []string) error {
	logger, err := initLogger(logLevel)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}
	defer logger.Sync()

	if deviceID == "" {
		return fmt.Errorf("device-id is required")
	}

	logger.Info("starting SafeEdge agent",
		zap.String("device_id", deviceID),
		zap.String("control_plane", controlPlaneURL),
	)

	// Connect to control plane gRPC
	conn, err := grpc.NewClient(controlPlaneURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to control plane: %w", err)
	}
	defer conn.Close()

	client := pb.NewDeviceServiceClient(conn)

	// Establish bidirectional stream
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := client.DeviceStream(ctx)
	if err != nil {
		return fmt.Errorf("failed to establish stream: %w", err)
	}

	logger.Info("connected to control plane")

	// Start heartbeat goroutine
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := sendHeartbeat(stream, deviceID, logger); err != nil {
					logger.Error("failed to send heartbeat", zap.Error(err))
				}
			}
		}
	}()

	// Send initial heartbeat
	if err := sendHeartbeat(stream, deviceID, logger); err != nil {
		logger.Error("failed to send initial heartbeat", zap.Error(err))
	}

	// Start message receiver goroutine
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				logger.Error("stream receive error", zap.Error(err))
				cancel()
				return
			}

			handleControlMessage(msg, logger)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down agent...")
	return nil
}

func enrollDevice(cmd *cobra.Command, args []string) error {
	// TODO: Implement device enrollment
	// This will:
	// 1. Generate Ed25519 keypair
	// 2. Generate WireGuard keypair
	// 3. Call enrollment API with token
	// 4. Store identity.json
	// 5. Configure WireGuard tunnel

	fmt.Println("Enrollment not yet implemented")
	return nil
}

func sendHeartbeat(stream pb.DeviceService_DeviceStreamClient, deviceID string, logger *zap.Logger) error {
	msg := &pb.DeviceMessage{
		Payload: &pb.DeviceMessage_Heartbeat{
			Heartbeat: &pb.HeartbeatRequest{
				DeviceId:     deviceID,
				AgentVersion: "0.1.0",
				Metrics: &pb.DeviceMetrics{
					CpuPercent:    10.5,
					MemoryPercent: 45.2,
				},
			},
		},
	}

	if err := stream.Send(msg); err != nil {
		return fmt.Errorf("send error: %w", err)
	}

	logger.Debug("heartbeat sent", zap.String("device_id", deviceID))
	return nil
}

func handleControlMessage(msg *pb.ControlMessage, logger *zap.Logger) {
	switch payload := msg.Payload.(type) {
	case *pb.ControlMessage_HeartbeatAck:
		logger.Debug("heartbeat acknowledged")

	case *pb.ControlMessage_Update:
		logger.Info("update notification received",
			zap.String("rollout_id", payload.Update.RolloutId),
			zap.String("artifact_id", payload.Update.ArtifactId),
		)
		// TODO: Download, verify, and apply update

	case *pb.ControlMessage_Rollback:
		logger.Info("rollback request received",
			zap.String("rollout_id", payload.Rollback.RolloutId),
			zap.String("reason", payload.Rollback.Reason),
		)
		// TODO: Rollback to previous version

	default:
		logger.Warn("unknown control message type")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.NewProductionConfig()
	config.Level = zapLevel
	config.Encoding = "json"

	return config.Build()
}
