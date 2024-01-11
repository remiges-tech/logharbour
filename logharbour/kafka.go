package logharbour

import (
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
	topic    string
}

func (sw *kafkaWriter) Write(p []byte) (n int, err error) {
	msg := &sarama.ProducerMessage{
		Topic: sw.topic,
		Value: sarama.ByteEncoder(p),
	}

	_, _, err = sw.producer.SendMessage(msg)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (sw *kafkaWriter) Flush() error {
	// Sarama does not have a Flush method, as it sends messages as soon as they are available.
	return nil
}

func (sw *kafkaWriter) Configure(config KafkaConfig) error {
	// Implement this method based on your requirements.
	return nil
}

func (sw *kafkaWriter) Close() error {
	return sw.producer.Close()
}

func NewKafkaWriter(config KafkaConfig) (KafkaWriter, error) {
	producer, err := sarama.NewSyncProducer(config.Brokers, nil)
	if err != nil {
		return nil, err
	}

	return &kafkaWriter{
		producer: producer,
		topic:    config.Topic,
	}, nil
}
