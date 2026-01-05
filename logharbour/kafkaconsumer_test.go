package logharbour

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/IBM/sarama"
)

// mockConsumerGroupSession tracks MarkMessage calls
type mockConsumerGroupSession struct {
	markedMessages []*sarama.ConsumerMessage
	mu             sync.Mutex
	ctx            context.Context
	cancel         context.CancelFunc
}

func newMockSession() *mockConsumerGroupSession {
	ctx, cancel := context.WithCancel(context.Background())
	return &mockConsumerGroupSession{
		markedMessages: make([]*sarama.ConsumerMessage, 0),
		ctx:            ctx,
		cancel:         cancel,
	}
}

func (m *mockConsumerGroupSession) Claims() map[string][]int32 {
	return nil
}

func (m *mockConsumerGroupSession) MemberID() string {
	return "test-member"
}

func (m *mockConsumerGroupSession) GenerationID() int32 {
	return 1
}

func (m *mockConsumerGroupSession) MarkOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockConsumerGroupSession) Commit() {
}

func (m *mockConsumerGroupSession) ResetOffset(topic string, partition int32, offset int64, metadata string) {
}

func (m *mockConsumerGroupSession) MarkMessage(msg *sarama.ConsumerMessage, metadata string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.markedMessages = append(m.markedMessages, msg)
}

func (m *mockConsumerGroupSession) Context() context.Context {
	return m.ctx
}

func (m *mockConsumerGroupSession) getMarkedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.markedMessages)
}

// mockConsumerGroupClaim provides messages for testing
type mockConsumerGroupClaim struct {
	messages  chan *sarama.ConsumerMessage
	topic     string
	partition int32
}

func newMockClaim(topic string, partition int32) *mockConsumerGroupClaim {
	return &mockConsumerGroupClaim{
		messages:  make(chan *sarama.ConsumerMessage, 100),
		topic:     topic,
		partition: partition,
	}
}

func (m *mockConsumerGroupClaim) Topic() string {
	return m.topic
}

func (m *mockConsumerGroupClaim) Partition() int32 {
	return m.partition
}

func (m *mockConsumerGroupClaim) InitialOffset() int64 {
	return 0
}

func (m *mockConsumerGroupClaim) HighWaterMarkOffset() int64 {
	return 0
}

func (m *mockConsumerGroupClaim) Messages() <-chan *sarama.ConsumerMessage {
	return m.messages
}

func TestConsumerGroupHandler_MarkAfterHandlerSuccess(t *testing.T) {
	session := newMockSession()
	claim := newMockClaim("test-topic", 0)

	handlerCalled := false
	handler := &ConsumerGroupHandler{
		handler: func(messages []*sarama.ConsumerMessage) error {
			handlerCalled = true
			return nil
		},
		batchSize:    2,
		batchTimeout: 100 * time.Millisecond,
	}

	// Send 2 messages to trigger full batch
	claim.messages <- &sarama.ConsumerMessage{Offset: 1, Topic: "test-topic", Partition: 0}
	claim.messages <- &sarama.ConsumerMessage{Offset: 2, Topic: "test-topic", Partition: 0}

	// Close channel to signal end
	go func() {
		time.Sleep(50 * time.Millisecond)
		close(claim.messages)
	}()

	err := handler.ConsumeClaim(session, claim)
	if err != nil {
		t.Fatalf("ConsumeClaim returned error: %v", err)
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	markedCount := session.getMarkedCount()
	if markedCount != 2 {
		t.Errorf("Expected 2 messages marked, got %d", markedCount)
	}
}

func TestConsumerGroupHandler_NoMarkOnHandlerFailure(t *testing.T) {
	session := newMockSession()
	claim := newMockClaim("test-topic", 0)

	handlerError := errors.New("handler failed")
	handler := &ConsumerGroupHandler{
		handler: func(messages []*sarama.ConsumerMessage) error {
			return handlerError
		},
		batchSize:    2,
		batchTimeout: 100 * time.Millisecond,
	}

	// Send 2 messages to trigger full batch
	claim.messages <- &sarama.ConsumerMessage{Offset: 1, Topic: "test-topic", Partition: 0}
	claim.messages <- &sarama.ConsumerMessage{Offset: 2, Topic: "test-topic", Partition: 0}

	err := handler.ConsumeClaim(session, claim)
	if err != handlerError {
		t.Fatalf("Expected handler error, got: %v", err)
	}

	markedCount := session.getMarkedCount()
	if markedCount != 0 {
		t.Errorf("Expected 0 messages marked on handler failure, got %d", markedCount)
	}
}

func TestConsumerGroupHandler_TimeoutBatchMarkAfterSuccess(t *testing.T) {
	session := newMockSession()
	claim := newMockClaim("test-topic", 0)

	handlerCalled := false
	handler := &ConsumerGroupHandler{
		handler: func(messages []*sarama.ConsumerMessage) error {
			handlerCalled = true
			return nil
		},
		batchSize:    10, // Large batch size so timeout triggers
		batchTimeout: 50 * time.Millisecond,
	}

	// Send 1 message (less than batch size)
	claim.messages <- &sarama.ConsumerMessage{Offset: 1, Topic: "test-topic", Partition: 0}

	// Cancel context after timeout to end the test
	go func() {
		time.Sleep(100 * time.Millisecond)
		session.cancel()
	}()

	err := handler.ConsumeClaim(session, claim)
	if err != nil {
		t.Fatalf("ConsumeClaim returned error: %v", err)
	}

	if !handlerCalled {
		t.Error("Handler was not called on timeout")
	}

	markedCount := session.getMarkedCount()
	if markedCount != 1 {
		t.Errorf("Expected 1 message marked after timeout, got %d", markedCount)
	}
}

func TestConsumerGroupHandler_TimeoutBatchNoMarkOnFailure(t *testing.T) {
	session := newMockSession()
	claim := newMockClaim("test-topic", 0)

	handlerError := errors.New("handler failed on timeout batch")
	handler := &ConsumerGroupHandler{
		handler: func(messages []*sarama.ConsumerMessage) error {
			return handlerError
		},
		batchSize:    10, // Large batch size so timeout triggers
		batchTimeout: 50 * time.Millisecond,
	}

	// Send 1 message (less than batch size)
	claim.messages <- &sarama.ConsumerMessage{Offset: 1, Topic: "test-topic", Partition: 0}

	err := handler.ConsumeClaim(session, claim)
	if err != handlerError {
		t.Fatalf("Expected handler error, got: %v", err)
	}

	markedCount := session.getMarkedCount()
	if markedCount != 0 {
		t.Errorf("Expected 0 messages marked on timeout batch failure, got %d", markedCount)
	}
}
