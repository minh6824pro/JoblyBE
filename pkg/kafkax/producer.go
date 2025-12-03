package kafkax

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
	"go.uber.org/zap"
)

// Producer represents a Kafka producer
type Producer struct {
	writer *kafka.Writer
	log    *log.Helper
}

// ProducerConfig holds the configuration for Kafka producer
type ProducerConfig struct {
	Brokers    []string // Kafka broker addresses, e.g., []string{"localhost:9092"}
	Username   string
	Password   string
	EnableSASL bool
	Timeout    time.Duration
}

// NewProducer creates a new Kafka producer
func NewProducer(config *ProducerConfig, logger log.Logger) *Producer {

	logx := log.NewHelper(logger)
	transport := &kafka.Transport{
		DialTimeout: config.Timeout,
		IdleTimeout: 30 * time.Second,
	}

	// Enable SASL authentication if configured
	if config.EnableSASL {
		mechanism := plain.Mechanism{
			Username: config.Username,
			Password: config.Password,
		}
		transport.SASL = mechanism
		transport.TLS = &tls.Config{}
	}
	writer := &kafka.Writer{
		Addr:         kafka.TCP(config.Brokers...),
		Transport:    transport,
		Balancer:     &kafka.LeastBytes{},
		MaxAttempts:  3,
		WriteTimeout: config.Timeout,
		ReadTimeout:  config.Timeout,
		RequiredAcks: kafka.RequireOne,
		Async:        false,
		Compression:  kafka.Snappy,
	}
	// Ping Kafka to test connection
	conn, err := kafka.Dial("tcp", config.Brokers[0])
	if err != nil {
		logx.Fatal("Kafka connection failed", zap.Error(err))
	}
	defer conn.Close()

	logx.Info("Kafka writer initialized successfully")
	return &Producer{
		writer: writer,
		log:    logx,
	}
}

// SendMessage sends a message to the specified Kafka topic
func (p *Producer) SendMessage(ctx context.Context, topic string, key string, value interface{}) error {
	// Marshal value to JSON
	valueBytes, err := json.Marshal(value)
	if err != nil {
		p.log.Errorf("Failed to marshal message: %v", err)
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	// Create Kafka message
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: valueBytes,
	}

	// Send message
	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.Errorf("Failed to send message to Kafka topic %s: %v", topic, err)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

// SendRawMessage sends a raw message (already serialized) to Kafka
func (p *Producer) SendRawMessage(ctx context.Context, topic string, key string, value []byte) error {
	msg := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
	}

	if err := p.writer.WriteMessages(ctx, msg); err != nil {
		p.log.Errorf("Failed to send raw message to Kafka topic %s: %v", topic, err)
		return fmt.Errorf("failed to send raw message: %w", err)
	}

	p.log.Infof("Raw message sent to topic %s with key %s", topic, key)
	return nil
}

// SendBatch sends multiple messages to Kafka in a single batch
func (p *Producer) SendBatch(ctx context.Context, topic string, messages []kafka.Message) error {
	// Set topic for all messages
	for i := range messages {
		messages[i].Topic = topic
	}

	if err := p.writer.WriteMessages(ctx, messages...); err != nil {
		p.log.Errorf("Failed to send batch messages to Kafka topic %s: %v", topic, err)
		return fmt.Errorf("failed to send batch messages: %w", err)
	}

	p.log.Infof("Batch of %d messages sent to topic %s", len(messages), topic)
	return nil
}

// Close closes the Kafka producer
func (p *Producer) Close() error {
	if err := p.writer.Close(); err != nil {
		p.log.Errorf("Failed to close Kafka producer: %v", err)
		return err
	}
	p.log.Info("Kafka producer closed")
	return nil
}
