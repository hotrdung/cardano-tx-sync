// internal/kafka/producer.go
package kafka

import (
	"encoding/json"

	"github.com/IBM/sarama"
)

// Producer wraps a Sarama SyncProducer.
type Producer struct {
	producer sarama.SyncProducer
}

// NewProducer creates a new Kafka producer.
func NewProducer(brokers []string) (*Producer, error) {
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, err
	}

	return &Producer{producer: producer}, nil
}

// SendMessage sends a message to a Kafka topic.
func (p *Producer) SendMessage(topic string, message interface{}) error {
	msgBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	msg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msgBytes),
	}

	_, _, err = p.producer.SendMessage(msg)
	return err
}

// Close closes the producer.
func (p *Producer) Close() error {
	return p.producer.Close()
}
