package resilience

import (
	"fmt"
	"time"
)

func Retry(operation func() error, maxRetries int, baseDelay time.Duration) error {
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		if i == maxRetries-1 {
			break
		}

		delay := baseDelay * time.Duration(1<<uint(i))
		time.Sleep(delay)
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
