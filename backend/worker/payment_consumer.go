package worker

import (
	"encoding/json"
	"log"

	"qris-latency-optimizer/internal/websocket"
	"qris-latency-optimizer/repository/rabbitmq"

	
)

// NotificationPayload - struktur data dari RabbitMQ
type NotificationPayload struct {
	TransactionID string    `json:"transaction_id"`
	MerchantID    string    `json:"merchant_id"`
	MerchantName  string    `json:"merchant_name"`
	Amount        float64   `json:"amount"`
	Status        string    `json:"status"`
	Timestamp     string    `json:"timestamp"`
}

// Global reference ke WebSocket hub
var WSHub *websocket.Hub

// SetWSHub - set the WebSocket hub reference
func SetWSHub(hub *websocket.Hub) {
	WSHub = hub
}

// StartPaymentConsumer - start consuming messages dari RabbitMQ
func StartPaymentConsumer() {
	go func() {
		channel := rabbitmq.Channel
		if channel == nil {
			log.Println("⚠ RabbitMQ channel not available, consumer not started")
			return
		}

		// Use NotificationQueue yang sudah dideklarasi
		q := rabbitmq.GetNotificationQueue()

		msgs, err := channel.Consume(
			q.Name,
			"consumer",
			false, // manual ack
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			log.Fatalf("❌ Failed to consume: %v", err)
		}

		log.Println("✓ Consumer worker started, listening for notifications...")

		for msg := range msgs {
			var payload NotificationPayload
			err := json.Unmarshal(msg.Body, &payload)
			if err != nil {
				log.Printf("❌ Failed to unmarshal: %v", err)
				msg.Nack(false, false)
				continue
			}

			log.Printf("📨 Processing notification [TX: %s, Merchant: %s]", 
				payload.TransactionID, payload.MerchantName)

			// Push ke WebSocket hub
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
					log.Printf("⚠ Failed to send via WebSocket: %v", err)
				}
			} else {
				log.Println("⚠ WebSocket hub not initialized")
			}

			msg.Ack(false) // acknowledge success
		}
	}()
}