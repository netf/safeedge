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
	"github.com/netf/safeedge/internal/agent/enrollment"
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

	var enrollmentToken, siteTag, identityPath string

	runCmd.Flags().StringVar(&controlPlaneURL, "control-plane", getEnv("CONTROL_PLANE_URL", "localhost:9090"), "Control plane gRPC address")
	runCmd.Flags().StringVar(&deviceID, "device-id", getEnv("DEVICE_ID", ""), "Device ID (from identity file)")
	runCmd.Flags().StringVar(&identityPath, "identity", getEnv("IDENTITY_PATH", "/var/lib/safeedge/identity.json"), "Path to identity file")
	runCmd.Flags().StringVar(&logLevel, "log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")

	enrollCmd.Flags().StringVar(&controlPlaneURL, "control-plane", getEnv("CONTROL_PLANE_URL", "http://localhost:8080"), "Control plane HTTP address")
	enrollCmd.Flags().StringVar(&enrollmentToken, "token", getEnv("ENROLLMENT_TOKEN", ""), "Enrollment token (required)")
	enrollCmd.Flags().StringVar(&siteTag, "site-tag", getEnv("SITE_TAG", ""), "Site tag for device grouping")
	enrollCmd.Flags().StringVar(&identityPath, "identity", getEnv("IDENTITY_PATH", "/var/lib/safeedge/identity.json"), "Path to save identity file")

	enrollCmd.MarkFlagRequired("token")

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

	identityPath, _ := cmd.Flags().GetString("identity")

	// Load device identity
	identity, err := enrollment.LoadIdentity(identityPath)
	if err != nil {
		return fmt.Errorf("failed to load identity (run 'enroll' first): %w", err)
	}

	deviceID = identity.DeviceID

	// Extract gRPC address from control plane URL
	// If HTTP URL, convert to gRPC (port 9090)
	grpcURL := controlPlaneURL
	if controlPlaneURL == "localhost:9090" || controlPlaneURL == "" {
		grpcURL = "localhost:9090"
	}

	logger.Info("starting SafeEdge agent",
		zap.String("device_id", deviceID),
		zap.String("control_plane", grpcURL),
		zap.String("wireguard_ip", identity.WireguardIP),
	)

	// Connect to control plane gRPC
	conn, err := grpc.NewClient(grpcURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	enrollmentToken, _ := cmd.Flags().GetString("token")
	siteTag, _ := cmd.Flags().GetString("site-tag")
	identityPath, _ := cmd.Flags().GetString("identity")

	fmt.Printf("Enrolling device with control plane at %s...\n", controlPlaneURL)

	identity, err := enrollment.Enroll(controlPlaneURL, enrollmentToken, siteTag, identityPath)
	if err != nil {
		return fmt.Errorf("enrollment failed: %w", err)
	}

	fmt.Println("✓ Enrollment successful!")
	fmt.Printf("  Device ID: %s\n", identity.DeviceID)
	fmt.Printf("  WireGuard IP: %s\n", identity.WireguardIP)
	fmt.Printf("  Identity saved to: %s\n", identityPath)
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Run agent: safeedge-agent run --identity %s\n", identityPath)
	fmt.Println("  2. Configure WireGuard tunnel (TODO)")

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
