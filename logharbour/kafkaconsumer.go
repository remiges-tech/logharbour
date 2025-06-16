package logharbour

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"github.com/IBM/sarama"
)

// MessageHandler is a function type that processes messages from Kafka.
type MessageHandler func(messages []*sarama.ConsumerMessage) error

// OffsetType represents different offset options
type OffsetType string

const (
	// OffsetEarliest starts consuming from the oldest available message
	OffsetEarliest OffsetType = "earliest"
	// OffsetLatest starts consuming only new messages
	OffsetLatest OffsetType = "latest"
	// OffsetSpecific uses a specific offset (handled via int64 parameter)
	OffsetSpecific OffsetType = "specific"
)

// Consumer defines the interface for a Kafka consumer.
type Consumer interface {
	Start(batchSize int) (<-chan error, error)
	Stop() error
}

type kafkaConsumer struct {
	consumer sarama.Consumer
	topic    string
	handler  MessageHandler
	offset   int64 // The offset to start consuming from
}

type kafkaConsumerGroup struct {
	consumerGroup sarama.ConsumerGroup
	topic         string
	handler       MessageHandler
	batchSize     int
	ctx           context.Context
	cancel        context.CancelFunc
}

// ConsumerGroupHandler implements sarama.ConsumerGroupHandler interface
type ConsumerGroupHandler struct {
	handler   MessageHandler
	batchSize int
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// Panic recovery - log and return error to trigger rebalance
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic recovered in ConsumeClaim",
				slog.String("topic", claim.Topic()),
				slog.Int("partition", int(claim.Partition())),
				slog.Any("panic", r),
				slog.String("stack_trace", string(debug.Stack())))
			// Don't continue processing - let the consumer group handle the error
		}
	}()
	
	batch := make([]*sarama.ConsumerMessage, 0, h.batchSize)
	
	for {
		select {
		case message := <-claim.Messages():
			if message == nil {
				// Process remaining batch if any
				if len(batch) > 0 {
					if err := h.handler(batch); err != nil {
						return err
					}
				}
				return nil
			}
			
			batch = append(batch, message)
			session.MarkMessage(message, "")
			
			if len(batch) >= h.batchSize {
				if err := h.handler(batch); err != nil {
					slog.Error("Failed to handle batch", 
						slog.String("error", err.Error()),
						slog.Int("batch_size", len(batch)))
					return err
				}
				batch = batch[:0]
				
				// Log current offset information
				slog.Debug("Batch processed, current offset marked",
					slog.Int64("offset", message.Offset),
					slog.Int("partition", int(message.Partition)),
					slog.String("topic", message.Topic))
			}
		case <-session.Context().Done():
			// Process remaining batch if any
			if len(batch) > 0 {
				if err := h.handler(batch); err != nil {
					return err
				}
			}
			return nil
		}
	}
}

