package errorshandler

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func WithRetry(
	ctx context.Context,
	callback func() error,
	classifiers []Classifier,
	retryTime []time.Duration,
	maxRetries int,
) error {
	if maxRetries < 0 {
		return fmt.Errorf("maxRetries must be non-negative")
	}
	for i, classifier := range classifiers {
		if classifier == nil {
			return fmt.Errorf("classifier at index %d must not be nil", i)
		}
	}

	for attempt := 0; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := callback()
		if err == nil {
			return nil
		}
		if !shouldRetry(err, classifiers) {
			return err
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if attempt >= maxRetries {
			return err
		}
		if len(retryTime) == 0 {
			continue
		}

		index := min(attempt, len(retryTime)-1)
		timer := time.NewTimer(retryTime[index])
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

// shouldRetry решает можно ли повторять вызов колбэка,
// опросив все классификаторы из массива
func shouldRetry(err error, classifiers []Classifier) bool {
	retry := false
	if errors.Is(err, context.Canceled) {
		return false
	}
	for _, classifier := range classifiers {
		switch classifier.Classify(err) {
		case NonRetriable:
			return false
		case Retriable:
			retry = true
		default:
			continue
		}
	}
	return retry
}
