/*
 * Copyright (c) 2026-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package utils

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
)

// GetUserEmail - github user email
func GetUserEmail() (string, error) {
	var stdout string
	err := Retry(func() error {
		var apiErr error
		stdout, _, apiErr = new(exec.PipedExec).
			Command("gh", "api", "user", "--jq", ".email").
			RunToStrings()
		return apiErr
	})

	return strings.TrimSpace(stdout), err
}

// DelayIfTest is a helper function to delay execution for a specified time.
// It reads the timeout from the environment variable GH_TIMEOUT_MS, defaulting to 1500 ms if not set.
func DelayIfTest() {
	if !testing.Testing() {
		return
	}

	var err error
	timeoutMs := defaultGhTimeoutMs

	ghTimeoutMsString := os.Getenv(ghTimeoutMsEnv)
	if ghTimeoutMsString != "" {
		timeoutMs, err = strconv.Atoi(ghTimeoutMsString)
		if err != nil {
			logger.Verbose(fmt.Sprintf("Error converting %s to int: %v", ghTimeoutMsString, err))
			timeoutMs = defaultGhTimeoutMs
		}
	}
	logger.Verbose(fmt.Sprintf("ghTimeoutMs: %d", timeoutMs))

	time.Sleep(time.Duration(timeoutMs) * time.Millisecond)

}

func CleanArgFromSpecSymbols(arg string) string {
	var symbol string

	arg = strings.ReplaceAll(arg, "https://", "")
	replaceToMinus := []string{" ", ",", ";", ".", ":", "?", "/", "!"}
	for _, symbol = range replaceToMinus {
		arg = strings.ReplaceAll(arg, symbol, "-")
	}
	replaceToNone := []string{"&", "$", "@", "%", "\\", "(", ")", "{", "}", "[", "]", "<", ">", "'", "\""}
	for _, symbol = range replaceToNone {
		arg = strings.ReplaceAll(arg, symbol, "")
	}
	for string(arg[0]) == "-" {
		arg = arg[1:]
	}

	arg = deleteDupMinus(arg)
	if len(arg) > 50 {
		arg = arg[:50]
	}
	for string(arg[len(arg)-1]) == "-" {
		arg = arg[:len(arg)-1]
	}
	return arg
}

func deleteDupMinus(str string) string {
	var buf bytes.Buffer
	var pc rune
	for _, c := range str {
		if pc == c && string(c) == "-" {
			continue
		}
		pc = c
		buf.WriteRune(c)
	}
	return buf.String()
}
