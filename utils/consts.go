/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import "time"

const (
	defaultGhTimeoutMs = 1500
	ghTimeoutMsEnv     = "GH_TIMEOUT_MS"

	// Retry configuration constants
	defaultMaxRetries    = 3
	defaultRetryDelay    = 2 * time.Second
	defaultMaxRetryDelay = 30 * time.Second

	// Retry configuration environment variables
	maxRetriesEnv      = "QS_MAX_RETRIES"
	retryDelayMsEnv    = "QS_RETRY_DELAY_MS"
	maxRetryDelayMsEnv = "QS_MAX_RETRY_DELAY_MS"

	RefsNotes = "refs/notes/*:refs/notes/*"
)
