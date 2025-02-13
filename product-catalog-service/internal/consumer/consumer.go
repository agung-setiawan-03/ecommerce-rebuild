package consumer

import (
	"context"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/kafka-go"
	"product-catalog-service/internal/entity"
	"product-catalog-service/internal/service"
	"strings"
)

type Consumer struct {
	productSvc *service.ProductService
}

func NewConsumer(productSvc *service.ProductService) *Consumer {
	return &Consumer{productSvc: productSvc}
}

// StartKafkaConsumer starts a Kafka consumer to listen for order events
func (c *Consumer) StartKafkaConsumer() {
	// Create Kafka reader for order topic
	orderReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092", "localhost:9093", "localhost:9094"},
		Topic:    "order-topic",
		GroupID:  "product-service-group",
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	for {
		// Read message from order topic
		ctx := context.Background()
		msg, err := orderReader.ReadMessage(ctx)
		if err != nil {
			log.Error().Msgf("Error reading message: %v", err)
			continue
		}

		// Process message
		c.processMessage(ctx, msg)
	}
}

// processMessage processes the message received from the Kafka topic
func (c *Consumer) processMessage(ctx context.Context, msg kafka.Message) {
	// Unmarshal the message payload
	var orderEvent entity.Order

	err := json.Unmarshal(msg.Value, &orderEvent)
	if err != nil {
		log.Error().Msgf("Error unmarshalling message: %v", err)
		return
	}

	// key -> "order.created.orderID" or "order.cancelled.orderID"
	key := string(msg.Key)
	listKey := strings.Split(key, ".")
	eventType := listKey[1]

	// Process the order event based on the status
	switch eventType {
	case "created":
		// Process order created event
		for _, item := range orderEvent.ProductRequests {
			err := c.productSvc.ReserveProductStock(ctx, item.ProductID, item.Quantity)
			if err != nil {
				log.Error().Msgf("Error updating stock for product %d: %v", item.ProductID, err)
			}
		}
	case "cancelled":
		// Process order cancelled event
		for _, item := range orderEvent.ProductRequests {
			err := c.productSvc.ReleaseProductStock(ctx, item.ProductID, item.Quantity)
			if err != nil {
				log.Error().Msgf("Error updating stock for product %d: %v", item.ProductID, err)
			}
		}
	default:
		log.Error().Msgf("Unknown order status: %s", orderEvent.Status)
	}
}
