package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"qris-latency-optimizer/delivery/middleware"
	"qris-latency-optimizer/internal/websocket"
	"qris-latency-optimizer/repository/rabbitmq"
	"qris-latency-optimizer/usecase"
)

type NotificationPayload struct {
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	Timestamp     time.Time `json:"timestamp"`
}

var WSHub *websocket.Hub

func SetWSHub(hub *websocket.Hub) {
	WSHub = hub
}

// StartPaymentConsumer runs a background goroutine to process async payment confirmations
func StartPaymentConsumer(txUsecase usecase.TransactionUsecase) {
	msgs, err := rabbitmq.Channel.Consume(
		rabbitmq.Queue.Name, // queue
		"",                  // consumer
		true,                // auto-ack
		false,               // exclusive
		false,               // no-local
		false,               // no-wait
		nil,                 // args
	)

	if err != nil {
		log.Fatalf("Failed to register RabbitMQ consumer: %v", err)
	}

	go func() {
		for d := range msgs {
			processStart := time.Now()

			var event map[string]string
			if err := json.Unmarshal(d.Body, &event); err != nil {
				log.Printf("[Worker] Error unmarshalling message: %v | Body: %s", err, string(d.Body))
				middleware.RecordPaymentWorkerProcessed("error", time.Since(processStart).Seconds())
				continue
			}

			transactionID := event["transaction_id"]
			if transactionID == "" {
				log.Printf("[Worker] Skipping message with empty transaction_id")
				middleware.RecordPaymentWorkerProcessed("error", time.Since(processStart).Seconds())
				continue
			}

			// We re-use the Sync method because it updates DB and invalidates cache
			_, err := txUsecase.ConfirmPaymentSync(transactionID)
			if err != nil {
				log.Printf("[Worker] Failed to update transaction %s: %v", transactionID, err)
				middleware.RecordPaymentWorkerProcessed("error", time.Since(processStart).Seconds())
				continue
			}

			elapsed := time.Since(processStart)
			middleware.RecordPaymentWorkerProcessed("success", elapsed.Seconds())
			log.Printf("[Worker] Confirmed payment %s in %v", transactionID, elapsed)
		}
	}()

	fmt.Println("RabbitMQ Worker is running and waiting for messages...")
}

func StartNotificationConsumer() {
	go func() {
		channel := rabbitmq.Channel
		if channel == nil {
			log.Println("RabbitMQ channel not available, notification consumer not started")
			return
		}

		q := rabbitmq.GetNotificationQueue()

		msgs, err := channel.Consume(
			q.Name,
			"merchant-notification-consumer",
			false, // manual ack
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Fatalf("Failed to register RabbitMQ notification consumer: %v", err)
		}

		log.Println("RabbitMQ notification consumer is running and waiting for messages...")

		for msg := range msgs {
			var payload NotificationPayload
			err := json.Unmarshal(msg.Body, &payload)
			if err != nil {
				log.Printf("[NotificationWorker] Failed to unmarshal message: %v", err)
				msg.Nack(false, false)
				continue
			}

			if WSHub != nil {
				notification := map[string]interface{}{
					"type":           "transaction_notification",
					"transaction_id": payload.TransactionID,
					"merchant_name":  payload.MerchantName,
					"merchant_id":    payload.MerchantID,
					"amount":         payload.Amount,
					"status":         payload.Status,
					"timestamp":      payload.Timestamp,
				}

				err := WSHub.SendToMerchant(payload.MerchantID, notification)
				if err != nil {
					log.Printf("[NotificationWorker] Failed to send via WebSocket: %v", err)
				}
			} else {
				log.Println("[NotificationWorker] WebSocket hub not initialized")
			}

			msg.Ack(false)
		}
	}()
}
