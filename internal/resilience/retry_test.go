package resilience

import (
	"errors"
	"testing"
	"time"
)

func TestRetry_Success(t *testing.T) {
	counter := 0

	op := func() error {
		counter++
		if counter < 3 {
			return errors.New("temp error")
		}

		return nil
	}

	err := Retry(op, 5, 10*time.Millisecond)

	if err != nil {
		t.Errorf("should succeed")
	}
	if counter != 3 {
		t.Errorf("should call 3 times")
	}
}

func TestRetry_Failure(t *testing.T) {
	op := func() error {
		return errors.New("permanent error")
	}

	err := Retry(op, 3, 10*time.Millisecond)

	if err == nil {
		t.Errorf("should fail")
	}
}
