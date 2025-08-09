package godfather

import (
	"fmt"

	"log/slog"

	"github.com/nats-io/nats.go"
)

type MessageBus struct {
	connection *nats.Conn
	streamCtx  nats.JetStreamContext
}

// ----------------------------------------------------------------
func NewMessageBus(host string, port int, user string) (*MessageBus, error) {
	// Validate input parameters
	if host == "" {
		return nil, fmt.Errorf("host cannot be empty")
	}
	if port <= 0 || port > 65535 {
		return nil, fmt.Errorf("port must be a valid TCP port (1-65535)")
	}
	if user == "" {
		return nil, fmt.Errorf("user cannot be empty")
	}

	// Create a NATS connection string
	natsConnString := fmt.Sprintf("nats://%s:%d", host, port)
	slog.Debug("Connecting to NATS server", "url", natsConnString)
	nc, err := nats.Connect(natsConnString, nats.UserInfo(user, ""))
	if err != nil {
		return nil, err
	}

	// Create a JetStream for alerts if it doesn't already exist
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Stream configuration
	streamName := "alerts"
	streamSubjects := "alerts.*"

	// Check if stream already exists, if not create it
	stream, _ := js.StreamInfo(streamName)
	if stream == nil {
		_, err = js.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Subjects:  []string{streamSubjects},
			Retention: nats.InterestPolicy, // Messages are retained as long as there's interest
			MaxBytes:  1 * 1024 * 1024,     // 1MB max size
			Storage:   nats.FileStorage,
		})
		if err != nil {
			nc.Close()
			return nil, fmt.Errorf("failed to create stream '%s': %w", streamName, err)
		}
		slog.Debug("Created stream 'alerts'")
	} else {
		slog.Debug("Stream 'alerts' already exists")
	}

	return &MessageBus{connection: nc, streamCtx: js}, nil
}

// ----------------------------------------------------------------
func (mb *MessageBus) PublishAlert(subject string, message []byte) error {
	if mb.connection == nil {
		return fmt.Errorf("message bus connection is not initialized")
	}

	// Publish the message to the specified subject
	_, err := mb.streamCtx.Publish(subject, message)
	if err != nil {
		return fmt.Errorf("failed to publish alert: %w", err)
	}

	slog.Debug("Alert published", "subject", subject)
	return nil
}

// ----------------------------------------------------------------
func (mb *MessageBus) Close() {
	if mb.connection != nil {
		mb.connection.Close()
		slog.Debug("Message bus connection closed")
	}
}
