package testutil

import (
	"fmt"
	"os"
	"testing"

	"github.com/davoodharun/terragrunt-scaffolder/internal/logger"
)

// TestLogger provides clean test logging
type TestLogger struct {
	t *testing.T
}

// NewTestLogger creates a new test logger
func NewTestLogger(t *testing.T) *TestLogger {
	return &TestLogger{t: t}
}

// LogTestStart logs the start of a test
func (l *TestLogger) LogTestStart(name string) {
	l.t.Logf("=== Running test: %s ===", name)
}

// LogTestResult logs the result of a test
func (l *TestLogger) LogTestResult(name string, err error) {
	if err != nil {
		l.t.Logf("❌ Test failed: %s - %v", name, err)
		os.Exit(1) // Exit with error code 1 on test failure
	} else {
		l.t.Logf("✅ Test passed: %s", name)
	}
}

// RunTest runs a test with clean logging
func RunTest(t *testing.T, name string, testFunc func() error) {
	// Disable application logging during tests
	logger.Disable()
	defer logger.Enable()

	logger := NewTestLogger(t)
	logger.LogTestStart(name)
	err := testFunc()
	logger.LogTestResult(name, err)
}

// AssertEqual compares two values and returns an error if they're not equal
func AssertEqual[T comparable](expected, actual T) error {
	if expected != actual {
		return fmt.Errorf("expected %v, got %v", expected, actual)
	}
	return nil
}

// AssertNoError checks if an error is nil and returns a formatted error if not
func AssertNoError(err error) error {
	if err != nil {
		return fmt.Errorf("unexpected error: %v", err)
	}
	return nil
}

// AssertError checks if an error matches the expected error
func AssertError(err, expectedErr error) error {
	if err == nil {
		return fmt.Errorf("expected error %v, got nil", expectedErr)
	}
	if err.Error() != expectedErr.Error() {
		return fmt.Errorf("expected error %v, got %v", expectedErr, err)
	}
	return nil
}
