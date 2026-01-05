package main

import (
	"errors"
	"sync"
	"testing"

	"github.com/IBM/sarama"
	"github.com/remiges-tech/logharbour/logharbour"
	"github.com/stretchr/testify/require"
)

// mockSyncProducer implements sarama.SyncProducer for testing
type mockSyncProducer struct {
	sentMessages []*sarama.ProducerMessage
	returnError  error
	mu           sync.Mutex
}

func (m *mockSyncProducer) SendMessage(msg *sarama.ProducerMessage) (int32, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sentMessages = append(m.sentMessages, msg)
	return 0, 0, m.returnError
}

func (m *mockSyncProducer) SendMessages(msgs []*sarama.ProducerMessage) error {
	return nil
}

func (m *mockSyncProducer) Close() error {
	return nil
}

func (m *mockSyncProducer) TxnStatus() sarama.ProducerTxnStatusFlag {
	return sarama.ProducerTxnFlagReady
}

func (m *mockSyncProducer) IsTransactional() bool {
	return false
}

func (m *mockSyncProducer) BeginTxn() error {
	return nil
}

func (m *mockSyncProducer) CommitTxn() error {
	return nil
}

func (m *mockSyncProducer) AbortTxn() error {
	return nil
}

func (m *mockSyncProducer) AddOffsetsToTxn(offsets map[string][]*sarama.PartitionOffsetMetadata, groupId string) error {
	return nil
}

func (m *mockSyncProducer) AddMessageToTxn(msg *sarama.ConsumerMessage, groupId string, metadata *string) error {
	return nil
}

// findHeader returns the value of a header by key
func findHeader(headers []sarama.RecordHeader, key string) string {
	for _, h := range headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}
	return ""
}

// TestSendToDLQ_Success verifies message is sent to correct topic with correct value
func TestSendToDLQ_Success(t *testing.T) {
	setupLogger("info")

	originalMsg := &sarama.ConsumerMessage{
		Value:     []byte(`{"id":"test-doc-123","app":"myapp"}`),
		Topic:     "log_topic",
		Partition: 0,
		Offset:    123,
	}
	mockProducer := &mockSyncProducer{}

	sendToDLQ(mockProducer, "log_topic_dlq", originalMsg, "test_reason")

	require.Equal(t, 1, len(mockProducer.sentMessages))
	require.Equal(t, "log_topic_dlq", mockProducer.sentMessages[0].Topic)

	value, err := mockProducer.sentMessages[0].Value.Encode()
	require.NoError(t, err)
	require.Equal(t, originalMsg.Value, value)
}

// TestSendToDLQ_HeadersCorrect verifies all 4 headers are set correctly
func TestSendToDLQ_HeadersCorrect(t *testing.T) {
	setupLogger("info")

	originalMsg := &sarama.ConsumerMessage{
		Value:     []byte(`{"id":"test-doc-123","app":"myapp"}`),
		Topic:     "log_topic",
		Partition: 0,
		Offset:    123,
	}
	mockProducer := &mockSyncProducer{}

	sendToDLQ(mockProducer, "log_topic_dlq", originalMsg, "test_reason")

	require.Equal(t, 1, len(mockProducer.sentMessages))
	msg := mockProducer.sentMessages[0]

	require.Equal(t, "test_reason", findHeader(msg.Headers, "dlq_reason"))
	require.Equal(t, "log_topic", findHeader(msg.Headers, "original_topic"))
	require.Equal(t, "0", findHeader(msg.Headers, "original_partition"))
	require.Equal(t, "123", findHeader(msg.Headers, "original_offset"))
}

// TestSendToDLQ_ProducerError verifies function handles Kafka failures gracefully
func TestSendToDLQ_ProducerError(t *testing.T) {
	setupLogger("info")

	mockProducer := &mockSyncProducer{
		returnError: errors.New("kafka unavailable"),
	}
	originalMsg := &sarama.ConsumerMessage{
		Value:     []byte(`{"id":"test"}`),
		Topic:     "log_topic",
		Partition: 0,
		Offset:    123,
	}

	// Should not panic
	sendToDLQ(mockProducer, "log_topic_dlq", originalMsg, "test_reason")

	require.Equal(t, 1, len(mockProducer.sentMessages))
}

