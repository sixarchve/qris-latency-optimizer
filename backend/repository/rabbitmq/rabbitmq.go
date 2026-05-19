package rabbitmq

import (
	"context"
	"fmt"
	"log"
	"qris-latency-optimizer/config"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

var (
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Queue   amqp.Queue
)

// ConnectRabbitMQ connects to RabbitMQ with retry logic (3 attempts)
func ConnectRabbitMQ() {
	url := config.App.RabbitMQURL()
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

	// Declare the queue
	Queue, err = Channel.QueueDeclare(
		"payment_confirmations", // queue name
		true,                    // durable (survives server restart)
		false,                   // delete when unused
		false,                   // exclusive
		false,                   // no-wait
		nil,                     // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	fmt.Println("RabbitMQ connected successfully & Queue declared")
}

// PublishMessage publishes a JSON message to the payment_confirmations queue
func PublishMessage(body string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := Channel.PublishWithContext(ctx,
		"",         // exchange
		Queue.Name, // routing key (queue name)
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        []byte(body),
		})
	return err
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

// Close gracefully closes the RabbitMQ channel and connection
func Close() {
	if Channel != nil {
		Channel.Close()
	}
	if Conn != nil {
		Conn.Close()
	}
}
