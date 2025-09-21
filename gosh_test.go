package gosh

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// logEntry is used to unmarshal the JSON log output for verification.
type logEntry struct {
	Timestamp int64  `json:"timestamp"`
	Level     string `json:"level"`
	Msg       string `json:"msg"`
}

// captureOutput captures everything written to os.Stdout during the execution of a function.
func captureOutput(f func()) string {
	originalStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Defer the restoration of stdout and the closing of the pipe.
	// This ensures it happens even if the test panics.
	defer func() {
		os.Stdout = originalStdout
	}()

	f() // Execute the function that writes to stdout

	w.Close()
	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

func TestExecSuccess(t *testing.T) {
	ConfigureGlobals() // Ensure our JSON log format is set

	testCases := []struct {
		name             string
		shell            *Shell
		expectedOut      string
		expectedLogLevel string
	}{
		{
			name:             "Simple Echo",
			shell:            New("echo", "hello world"),
			expectedOut:      "hello world",
			expectedLogLevel: "info",
		},
		{
			name:             "Chained Args",
			shell:            New("echo").Arg("hello").Arg("gosh"),
			expectedOut:      "hello gosh",
			expectedLogLevel: "info",
		},
		{
			name:             "Slice of Args",
			shell:            New("echo").Args("hello", "from", "args"),
			expectedOut:      "hello from args",
			expectedLogLevel: "info",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var output string
			var err error

			logOutput := captureOutput(func() {
				output, err = tc.shell.Exec()
			})

			if err != nil {
				t.Fatalf("expected command to succeed, but it failed: %v", err)
			}

			if output != tc.expectedOut {
				t.Errorf("expected stdout %q, but got %q", tc.expectedOut, output)
			}

			// Verify the JSON log entry
			var entry logEntry
			if err := json.Unmarshal([]byte(logOutput), &entry); err != nil {
				t.Fatalf("failed to unmarshal log output: %v\nLog was: %s", err, logOutput)
			}

			if entry.Level != tc.expectedLogLevel {
				t.Errorf("expected log level %q, but got %q", tc.expectedLogLevel, entry.Level)
			}

			if entry.Msg != tc.expectedOut {
				t.Errorf("expected log msg to be the stdout %q, but got %q", tc.expectedOut, entry.Msg)
			}

			if entry.Timestamp == 0 {
				t.Error("expected timestamp to be set in log, but it was zero")
			}
		})
	}
}

func TestExecFailure(t *testing.T) {
	ConfigureGlobals()

	var err error

	logOutput := captureOutput(func() {
		// This command is guaranteed to fail
		_, err = New("ls", "non-existent-dir-for-gosh-test").Exec()
	})

	if err == nil {
		t.Fatal("expected command to fail, but it succeeded")
	}

	// Output might contain partial results, so we don't strictly check it.

	// Verify the JSON log entry for the error
	var entry logEntry
	if err := json.Unmarshal([]byte(logOutput), &entry); err != nil {
		t.Fatalf("failed to unmarshal log output: %v\nLog was: %s", err, logOutput)
	}

	if entry.Level != "error" {
		t.Errorf("expected log level 'error', but got %q", entry.Level)
	}

	// The exact stderr message can vary slightly by OS, so we check for a substring.
	expectedErrMsg := "No such file or directory"
	if !strings.Contains(entry.Msg, expectedErrMsg) {
		t.Errorf("expected log msg to contain %q, but got %q", expectedErrMsg, entry.Msg)
	}
}

func TestExecInDir(t *testing.T) {
	ConfigureGlobals()

	// t.TempDir() creates a temporary directory that is automatically cleaned up after the test.
	tempDir := t.TempDir()
	testFile := "my_test_file.txt"
	filePath := filepath.Join(tempDir, testFile)

	// Create a file in the temporary directory
	if err := os.WriteFile(filePath, []byte("hello"), 0666); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}

	var output string
	var err error

	captureOutput(func() {
		output, err = New("ls").Dir(tempDir).Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	if !strings.Contains(output, testFile) {
		t.Errorf("expected output to contain temp file name %q, but got %q", testFile, output)
	}
}

func TestExecWithEnv(t *testing.T) {
	ConfigureGlobals()

	// This test is written for Unix-like systems (Linux, macOS).
	// On Windows, the command would be `cmd /C "echo %MY_TEST_VAR%"`
	if runtime.GOOS == "windows" {
		t.Skip("Skipping env test on Windows; requires different command syntax.")
	}

	expectedVar := "gosh-is-great"
	var output string
	var err error

	captureOutput(func() {
		output, err = New("sh", "-c", "echo $MY_TEST_VAR").Env("MY_TEST_VAR", expectedVar).Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	if output != expectedVar {
		t.Errorf("expected output to be the env var %q, but got %q", expectedVar, output)
	}
}