// TestSendToDLQ_IndexingErrorReason verifies ES indexing errors are captured in the reason header
func TestSendToDLQ_IndexingErrorReason(t *testing.T) {
	setupLogger("info")

	mockProducer := &mockSyncProducer{}
	originalMsg := &sarama.ConsumerMessage{
		Value:     []byte(`{"id":"test","field":"wrong_type"}`),
		Topic:     "log_topic",
		Partition: 0,
		Offset:    456,
	}

	sendToDLQ(mockProducer, "log_topic_dlq", originalMsg, "indexing_error: mapper_parsing_exception")

	require.Equal(t, 1, len(mockProducer.sentMessages))
	msg := mockProducer.sentMessages[0]
	require.Equal(t, "indexing_error: mapper_parsing_exception", findHeader(msg.Headers, "dlq_reason"))
}

// TestHandleIndexingFailures_SendsAllFailedDocs verifies all failed documents are sent to DLQ
func TestHandleIndexingFailures_SendsAllFailedDocs(t *testing.T) {
	setupLogger("info")

	errors := []logharbour.BulkError{
		{DocumentID: "doc1", Error: "mapping error"},
		{DocumentID: "doc2", Error: "field type error"},
		{DocumentID: "doc3", Error: "parsing error"},
	}
	docIDToMessage := map[string]*sarama.ConsumerMessage{
		"doc1": {Value: []byte(`{"id":"doc1"}`), Topic: "log_topic", Offset: 1},
		"doc2": {Value: []byte(`{"id":"doc2"}`), Topic: "log_topic", Offset: 2},
		"doc3": {Value: []byte(`{"id":"doc3"}`), Topic: "log_topic", Offset: 3},
	}
	mockProducer := &mockSyncProducer{}

	count := handleIndexingFailures(errors, docIDToMessage, mockProducer, "log_topic_dlq")

	require.Equal(t, 3, count)
	require.Equal(t, 3, len(mockProducer.sentMessages))
}

// TestHandleIndexingFailures_SkipsMissingDocIDs verifies function handles missing doc IDs gracefully
func TestHandleIndexingFailures_SkipsMissingDocIDs(t *testing.T) {
	setupLogger("info")

	errors := []logharbour.BulkError{
		{DocumentID: "doc1", Error: "error1"},
		{DocumentID: "doc_not_in_map", Error: "error2"},
	}
	docIDToMessage := map[string]*sarama.ConsumerMessage{
		"doc1": {Value: []byte(`{"id":"doc1"}`), Topic: "log_topic", Offset: 1},
	}
	mockProducer := &mockSyncProducer{}

	count := handleIndexingFailures(errors, docIDToMessage, mockProducer, "log_topic_dlq")

	require.Equal(t, 1, count)
	require.Equal(t, 1, len(mockProducer.sentMessages))
}

// TestHandleIndexingFailures_CorrectReasonFormat verifies the reason header format
func TestHandleIndexingFailures_CorrectReasonFormat(t *testing.T) {
	setupLogger("info")

	errors := []logharbour.BulkError{
		{DocumentID: "doc1", Error: "mapper_parsing_exception"},
	}
	docIDToMessage := map[string]*sarama.ConsumerMessage{
		"doc1": {Value: []byte(`{"id":"doc1"}`), Topic: "log_topic", Offset: 1},
	}
	mockProducer := &mockSyncProducer{}

	handleIndexingFailures(errors, docIDToMessage, mockProducer, "log_topic_dlq")

	require.Equal(t, 1, len(mockProducer.sentMessages))
	msg := mockProducer.sentMessages[0]
	require.Equal(t, "indexing_error: mapper_parsing_exception", findHeader(msg.Headers, "dlq_reason"))
}

// TestHandleIndexingFailures_EmptyErrors verifies function handles empty error slice gracefully
func TestHandleIndexingFailures_EmptyErrors(t *testing.T) {
	setupLogger("info")

	errors := []logharbour.BulkError{}
	docIDToMessage := map[string]*sarama.ConsumerMessage{}
	mockProducer := &mockSyncProducer{}

	count := handleIndexingFailures(errors, docIDToMessage, mockProducer, "log_topic_dlq")

	require.Equal(t, 0, count)
	require.Equal(t, 0, len(mockProducer.sentMessages))
}
