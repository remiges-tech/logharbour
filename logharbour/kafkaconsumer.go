package logharbour

import (
	"github.com/IBM/sarama"
)

// MessageHandler is a function type that processes messages from Kafka.
type MessageHandler func(messages []*sarama.ConsumerMessage) error

// Consumer defines the interface for a Kafka consumer.
type Consumer interface {
	Start(batchSize int) (<-chan error, error)
	Stop() error
}

type kafkaConsumer struct {
	consumer sarama.Consumer
	topic    string
	handler  MessageHandler
}

func NewConsumer(brokers []string, topic string, handler MessageHandler) (Consumer, error) {
	// Initialize Kafka consumer configuration
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Consumer.Return.Errors = true

	// Create a new consumer
	consumer, err := sarama.NewConsumer(brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	return &kafkaConsumer{
		consumer: consumer,
		topic:    topic,
		handler:  handler,
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
	pc, err := kc.consumer.ConsumePartition(kc.topic, partition, sarama.OffsetOldest)
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
