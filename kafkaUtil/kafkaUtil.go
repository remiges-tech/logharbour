// this utility class is used to write log messages from logHarbour to Kafka
package logHarbour

import (
	"context"
	"log/slog"
	"os"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

var KafkaProducer *kafka.Producer
var topic = "logHarbour"
var logId []byte

var defaultLogger *slog.Logger
var ctx context.Context
var programLevel = new(slog.LevelVar) // Info by default

// initialize Kafka Producer with a log ID which will be a key to all messages written to Kafka.
// This key should ideally be combination of appName+moduleName+systemName
func KafkaInit(appName, moduleName, systemName string) {
	logId = []byte(appName + "_" + moduleName + "_" + systemName)
	programLevel.Set(0) //TODO refer to constants
	initDefaultLogger(appName, moduleName, systemName)
	kafkaStart()
}

func initDefaultLogger(appName, moduleName, systemName string) {
	defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: programLevel})).
		With("handle", "DEFAULT_LOGGER").With("app", appName).With("module", moduleName).With("system", systemName)
}

// start kafka producer.
func kafkaStart() {
	KafkaProducer, _ = kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": "127.0.0.1:9092"}) //TODO read kafka server details from RIGEL
	//fmt.Println("->> New Kafka Producer <<-")
	go func() {
		//fmt.Println("->> New Kafka Event Listener <<-")
		for e := range KafkaProducer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					defaultLogger.Error("Failed to deliver message:", "TopicPartition", ev.TopicPartition)
				} else {
					//TODO: check log level from Rigel to print this
					if getRigelLogLevel() <= -4 {
						defaultLogger.Log(ctx, slog.Level(-4), "Produced event to topic",
							"TopicPartition", *ev.TopicPartition.Topic,
							"key", string(ev.Key), "value", string(ev.Value))
					}

				}
			}
		}
	}()
	//TODO: decide on this flush timeout
	KafkaProducer.Flush(100)
}

func sendMsgToKafka(msg []byte) {

	if KafkaProducer == nil {
		kafkaStart()
	}
	//deliveryChan := make(chan kafka.Event)

	message := &kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          msg,
		Key:            logId, //logId ideally should be a combination of appName+moduleName+systemName
	}

	// produce the message
	KafkaProducer.Produce(message, nil)
	//defer KafkaProducer.Close()
}

type KafkaWriter struct {
}

func (e KafkaWriter) Write(msg []byte) (int, error) {
	sendMsgToKafka(msg)
	return len(msg), nil
}

// TODO: STUB func to get log level from Rigel
func getRigelLogLevel() slog.Level {
	//TODO : change this with corresponding log level call from Rigel
	//Else, by default, it returns the log level of slog
	return programLevel.Level()
}
