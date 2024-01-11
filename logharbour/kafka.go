package logharbour

import (
	"errors"
	"sync"

	"github.com/IBM/sarama"
)

type KafkaConfig struct {
	Brokers []string
	Topic   string
	// Include other Kafka producer configurations as needed
}

type KafkaWriter interface {
	Write(p []byte) (n int, err error)
	Flush() error
	Close() error

	// Errors() <-chan error
	Configure(config KafkaConfig) error
}

type kafkaWriter struct {
	producer sarama.SyncProducer
	pool     *KafkaWriterPool
	closed   bool
	mu       sync.Mutex
	topic    string
}

func (kw *kafkaWriter) Write(p []byte) (n int, err error) {
	writer, err := kw.pool.Get()
	if err != nil {
		return 0, err
	}
	defer kw.pool.Put(writer)

	msg := &sarama.ProducerMessage{
		Topic: kw.topic,
		Value: sarama.ByteEncoder(p),
	}

	_, _, err = writer.producer.SendMessage(msg)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (kw *kafkaWriter) Flush() error {
	// Sarama does not have a Flush method, as it sends messages as soon as they are available.
	return nil
}

func (kw *kafkaWriter) Configure(config KafkaConfig) error {
	// Implement this method based on your requirements.
	return nil
}

func (kw *kafkaWriter) Close() error {
	kw.mu.Lock()
	defer kw.mu.Unlock()

	if kw.closed {
		return nil
	}

	err := kw.producer.Close()
	if err == nil {
		kw.closed = true
	}

	return err
}

func NewKafkaWriter(config KafkaConfig, poolSize int) (KafkaWriter, error) {
	writerPool, err := NewKafkaWriterPool(config, poolSize)
	if err != nil {
		return nil, err
	}

	return &kafkaWriter{
		pool:  writerPool,
		topic: config.Topic,
	}, nil
}

///////

type KafkaWriterPool struct {
	pool   chan *kafkaWriter
	config KafkaConfig
	mu     sync.Mutex
}

func NewKafkaWriterPool(config KafkaConfig, size int) (*KafkaWriterPool, error) {
	pool := make(chan *kafkaWriter, size)
	for i := 0; i < size; i++ {
		producer, err := sarama.NewSyncProducer(config.Brokers, nil)
		if err != nil {
			return nil, err
		}

		writer := &kafkaWriter{
			producer: producer,
			topic:    config.Topic,
		}

		pool <- writer
	}

	return &KafkaWriterPool{
		pool:   pool,
		config: config,
	}, nil
}

func (p *KafkaWriterPool) Get() (*kafkaWriter, error) {
	select {
	case writer := <-p.pool:
		return writer, nil
	default:
		return nil, errors.New("no available Kafka writers")
	}
}

func (p *KafkaWriterPool) Put(writer *kafkaWriter) {
	p.pool <- writer
}

func (p *KafkaWriterPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	close(p.pool)
	for writer := range p.pool {
		if err := writer.Close(); err != nil {
			return err
		}
	}

	return nil
}
