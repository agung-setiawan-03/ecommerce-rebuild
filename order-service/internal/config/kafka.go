package config

import (
	"github.com/segmentio/kafka-go"
	"os"
	"strings"
)

func getKafkaBrokerURLs() []string {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092,localhost:9093,localhost:9094" // Default brokers
	}
	return strings.Split(brokers, ",")
}

func NewKafkaWriter(topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(getKafkaBrokerURLs()...),
		Topic:                  topic,
		Balancer:               &kafka.LeastBytes{}, // Balancer for selecting partition
		AllowAutoTopicCreation: true,
	}
}
