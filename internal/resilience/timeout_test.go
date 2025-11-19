package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestTimeout_Success(t *testing.T) {
	op := func() error {
		time.Sleep(50 * time.Millisecond)
		return nil
	}

	err := Timeout(op, 100)

	if err != nil {
		t.Errorf("should complete in time")
	}
}

func TestTimeout_Timeout(t *testing.T) {
	op := func() error {
		time.Sleep(200 * time.Millisecond)
		return nil
	}

	err := Timeout(op, 100)

	if err == nil {
		t.Errorf("should timeout")
	}
}

func TestTimeout_Error(t *testing.T) {
	op := func() error {
		return errors.New("operation error")
	}

	err := Timeout(op, 100)

	if err == nil {
		t.Errorf("should return error")
	}
}
