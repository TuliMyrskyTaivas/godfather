package godfather

import (
	"fmt"

	"log/slog"

	"github.com/nats-io/nats.go"
)

type MessageBus struct {
	connection *nats.Conn
	stream     nats.JetStreamContext
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

	// Create a JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &MessageBus{connection: nc, stream: js}, nil
}

// ----------------------------------------------------------------
func (mb *MessageBus) CreateStream(streamName string, streamSubjects string) error {
	// Validate input parameters
	if streamName == "" {
		return fmt.Errorf("stream name cannot be empty")
	}
	if streamSubjects == "" {
		return fmt.Errorf("stream subjects cannot be empty")
	}

	// Check if stream already exists, if not create it
	stream, _ := mb.stream.StreamInfo(streamName)
	if stream == nil {
		_, err := mb.stream.AddStream(&nats.StreamConfig{
			Name:      streamName,
			Subjects:  []string{streamSubjects},
			Retention: nats.InterestPolicy, // Messages are retained as long as there's interest
			MaxBytes:  1 * 1024 * 1024,     // 1MB max size
			Storage:   nats.FileStorage,
		})
		if err != nil {
			return fmt.Errorf("failed to create stream '%s': %w", streamName, err)
		}
		slog.Debug("Created stream", "name", streamName, "subjects", streamSubjects)
	} else {
		slog.Debug("Stream already exists", "name", streamName)
	}
	return nil
}

// ----------------------------------------------------------------
func (mb *MessageBus) Publish(subject string, message []byte) error {
	if mb.connection == nil {
		return fmt.Errorf("message bus connection is not initialized")
	}

	// Publish the message to the specified subject
	_, err := mb.stream.Publish(subject, message)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	slog.Debug("Message published", "subject", subject)
	return nil
}

// ----------------------------------------------------------------
func (mb *MessageBus) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	if mb.connection == nil {
		return nil, fmt.Errorf("message bus connection is not initialized")
	}

	// Subscribe to the specified subject
	subscription, err := mb.stream.Subscribe(subject, handler)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to subject '%s': %w", subject, err)
	}

	slog.Debug("Subscribed to subject", "subject", subject)
	return subscription, nil
}

// ----------------------------------------------------------------
func (mb *MessageBus) Close() {
	if mb.connection != nil {
		mb.connection.Close()
		slog.Debug("Message bus connection closed")
	}
}
