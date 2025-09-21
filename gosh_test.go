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
	Timestamp int64    `json:"timestamp"`
	Level     string   `json:"level"`
	Msg       string   `json:"msg"`
	Command   string   `json:"command,omitempty"`
	Args      []string `json:"args,omitempty"`
	Dir       string   `json:"dir,omitempty"`
	Stdout    string   `json:"stdout,omitempty"`
	Stderr    string   `json:"stderr,omitempty"`
	Error     string   `json:"error,omitempty"`
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

func TestNewBuilderPattern(t *testing.T) {
	ConfigureGlobals()

	testCases := []struct {
		name        string
		setupShell  func() *Shell
		expectedOut string
		shouldError bool
	}{
		{
			name: "First Arg sets command",
			setupShell: func() *Shell {
				return New().Arg("echo").Arg("hello")
			},
			expectedOut: "hello",
			shouldError: false,
		},
		{
			name: "Args with first as command",
			setupShell: func() *Shell {
				return New().Args("echo", "hello", "world")
			},
			expectedOut: "hello world",
			shouldError: false,
		},
		{
			name: "Explicit Command method",
			setupShell: func() *Shell {
				return New().Command("echo").Arg("test")
			},
			expectedOut: "test",
			shouldError: false,
		},
		{
			name: "Mixed Command and Args",
			setupShell: func() *Shell {
				return New().Command("echo").Args("mixed", "test")
			},
			expectedOut: "mixed test",
			shouldError: false,
		},
		{
			name: "No command should error",
			setupShell: func() *Shell {
				return New().Dir("/tmp")
			},
			expectedOut: "",
			shouldError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var output string
			var err error

			logOutput := captureOutput(func() {
				output, err = tc.setupShell().Exec()
			})

			if tc.shouldError {
				if err == nil {
					t.Fatal("expected command to fail, but it succeeded")
				}
				return
			}

			if err != nil {
				t.Fatalf("expected command to succeed, but it failed: %v", err)
			}

			if output != tc.expectedOut {
				t.Errorf("expected stdout %q, but got %q", tc.expectedOut, output)
			}

			// Parse and verify log entries
			lines := strings.Split(strings.TrimSpace(logOutput), "\n")
			if len(lines) < 2 {
				t.Fatalf("expected at least 2 log lines, got %d: %v", len(lines), lines)
			}

			// First log should be the "executing command" log
			var execEntry logEntry
			if err := json.Unmarshal([]byte(lines[0]), &execEntry); err != nil {
				t.Fatalf("failed to unmarshal execution log: %v\nLog was: %s", err, lines[0])
			}

			if execEntry.Level != "info" {
				t.Errorf("expected execution log level 'info', got %q", execEntry.Level)
			}

			if execEntry.Msg != "executing command" {
				t.Errorf("expected execution log msg 'executing command', got %q", execEntry.Msg)
			}

			// Second log should be the completion log
			var completionEntry logEntry
			if err := json.Unmarshal([]byte(lines[1]), &completionEntry); err != nil {
				t.Fatalf("failed to unmarshal completion log: %v\nLog was: %s", err, lines[1])
			}

			if completionEntry.Level != "info" {
				t.Errorf("expected completion log level 'info', got %q", completionEntry.Level)
			}

			if completionEntry.Msg != "command completed successfully" {
				t.Errorf("expected completion log msg 'command completed successfully', got %q", completionEntry.Msg)
			}

			if completionEntry.Stdout != tc.expectedOut {
				t.Errorf("expected log stdout %q, got %q", tc.expectedOut, completionEntry.Stdout)
			}
		})
	}
}

func TestExecFailure(t *testing.T) {
	ConfigureGlobals()

	var err error

	logOutput := captureOutput(func() {
		// This command is guaranteed to fail
		_, err = New().Arg("ls").Arg("non-existent-dir-for-gosh-test").Exec()
	})

	if err == nil {
		t.Fatal("expected command to fail, but it succeeded")
	}

	// Parse log entries
	lines := strings.Split(strings.TrimSpace(logOutput), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d", len(lines))
	}

	// Last log should be the error log
	var errorEntry logEntry
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &errorEntry); err != nil {
		t.Fatalf("failed to unmarshal error log: %v\nLog was: %s", err, lines[len(lines)-1])
	}

	if errorEntry.Level != "error" {
		t.Errorf("expected log level 'error', but got %q", errorEntry.Level)
	}

	if errorEntry.Msg != "command failed" {
		t.Errorf("expected log msg 'command failed', but got %q", errorEntry.Msg)
	}

	if errorEntry.Command != "ls" {
		t.Errorf("expected command to be 'ls', got %q", errorEntry.Command)
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
		output, err = New().Arg("ls").Dir(tempDir).Exec()
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
		output, err = New().
			Command("sh").
			Args("-c", "echo $MY_TEST_VAR").
			Env("MY_TEST_VAR", expectedVar).
			Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	if output != expectedVar {
		t.Errorf("expected output to be the env var %q, but got %q", expectedVar, output)
	}
}

func TestBuilderChaining(t *testing.T) {
	ConfigureGlobals()

	// Test complex chaining
	var output string
	var err error

	captureOutput(func() {
		output, err = New().
			Command("echo").
			Arg("hello").
			Arg("builder").
			Args("pattern", "test").
			Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	expected := "hello builder pattern test"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}
}

func TestEmptyArgsHandling(t *testing.T) {
	ConfigureGlobals()

	// Test that empty Args() call doesn't break anything
	var output string
	var err error

	captureOutput(func() {
		output, err = New().
			Args(). // Empty args call
			Arg("echo").
			Arg("test").
			Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	if output != "test" {
		t.Errorf("expected output 'test', got %q", output)
	}
}

func TestArgsWithEmptyCommand(t *testing.T) {
	ConfigureGlobals()

	// Test Args() when no command is set yet
	var output string
	var err error

	captureOutput(func() {
		output, err = New().
			Args("echo", "from", "args").
			Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	expected := "from args"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}
}

// Test backward compatibility
func TestLegacyConstructor(t *testing.T) {
	ConfigureGlobals()

	var output string
	var err error

	captureOutput(func() {
		output, err = NewLegacy("echo", "legacy", "test").Exec()
	})

	if err != nil {
		t.Fatalf("expected command to succeed, but it failed: %v", err)
	}

	expected := "legacy test"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}
}