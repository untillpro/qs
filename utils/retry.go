/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/untillpro/goutils/logger"
)

// RetryConfig holds configuration for retry operations
type RetryConfig struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Backoff      func(attempt int, delay time.Duration) time.Duration
}

// getMaxRetries returns the maximum number of retries from environment or default
func getMaxRetries() int {
	if envVal := os.Getenv(maxRetriesEnv); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val >= 0 {
			return val
		}
		logger.Verbose(fmt.Sprintf("Invalid %s value: %s, using default: %d", maxRetriesEnv, envVal, defaultMaxRetries))
	}

	return defaultMaxRetries
}

// getRetryDelay returns the initial retry delay from environment or default
func getRetryDelay() time.Duration {
	if envVal := os.Getenv(retryDelayMsEnv); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			return time.Duration(val) * time.Millisecond
		}
		logger.Verbose(fmt.Sprintf("Invalid %s value: %s, using default: %v", retryDelayMsEnv, envVal, defaultRetryDelay))
	}

	return defaultRetryDelay
}

// getMaxRetryDelay returns the maximum retry delay from environment or default
func getMaxRetryDelay() time.Duration {
	if envVal := os.Getenv(maxRetryDelayMsEnv); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			return time.Duration(val) * time.Millisecond
		}
		logger.Verbose(fmt.Sprintf("Invalid %s value: %s, using default: %v", maxRetryDelayMsEnv, envVal, defaultMaxRetryDelay))
	}

	return defaultMaxRetryDelay
}

// DefaultRetryConfig returns a default retry configuration with environment variable support
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:   getMaxRetries(),
		InitialDelay: getRetryDelay(),
		MaxDelay:     getMaxRetryDelay(),
		Backoff:      ExponentialBackoff,
	}
}

// ExponentialBackoff implements exponential backoff with jitter
func ExponentialBackoff(attempt int, delay time.Duration) time.Duration {
	newDelay := delay * time.Duration(1<<attempt)
	maxDelay := getMaxRetryDelay()
	if newDelay > maxDelay {
		return maxDelay
	}

	return newDelay
}

// LinearBackoff implements linear backoff
func LinearBackoff(attempt int, delay time.Duration) time.Duration {
	newDelay := delay * time.Duration(attempt+1)
	maxDelay := getMaxRetryDelay()
	if newDelay > maxDelay {
		return maxDelay
	}

	return newDelay
}

// RetryWithConfig executes a function with retry logic using the provided configuration
func RetryWithConfig(fn func() error, config *RetryConfig) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := config.Backoff(attempt-1, config.InitialDelay)
			logger.Verbose(fmt.Sprintf("Retry attempt %d/%d, waiting %v before retry", attempt, config.MaxRetries, delay))
			time.Sleep(delay)
		}

		lastErr = fn()
		if lastErr == nil {
			if attempt > 0 {
				logger.Verbose(fmt.Sprintf("Operation succeeded on attempt %d", attempt+1))
			}

			return nil
		}

		if attempt < config.MaxRetries {
			logger.Verbose(fmt.Sprintf("Attempt %d failed: %v", attempt+1, lastErr))
		}
	}

	return fmt.Errorf("operation failed after %d attempts, last error: %w", config.MaxRetries+1, lastErr)
}

// Retry executes a function with default retry logic
func Retry(fn func() error) error {
	return RetryWithConfig(fn, DefaultRetryConfig())
}

// RetryConfigWithMaxAttempts creates a retry config with custom max attempts but environment-based delays
func RetryConfigWithMaxAttempts(maxAttempts int) *RetryConfig {
	return &RetryConfig{
		MaxRetries:   maxAttempts - 1, // MaxRetries is additional attempts beyond the first
		InitialDelay: getRetryDelay(),
		MaxDelay:     getMaxRetryDelay(),
		Backoff:      ExponentialBackoff,
	}
}

// RetryWithMaxAttempts executes a function with specified maximum attempts
func RetryWithMaxAttempts(fn func() error, maxAttempts int) error {
	config := RetryConfigWithMaxAttempts(maxAttempts)

	return RetryWithConfig(fn, config)
}
