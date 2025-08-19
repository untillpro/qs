// Package helper provides utility functions for the qs application.
//
// Retry Configuration Environment Variables:
//   - QS_MAX_RETRIES: Maximum number of retry attempts (default: 3)
//   - QS_RETRY_DELAY_SECONDS: Initial delay between retries in seconds (default: 2)
//   - QS_MAX_RETRY_DELAY_SECONDS: Maximum delay between retries in seconds (default: 30)
//   - GH_TIMEOUT_MS: GitHub CLI timeout in milliseconds (default: 1500)
//
// Example usage:
//
//	export QS_MAX_RETRIES=5
//	export QS_RETRY_DELAY_SECONDS=3
//	export QS_MAX_RETRY_DELAY_SECONDS=60
package helper

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	osExec "os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/untillpro/goutils/exec"
	"github.com/untillpro/goutils/logger"
	"golang.org/x/mod/semver"
)

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
)

func IsTest() bool {
	return testing.Testing()
}

// Delay is a helper function to delay execution for a specified time.
// It reads the timeout from the environment variable GH_TIMEOUT_MS, defaulting to 1500 ms if not set.
func Delay() {
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

func CheckGH() bool {
	if !ghInstalled() {
		fmt.Print("\nGithub cli utility 'gh' is not installed.\nTo install visit page https://cli.github.com/\n")

		return false
	}
	if !ghLoggedIn() {
		fmt.Print("\nGH utility is not logged in\n")

		return false
	}

	return true
}

func CheckQsVer() bool {
	installedVer, err := GetInstalledQSVersion()
	if err != nil {
		logger.Verbose(fmt.Sprintf("Error getting installed qs version: %v", err))

		return false
	}

	lastQSVersion := getLastQSVersion()
	if semver.Compare(installedVer, lastQSVersion) < 0 {
		fmt.Printf("Installed qs version %s is too old (last version is %s)\n", installedVer, lastQSVersion)
		fmt.Println("You can install last version with:")
		fmt.Println("-----------------------------------------")
		fmt.Println("go install github.com/untillpro/qs@latest")
		fmt.Println("-----------------------------------------")
		fmt.Print("Ignore it and continue with current version(y/n)?")
		var response string
		_, _ = fmt.Scanln(&response)

		return response == pushYes
	}

	return true
}

// ghInstalled returns is gh utility installed
func ghInstalled() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "--version").
		RunToStrings()
	return err == nil
}

// ghLoggedIn returns is gh logged in
func ghLoggedIn() bool {
	_, _, err := new(exec.PipedExec).
		Command("gh", "auth", "status").
		RunToStrings()
	return err == nil
}

func GetInstalledQSVersion() (string, error) {
	stdout, stderr, err := new(exec.PipedExec).
		Command("go", "env", "GOPATH").
		RunToStrings()
	if err != nil {
		return "", fmt.Errorf("GetInstalledVersion error: %s", stderr)
	}

	gopath := strings.TrimSpace(stdout)
	if len(gopath) == 0 {
		return "", errors.New("GetInstalledVersion error: \"GOPATH is not defined\"")
	}
	qsExe := "qs"
	if runtime.GOOS == "windows" {
		qsExe = "qs.exe"
	}

	stdout, stderr, err = new(exec.PipedExec).
		Command("go", "version", "-m", gopath+"/bin/"+qsExe).
		Command("grep", "-i", "-h", "mod.*github.com/untillpro/qs").
		Command("gawk", "{print $3}").
		RunToStrings()
	if err != nil {
		return "", fmt.Errorf("GetInstalledQSVersion error: %s", stderr)
	}

	return strings.TrimSpace(stdout), nil
}

func getLastQSVersion() string {
	stdout, stderr, err := new(exec.PipedExec).
		Command("go", "list", "-m", "-versions", "github.com/untillpro/qs").
		RunToStrings()
	if err != nil {
		logger.Verbose(fmt.Sprintf("getLastQSVersion error: %v", stderr))
	}

	arr := strings.Split(strings.TrimSpace(stdout), oneSpace)
	if len(arr) == 0 {
		return ""
	}

	return arr[len(arr)-1]
}

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

// VerifyGitHubRepoExists checks if a GitHub repository exists and is accessible
func VerifyGitHubRepoExists(owner, repo, token string) error {
	//nolint:gosec
	cmd := osExec.Command("gh", "repo", "view", fmt.Sprintf("%s/%s", owner, repo))

	// Only set GITHUB_TOKEN if a token is provided, otherwise use current gh auth
	if token != "" {
		cmd.Env = append(os.Environ(), fmt.Sprintf("GITHUB_TOKEN=%s", token))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("repository %s/%s not accessible: %w, output: %s", owner, repo, err, output)
	}

	return nil
}

// VerifyGitRemoteAccessible checks if a git remote URL is accessible
func VerifyGitRemoteAccessible(remoteURL string) error {
	cmd := osExec.Command("git", "ls-remote", remoteURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remote URL %s not accessible: %w, output: %s", remoteURL, err, output)
	}

	return nil
}

func CleanArgFromSpecSymbols(arg string) string {
	var symbol string

	arg = strings.ReplaceAll(arg, "https://", "")
	replaceToMinus := []string{oneSpace, ",", ";", ".", ":", "?", "/", "!"}
	for _, symbol = range replaceToMinus {
		arg = strings.ReplaceAll(arg, symbol, "-")
	}
	replaceToNone := []string{"&", "$", "@", "%", "\\", "(", ")", "{", "}", "[", "]", "<", ">", "'", "\""}
	for _, symbol = range replaceToNone {
		arg = strings.ReplaceAll(arg, symbol, "")
	}
	for string(arg[0]) == msymbol {
		arg = arg[1:]
	}

	arg = deleteDupMinus(arg)
	if len(arg) > maxDevBranchName {
		arg = arg[:maxDevBranchName]
	}
	for string(arg[len(arg)-1]) == msymbol {
		arg = arg[:len(arg)-1]
	}
	return arg
}

func deleteDupMinus(str string) string {
	var buf bytes.Buffer
	var pc rune
	for _, c := range str {
		if pc == c && string(c) == msymbol {
			continue
		}
		pc = c
		buf.WriteRune(c)
	}
	return buf.String()
}
