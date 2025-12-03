package kafkax

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"

	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

type Consumer struct {
	readers       []*kafka.Reader
	log           *log.Helper
	topic         string
	groupID       string
	numPartitions int
	numWorkers    int
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
}

type ConsumerConfig struct {
	Brokers       []string
	Username      string
	Password      string
	EnableSASL    bool
	Timeout       time.Duration
	Topic         string
	GroupID       string
	NumPartitions int
	NumWorkers    int
}

// MessageHandler is a function that processes a Kafka message
type MessageHandler func(ctx context.Context, message kafka.Message) error

func NewConsumer(config *ConsumerConfig, logger log.Logger) *Consumer {
	dialer := &kafka.Dialer{
		Timeout:   config.Timeout,
		DualStack: true,
	}

	// Enable SASL authentication if configured
	if config.EnableSASL {
		mechanism := plain.Mechanism{
			Username: config.Username,
			Password: config.Password,
		}
		dialer.SASLMechanism = mechanism
		dialer.TLS = &tls.Config{}
	}

	ctx, cancel := context.WithCancel(context.Background())

	numPartitions := config.NumPartitions
	if numPartitions <= 0 {
		numPartitions = 1 // Default to 1 partition
	}

	numWorkers := config.NumWorkers
	if numWorkers <= 0 {
		numWorkers = 1 // Default to 1 worker per consumer
	}

	// Create multiple readers (one per partition)
	readers := make([]*kafka.Reader, numPartitions)
	for i := 0; i < numPartitions; i++ {
		readerConfig := kafka.ReaderConfig{
			Brokers:        config.Brokers,
			Topic:          config.Topic,
			GroupID:        config.GroupID,
			Dialer:         dialer,
			MinBytes:       1,    // 1B
			MaxBytes:       10e6, // 10MB
			CommitInterval: 0,    // Disable auto-commit, use manual commit only
			StartOffset:    kafka.FirstOffset,
			MaxWait:        500 * time.Millisecond,
		}
		readers[i] = kafka.NewReader(readerConfig)
	}

	return &Consumer{
		readers:       readers,
		log:           log.NewHelper(logger),
		topic:         config.Topic,
		groupID:       config.GroupID,
		numPartitions: numPartitions,
		numWorkers:    numWorkers,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins consuming messages with multiple consumers and workers
func (c *Consumer) Start(handler MessageHandler) {
	c.log.Infof("Starting %d consumers x %d workers = %d total workers for topic: %s, group: %s",
		c.numPartitions, c.numWorkers, c.numPartitions*c.numWorkers, c.topic, c.groupID)

	// For each consumer (partition)
	for consumerID := 0; consumerID < c.numPartitions; consumerID++ {
		// For each worker in this consumer
		for workerID := 0; workerID < c.numWorkers; workerID++ {
			c.wg.Add(1)
			go c.worker(consumerID, workerID, c.readers[consumerID], handler)
		}
	}
}

// worker processes messages from Kafka
func (c *Consumer) worker(consumerID int, workerID int, reader *kafka.Reader, handler MessageHandler) {
	defer c.wg.Done()

	workerName := fmt.Sprintf("C%d-W%d", consumerID, workerID)
	c.log.Infof("[Consumer] %s started for topic=%s group=%s", workerName, c.topic, c.groupID)

	for {
		select {
		case <-c.ctx.Done():
			c.log.Infof("[Consumer] %s stopping for topic=%s", workerName, c.topic)
			return
		default:
			msg, err := reader.FetchMessage(c.ctx)
			if err != nil {
				if err == context.Canceled {
					return
				}
				c.log.Errorf("[Consumer] %s failed to fetch message from topic=%s: %v", workerName, c.topic, err)
				continue
			}

			// Attempt to extract region from message payload (best-effort) so logs can include region.
			region := "unknown"
			var payload map[string]interface{}
			if err := json.Unmarshal(msg.Value, &payload); err == nil {
				if r, ok := payload["region"].(string); ok && r != "" {
					region = r
				}
			}

			// Log message received (include region if available)
			c.log.Infof("[Consumer][%s] %s received message: topic=%s partition=%d offset=%d key=%s",
				region, workerName, c.topic, msg.Partition, msg.Offset, string(msg.Key))

			// Process the message
			if err := handler(c.ctx, msg); err != nil {
				c.log.Errorf("[Consumer][%s] %s processing failed: topic=%s partition=%d offset=%d key=%s error=%v",
					region, workerName, c.topic, msg.Partition, msg.Offset, string(msg.Key), err)
				// Still commit to avoid infinite retry loop
			} else {
				c.log.Infof("[Consumer][%s] %s processing success: topic=%s partition=%d offset=%d key=%s",
					region, workerName, c.topic, msg.Partition, msg.Offset, string(msg.Key))
			}

			// Commit the message after processing (success or failure)
			if err := reader.CommitMessages(c.ctx, msg); err != nil {
				c.log.Errorf("[Consumer][%s] %s commit failed: topic=%s partition=%d offset=%d key=%s error=%v",
					region, workerName, c.topic, msg.Partition, msg.Offset, string(msg.Key), err)
			} else {
				c.log.Infof("[Consumer][%s] %s committed: topic=%s partition=%d offset=%d key=%s",
					region, workerName, c.topic, msg.Partition, msg.Offset, string(msg.Key))
			}
		}
	}
}

// Stop gracefully stops all workers and consumers
func (c *Consumer) Stop() error {
	c.log.Infof("Stopping all consumers for topic: %s", c.topic)
	c.cancel()
	c.wg.Wait()

	// Close all readers
	for i, reader := range c.readers {
		if err := reader.Close(); err != nil {
			c.log.Errorf("Failed to close kafka reader %d: %v", i, err)
		}
	}

	c.log.Infof("All consumers stopped for topic: %s", c.topic)
	return nil
}
