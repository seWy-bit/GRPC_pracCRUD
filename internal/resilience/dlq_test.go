package resilience

import (
	"errors"
	"testing"
)

func TestDeadLetterQueue(t *testing.T) {
	dlq := NewDeadLetterQueue()
	messages := []string{"msg1", "msg2", "msg3"}

	ProcessWithDLQ(messages, func(msg string) error {
		if msg == "msg2" {
			return errors.New("processing failed")
		}
		return nil
	}, dlq)

	result := dlq.GetMessages()
	if len(result) != 1 {
		t.Errorf("Expected 1 message in DLQ, got %d", len(result))
	}
	if result[0] != "msg2" {
		t.Errorf("Expected 'msg2' in DLQ, got '%s'", result[0])
	}
}

func TestEmptyDLQ(t *testing.T) {
	dlq := NewDeadLetterQueue()
	messages := []string{"msg1", "msg2"}

	ProcessWithDLQ(messages, func(msg string) error {
		return nil
	}, dlq)

	if len(dlq.GetMessages()) != 0 {
		t.Errorf("DLQ should be empty")
	}
}
