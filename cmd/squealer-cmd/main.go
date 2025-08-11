package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/TuliMyrskyTaivas/godfather/internal/godfather"
	"github.com/nats-io/nats.go"
	"github.com/vmihailenco/msgpack"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// ----------------------------------------------------------------
func sendTelegramNotification(message string, botId string, chatId int64) {
	bot, err := tgbotapi.NewBotAPI(botId)
	if err != nil {
		slog.Error("Failed to create Telegram bot", "error", err)
		return
	}
	msg := tgbotapi.NewMessage(chatId, message)
	if _, err := bot.Send(msg); err != nil {
		slog.Error("Failed to send Telegram message", "error", err)
		return
	}
}

// ----------------------------------------------------------------
func handleNotifications(ctx context.Context, db *godfather.Database, mb *godfather.MessageBus) {
	// Subscribe to JetStream "alerts"
	subscription, err := mb.PushSubscribe("Squealer", "alerts", "alerts.MOEX", func(msg *nats.Msg) {
		if err := msg.Ack(); err != nil {
			slog.Error("Failed to acknowledge message", "error", err)
			return
		}

		// Handle the incoming message
		var alert godfather.AlertMessage
		if err := msgpack.Unmarshal(msg.Data, &alert); err != nil {
			slog.Error("Failed to unmarshal alert message", "error", err)
			return
		}

		slog.Debug(fmt.Sprintf("Received alert %s for notification ID %d", alert.Subject, alert.NotificationId))

		// Read the notification from the database
		notification, err := db.GetNotificationByID(alert.NotificationId)
		if err != nil {
			slog.Error("Failed to get notification by ID", "error", err)
			return
		}

		// Send telegram notification if configured
		if notification.TelegramBotID != "" && notification.TelegramChatID != 0 {
			sendTelegramNotification(alert.Subject, notification.TelegramBotID, notification.TelegramChatID)
		} else {
			slog.Warn("No Telegram bot or chat ID configured for notification", "notificationID", notification.ID)
		}
	})
	if err != nil {
		slog.Error("Failed to subscribe to alerts", "error", err)
		return
	}

	// Wait for the stop signal
	<-ctx.Done()

	// Unsubscribe from JetStream "alerts"
	if subscription != nil {
		if err := subscription.Unsubscribe(); err != nil {
			slog.Error("Failed to unsubscribe from alerts", "error", err)
		}
	}
}

// ----------------------------------------------------------------
func main() {
	var configPath string
	var verbose bool
	var help bool

	flag.StringVar(&configPath, "c", "squealer.json", "path to config file")
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&help, "h", false, "show help")
	flag.Parse()

	if help {
		flag.Usage()
		os.Exit(0)
	}

	logger := godfather.SetupLogger(verbose)

	config, err := ParseConfig(configPath)
	if err != nil {
		logger.Error("Failed to parse configuration", "error", err)
		os.Exit(1)
	}

	// Create a context that will be canceled on interrupt/termination
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // Kubernetes/Systemd termination
		syscall.SIGQUIT, // Graceful shutdown
	)
	defer stop() // Release signal resources when main exits

	// Initialize the database connection
	db, err := godfather.InitDB(config.Database.Host, config.Database.Port, config.Database.User, config.Database.Passwd, config.Database.Database)
	if err != nil {
		logger.Error("Failed to initialize database connection", "error", err)
		return
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error("Failed to close database connection", "error", err)
		}
	}()

	// Initialize the message bus (NATS)
	mb, err := godfather.NewMessageBus(config.NATS.Host, config.NATS.Port, config.NATS.User)
	if err != nil {
		logger.Error("Failed to initialize message bus", "error", err)
		return
	}
	defer mb.Close()

	// Create the alerts stream
	err = mb.CreateStream("alerts", "alerts.*")
	if err != nil {
		logger.Error("Failed to create stream for alerts", "error", err)
		return
	}

	// Start processing notifications
	go handleNotifications(ctx, db, mb)

	// Wait for the signal to stop
	<-ctx.Done()
	slog.Info("Received termination signal, shutting down...")
}
