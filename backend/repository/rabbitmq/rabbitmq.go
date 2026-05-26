package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Queue   amqp.Queue
	// Queue untuk merchant notifications
	NotificationQueue amqp.Queue
)

// getRabbitMQURL reads the connection URL from env with fallback default
func getRabbitMQURL() string {
	url := os.Getenv("RABBITMQ_URL")
	if url == "" {
		url = "amqp://guest:guest@rabbitmq:5672/"
	}
	return url
}

// ConnectRabbitMQ connects to RabbitMQ with retry logic (3 attempts)
func ConnectRabbitMQ() {
	url := getRabbitMQURL()
	var err error

	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		Conn, err = amqp.Dial(url)
		if err == nil {
			break
		}
		log.Printf("RabbitMQ connection attempt %d/%d failed: %v", attempt, maxRetries, err)
		if attempt < maxRetries {
			backoff := time.Duration(attempt) * 2 * time.Second
			log.Printf("Retrying in %v...", backoff)
			time.Sleep(backoff)
		}
	}

	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ after %d attempts: %v", maxRetries, err)
	}

	Channel, err = Conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open RabbitMQ channel: %v", err)
	}

	// Declare the existing queue (payment_confirmations)
	Queue, err = Channel.QueueDeclare(
		"payment_confirmations",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	// Declare queue untuk merchant notifications
	NotificationQueue, err = Channel.QueueDeclare(
		"merchant_notifications",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatalf("Failed to declare notification queue: %v", err)
	}

	fmt.Println("✓ RabbitMQ connected successfully & Queues declared")
}

// PublishMessage publishes a JSON message to the payment_confirmations queue
func PublishMessage(body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Channel.PublishWithContext(ctx,
		"",
		Queue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	return err
}

// NotificationPayload - struktur data untuk merchant notifications
type NotificationPayload struct {
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

// PublishNotification - publish merchant notification ke queue
func PublishNotification(txID, merchantID, merchantName string, amount float64) error {
	if !IsConnected() {
		return fmt.Errorf("RabbitMQ not connected")
	}

	payload := NotificationPayload{
		TransactionID: txID,
		MerchantID:    merchantID,
		MerchantName:  merchantName,
		Amount:        amount,
		Status:        "SUCCESS",
		Timestamp:     time.Now(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = Channel.PublishWithContext(ctx,
		"",
		NotificationQueue.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		log.Printf("⚠ Failed to publish notification: %v", err)
		return err
	}

	log.Printf("✓ Notification published [TX: %s, Merchant: %s]", txID, merchantName)
	return nil
}

// IsConnected returns true if RabbitMQ connection is alive
func IsConnected() bool {
	if Conn == nil || Conn.IsClosed() {
		return false
	}
	if Channel == nil {
		return false
	}
	return true
}

// GetNotificationQueue returns the notification queue
func GetNotificationQueue() amqp.Queue {
	return NotificationQueue
}

// Close gracefully closes the RabbitMQ channel and connection
func Close() {
	if Channel != nil {
		Channel.Close()
	}
	if Conn != nil {
		Conn.Close()
	}
	log.Println("✓ RabbitMQ closed")
}
