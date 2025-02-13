package config

import "github.com/segmentio/kafka-go"

var KafkaBrokerURLs = []string{"localhost:9092", "localhost:9093", "localhost:9094"}

func NewKafkaReader(topic, groupID string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:  KafkaBrokerURLs,
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
}