// NewConsumerGroup creates a new Kafka consumer group
func NewConsumerGroup(brokers []string, groupID, topic string, handler MessageHandler, offsetType OffsetType) (Consumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	
	// Explicitly enable auto-commit with a shorter interval
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Offsets.AutoCommit.Interval = 1 * time.Second
	
	// Set offset based on offsetType
	switch offsetType {
	case OffsetEarliest:
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	case OffsetLatest:
		config.Consumer.Offsets.Initial = sarama.OffsetNewest
	default:
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}
	
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, err
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &kafkaConsumerGroup{
		consumerGroup: consumerGroup,
		topic:         topic,
		handler:       handler,
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// NewConsumer creates a new Kafka consumer with the specified offset behavior
func NewConsumer(brokers []string, topic string, handler MessageHandler, offsetType OffsetType, specificOffset int64) (Consumer, error) {
	// Initialize Kafka consumer configuration
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Consumer.Return.Errors = true

	// Create a new consumer
	consumer, err := sarama.NewConsumer(brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	// Determine which offset to use
	var offset int64
	switch offsetType {
	case OffsetEarliest:
		offset = sarama.OffsetOldest
	case OffsetLatest:
		offset = sarama.OffsetNewest
	case OffsetSpecific:
		// Use specific offset if provided, otherwise default to oldest
		if specificOffset >= 0 {
			offset = specificOffset
		} else {
			offset = sarama.OffsetOldest
		}
	default:
		// Default to oldest
		offset = sarama.OffsetOldest
	}

	return &kafkaConsumer{
		consumer: consumer,
		topic:    topic,
		handler:  handler,
		offset:   offset,
	}, nil
}

// Start begins consuming messages from all partitions of the Kafka topic concurrently.
// It creates a separate goroutine for each partition to allow for simultaneous processing of messages.
// This can significantly improve throughput, especially for topics with multiple partitions.
// The method takes a batchSize parameter, which specifies the number of messages to accumulate
// before processing them as a batch. Once the batch size is reached, the handler function is
// called to process the batch of messages.
// The method returns a channel that emits errors encountered during message consumption. This allows
// the caller to handle these errors asynchronously. The channel should be continuously read from
// to prevent blocking the consumer.
func (kc *kafkaConsumer) Start(batchSize int) (<-chan error, error) {
	errs := make(chan error)

	partitionList, err := kc.getPartitions()
	if err != nil {
		return nil, err
	}

	for _, partition := range partitionList {
		err := kc.consumePartition(partition, batchSize, errs)
		if err != nil {
			return nil, err
		}
	}

	return errs, nil
}

// getPartitions retrieves the list of partitions for the Kafka topic.
// This is necessary to start consuming messages from all partitions.
func (kc *kafkaConsumer) getPartitions() ([]int32, error) {
	return kc.consumer.Partitions(kc.topic)
}

// consumePartition starts consuming messages from a specific partition.
// It also starts a goroutine to process the messages from the partition.
// This allows for messages from multiple partitions to be processed simultaneously.
func (kc *kafkaConsumer) consumePartition(partition int32, batchSize int, errs chan error) error {
	// Use the configured offset instead of hardcoded sarama.OffsetOldest
	pc, err := kc.consumer.ConsumePartition(kc.topic, partition, kc.offset)
	if err != nil {
		return err
	}

	go kc.processMessages(pc, batchSize, errs)

	return nil
}

// processMessages processes messages from a partition.
// It creates batches of messages and calls handleBatch to process each batch.
// This allows for efficient processing of messages, especially when the handler function
// is designed to process batches of messages.
func (kc *kafkaConsumer) processMessages(pc sarama.PartitionConsumer, batchSize int, errs chan error) {
	// Panic recovery - log and exit the goroutine
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic in processMessages: %v\nStack trace:\n%s", r, debug.Stack())
			slog.Error("Panic recovered in partition consumer",
				slog.String("topic", kc.topic),
				slog.Any("panic", r),
				slog.String("stack_trace", string(debug.Stack())))
			
			// Send error to channel
			select {
			case errs <- err:
			default:
				slog.Error("Error channel full, could not send panic error")
			}
			
			// Exit the goroutine - don't continue processing
			return
		}
	}()
	
	batch := make([]*sarama.ConsumerMessage, 0, batchSize)
	for message := range pc.Messages() {
		batch = append(batch, message)
		if len(batch) >= batchSize {
			kc.handleBatch(batch, errs)
			batch = batch[:0]
		}
	}
	if len(batch) > 0 {
		kc.handleBatch(batch, errs)
	}
}

// handleBatch calls the handler function with a batch of messages and sends any errors to the error channel.
// This allows the caller to handle errors asynchronously and continue processing other batches.
func (kc *kafkaConsumer) handleBatch(batch []*sarama.ConsumerMessage, errs chan error) {
	if err := kc.handler(batch); err != nil {
		errs <- err
	}
}

func (kc *kafkaConsumer) Stop() error {
	if err := kc.consumer.Close(); err != nil {
		return err
	}
	return nil
}

// Start begins consuming messages using consumer group
func (kcg *kafkaConsumerGroup) Start(batchSize int) (<-chan error, error) {
	kcg.batchSize = batchSize
	errs := make(chan error, 1)
	
	handler := &ConsumerGroupHandler{
		handler:   kcg.handler,
		batchSize: batchSize,
	}
	
	go func() {
		defer close(errs)
		// Panic recovery - log and exit the goroutine
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in consumer group consumption: %v\nStack trace:\n%s", r, debug.Stack())
				slog.Error("Panic recovered in consumer group",
					slog.String("topic", kcg.topic),
					slog.Any("panic", r),
					slog.String("stack_trace", string(debug.Stack())))
				
				// Send error to channel before closing
				select {
				case errs <- err:
				default:
					slog.Error("Error channel full, could not send panic error")
				}
			}
		}()
		
		for {
			// Check if context is cancelled
			if kcg.ctx.Err() != nil {
				slog.Debug("Consumer group context cancelled, stopping consumption")
				return
			}
			
			slog.Debug("Consumer group starting consumption cycle", slog.String("topic", kcg.topic))
			if err := kcg.consumerGroup.Consume(kcg.ctx, []string{kcg.topic}, handler); err != nil {
				slog.Error("Error from consumer group", 
					slog.String("error", err.Error()),
					slog.String("topic", kcg.topic))
				errs <- err
				return
			}
		}
	}()
	
	// Handle consumer group errors
	go func() {
		// Panic recovery - log and exit the goroutine
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic in consumer group error handler: %v\nStack trace:\n%s", r, debug.Stack())
				slog.Error("Panic recovered in consumer group error handler",
					slog.String("topic", kcg.topic),
					slog.Any("panic", r),
					slog.String("stack_trace", string(debug.Stack())))
				
				// Send error to channel
				select {
				case errs <- err:
				default:
					slog.Error("Error channel full, could not send panic error")
				}
			}
		}()
		
		errorCount := 0
		for err := range kcg.consumerGroup.Errors() {
			errorCount++
			slog.Error("Consumer group error", 
				slog.String("error", err.Error()),
				slog.Int("error_count", errorCount))
			errs <- err
		}
	}()
	
	return errs, nil
}

func (kcg *kafkaConsumerGroup) Stop() error {
	kcg.cancel()
	return kcg.consumerGroup.Close()
}
