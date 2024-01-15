package logharbour

import (
	"time"

	"github.com/IBM/sarama"
)

const defaultPoolSize = 10 // Set your default pool size

// KafkaConfig is a struct that holds the configuration for a Kafka producer.
// All fields are pointers, which allows us to distinguish between a field that is not set and a field set with its zero value.
type KafkaConfig struct {
	Brokers []string // List of broker addresses
	Topic   string   // Kafka topic to write messages to

	// Producer configurations
	Retries          *int                 // Maximum number of times to retry sending a message
	RequiredAcks     *sarama.RequiredAcks // Number of acknowledgments required before considering a message as sent
	Timeout          *time.Duration       // Maximum duration to wait for the broker to acknowledge the receipt of a message
	ReturnErrors     *bool                // Whether to return errors that occurred while producing the message
	ReturnSuccesses  *bool                // Whether to return successes of produced messages
	CompressionLevel *int                 // Compression level to use for messages

	// Network configurations
	DialTimeout     *time.Duration // Timeout for establishing network connections
	ReadTimeout     *time.Duration // Timeout for network reads
	WriteTimeout    *time.Duration // Timeout for network writes
	MaxOpenRequests *int           // Maximum number of unacknowledged requests to send before blocking

	// Client configurations
	ClientID *string // User-provided string sent with every request for logging, debugging, and auditing purposes
}

// KafkaWriter defines methods for Kafka writer
type KafkaWriter interface {
	Write(p []byte) (n int, err error)
	Close() error
}

type KafkaWriterOption func(*kafkaWriter)

func WithPoolSize(size int) KafkaWriterOption {
	return func(kw *kafkaWriter) {
		kw.pool.poolSize = size
	}
}

func NewKafkaWriter(kafkaConfig KafkaConfig, opts ...KafkaWriterOption) (KafkaWriter, error) {
	pool, err := newKafkaConnectionPool(defaultPoolSize, kafkaConfig)
	if err != nil {
		return nil, err
	}
	kw := &kafkaWriter{
		pool:  pool,
		topic: kafkaConfig.Topic,
	}
	for _, opt := range opts {
		opt(kw)
	}
	return kw, nil
}

type kafkaWriter struct {
	pool  *kafkaConnectionPool
	topic string
}

// Write sends a message to a Kafka topic. It implements io.Writer.
// It works with kafkaConnectionPool.
// It retrieves a connection from the pool and releases it back to the pool after use.
func (kw *kafkaWriter) Write(p []byte) (n int, err error) {
	producer := kw.pool.getConnection()
	defer kw.pool.releaseConnection(producer)

	msg := &sarama.ProducerMessage{
		Topic: kw.topic,
		Value: sarama.ByteEncoder(p),
	}

	_, _, err = producer.SendMessage(msg)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

// Close is used to close the writer and conforms to the io.Closer.
// It iterates over all connections in the pool and closes them.
// If there is an error in closing a connection, it returns the error immediately without closing the remaining connections.
// If all connections are successfully closed, it returns nil.
func (kw *kafkaWriter) Close() error {
	for _, producer := range kw.pool.all {
		if err := producer.Close(); err != nil {
			return err
		}
	}
	return nil
}

///// pool

// kafkaConnectionPool represents a pool of connections to a Kafka cluster.
// It maintains a channel of sarama.SyncProducer instances, which are used to send messages to Kafka.
// The maxConnections field specifies the maximum number of connections that can be open at the same time.
// The all field keeps track of all connections created, regardless of whether they are currently in use or in the pool.
// This is necessary because when closing the Kafka writer, we need to ensure that all connections are closed.
// If we only had the connections channel, we could only close the connections currently in the pool, not the ones in use.
type kafkaConnectionPool struct {
	connections chan sarama.SyncProducer
	all         []sarama.SyncProducer
	poolSize    int
}

func newKafkaConnectionPool(poolSize int, kafkaConfig KafkaConfig) (*kafkaConnectionPool, error) {
	pool := &kafkaConnectionPool{
		connections: make(chan sarama.SyncProducer, poolSize),
		all:         make([]sarama.SyncProducer, 0, poolSize),
		poolSize:    poolSize,
	}

	saramaConfig := applyKafkaConfig(kafkaConfig)

	for i := 0; i < poolSize; i++ {
		producer, err := sarama.NewSyncProducer(kafkaConfig.Brokers, saramaConfig)
		if err != nil {
			return nil, err
		}
		// Add the newly created producer to the pool (connections channel)
		pool.connections <- producer
		// And also add it to the all slice
		pool.all = append(pool.all, producer)
	}

	return pool, nil
}

// getConnection retrieves a connection from the pool.
// If all connections are currently in use, this method will block until a connection is released back into the pool.
func (pool *kafkaConnectionPool) getConnection() sarama.SyncProducer {
	return <-pool.connections
}

// releaseConnection releases a connection back into the pool, making it available for reuse.
// This method should be called after a connection is no longer needed, to allow other goroutines to use it.
func (pool *kafkaConnectionPool) releaseConnection(producer sarama.SyncProducer) {
	pool.connections <- producer
}

// ApplyKafkaConfig takes a KafkaConfig struct as input and returns a sarama.Config instance.
// It applies the settings from the KafkaConfig to the sarama.Config instance, including producer, network, and client configurations.
func applyKafkaConfig(kafkaConfig KafkaConfig) *sarama.Config {
	saramaConfig := sarama.NewConfig()

	// Set defaults as per our requirements

	// Set Producer.Return.Successes to true by default as it is required for sync producer
	saramaConfig.Producer.Return.Successes = true

	if kafkaConfig.Retries != nil {
		saramaConfig.Producer.Retry.Max = *kafkaConfig.Retries
	}

	if kafkaConfig.RequiredAcks != nil {
		saramaConfig.Producer.RequiredAcks = *kafkaConfig.RequiredAcks
	}

	if kafkaConfig.Timeout != nil {
		saramaConfig.Producer.Timeout = *kafkaConfig.Timeout
	}

	if kafkaConfig.ReturnErrors != nil {
		saramaConfig.Producer.Return.Errors = *kafkaConfig.ReturnErrors
	}

	if kafkaConfig.ReturnSuccesses != nil {
		saramaConfig.Producer.Return.Successes = *kafkaConfig.ReturnSuccesses
	}

	if kafkaConfig.CompressionLevel != nil {
		saramaConfig.Producer.CompressionLevel = *kafkaConfig.CompressionLevel
	}

	if kafkaConfig.DialTimeout != nil {
		saramaConfig.Net.DialTimeout = *kafkaConfig.DialTimeout
	}

	if kafkaConfig.ReadTimeout != nil {
		saramaConfig.Net.ReadTimeout = *kafkaConfig.ReadTimeout
	}

	if kafkaConfig.WriteTimeout != nil {
		saramaConfig.Net.WriteTimeout = *kafkaConfig.WriteTimeout
	}

	if kafkaConfig.MaxOpenRequests != nil {
		saramaConfig.Net.MaxOpenRequests = *kafkaConfig.MaxOpenRequests
	}

	if kafkaConfig.ClientID != nil {
		saramaConfig.ClientID = *kafkaConfig.ClientID
	}

	return saramaConfig
}
