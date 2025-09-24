package gosh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// ConfigureGlobals sets the package-level zerolog configuration for field names
// and timestamp format. This should be called once at the start of your main() func.
func ConfigureGlobals() {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

// HTTPStreamWriter implements io.Writer for sending logs to HTTP endpoints
type HTTPStreamWriter struct {
	url     string
	client  *http.Client
	buffer  bytes.Buffer
	mutex   sync.Mutex
	headers http.Header
	wg      sync.WaitGroup
}

// NewHTTPStreamWriter creates a new HTTP stream writer
func NewHTTPStreamWriter(url string, headers http.Header) *HTTPStreamWriter {
	return &HTTPStreamWriter{
		url:     url,
		client:  &http.Client{Timeout: 30 * time.Second}, // Increased timeout
		headers: headers,
	}
}

// Write implements io.Writer interface
func (w *HTTPStreamWriter) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	// Add incoming data to buffer
	w.buffer.Write(p)

	// Process complete lines (JSON objects end with newlines)
	for {
		line, err := w.buffer.ReadBytes('\n')
		if err != nil {
			// No complete line available, put data back and break
			w.buffer.Write(line)
			break
		}

		// Send complete line to HTTP endpoint
		w.wg.Add(1)
		go func(data []byte) {
			defer w.wg.Done()

			// Console log the payload being sent
			// fmt.Printf("HTTP Stream Payload: %s", string(data))

			req, err := http.NewRequest("POST", w.url, bytes.NewBuffer(data))
			if err != nil {
				fmt.Printf("HTTP Stream Error creating request: %v\n", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			for key, values := range w.headers {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			resp, err := w.client.Do(req)
			if err != nil {
				fmt.Printf("HTTP Stream Error sending request: %v\n", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode >= 400 {
				fmt.Printf("HTTP Stream Error response status: %s\n", resp.Status)
			}
		}(line)
	}

	return len(p), nil
}

// Close closes the writer and waits for all HTTP requests to complete
func (w *HTTPStreamWriter) Close() error {
	// Send any remaining buffered data
	w.mutex.Lock()
	if w.buffer.Len() > 0 {
		remaining := w.buffer.Bytes()
		w.buffer.Reset()
		w.mutex.Unlock()

		// Send remaining data if any
		w.wg.Add(1)
		go func(data []byte) {
			defer w.wg.Done()
			fmt.Printf("HTTP Stream Final Payload: %s", string(data))

			req, err := http.NewRequest("POST", w.url, bytes.NewBuffer(data))
			if err != nil {
				fmt.Printf("HTTP Stream Error creating final request: %v\n", err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			for key, values := range w.headers {
				for _, value := range values {
					req.Header.Add(key, value)
				}
			}

			resp, err := w.client.Do(req)
			if err != nil {
				fmt.Printf("HTTP Stream Error sending final request: %v\n", err)
				return
			}
			defer resp.Body.Close()
		}(remaining)
	} else {
		w.mutex.Unlock()
	}

	// Wait for all HTTP requests to complete
	w.wg.Wait()
	return nil
}

// Shell is the builder for executing shell commands.
type Shell struct {
	command      string
	args         []string
	dir          string
	env          []string
	log          zerolog.Logger
	httpWriter   *HTTPStreamWriter
	streamingURL string
	httpHeaders  http.Header
	logKVs       map[string]string
}

// New creates a new Shell builder instance.
// The first call to Arg() will set the command, subsequent calls add arguments.
func New() *Shell {
	return &Shell{
		log:         zerolog.New(os.Stdout).With().Timestamp().Logger(),
		httpHeaders: make(http.Header),
	}
}

// WithHTTPStream configures the Shell to stream logs to an HTTP endpoint.
// This uses io.Pipe for efficient streaming of logs to the HTTP endpoint.
func (s *Shell) WithHTTPStream(url string) *Shell {
	s.streamingURL = url
	s.httpWriter = NewHTTPStreamWriter(url, s.httpHeaders)

	// Create a multi-writer to send logs both to stdout and HTTP endpoint
	multiWriter := io.MultiWriter(os.Stdout, s.httpWriter)
	s.log = zerolog.New(multiWriter).With().Timestamp().Logger()

	return s
}

// WithHTTPStreamOnly configures the Shell to stream logs only to an HTTP endpoint.
// This sends logs exclusively to the HTTP endpoint without local stdout output.
func (s *Shell) WithHTTPStreamOnly(url string) *Shell {
	s.streamingURL = url
	s.httpWriter = NewHTTPStreamWriter(url, s.httpHeaders)
	s.log = zerolog.New(s.httpWriter).With().Timestamp().Logger()
	return s
}

// AddHTTPHeader adds a header to be sent with HTTP stream requests.
func (s *Shell) AddHTTPHeader(key, value string) *Shell {
	s.httpHeaders.Add(key, value)
	return s
}

// Arg adds an argument to the command. The first call to Arg sets the command,
// subsequent calls add arguments to that command.
func (s *Shell) Arg(arg string) *Shell {
	if s.command == "" {
		s.command = arg
	} else {
		s.args = append(s.args, arg)
	}
	return s
}

// Args adds multiple arguments to the command from a slice.
// If no command is set yet, the first argument becomes the command.
func (s *Shell) Args(args ...string) *Shell {
	if len(args) == 0 {
		return s
	}

	if s.command == "" && len(args) > 0 {
		s.command = args[0]
		if len(args) > 1 {
			s.args = append(s.args, args[1:]...)
		}
	} else {
		s.args = append(s.args, args...)
	}
	return s
}

// Command explicitly sets the command, allowing you to separate command setting
// from argument adding. This is useful if you want to be explicit about the command.
func (s *Shell) Command(cmd string) *Shell {
	s.command = cmd
	return s
}

// Dir sets the working directory for the command.
// If not set, it runs in the current process's working directory.
func (s *Shell) Dir(path string) *Shell {
	s.dir = path
	return s
}

// Env sets an environment variable for the command in "key=value" format.
// These are appended to the parent process's environment.
func (s *Shell) Env(key, value string) *Shell {
	s.env = append(s.env, key+"="+value)
	return s
}

func (s *Shell) LogKV(key, value string) *Shell {
	if s.logKVs == nil {
		s.logKVs = make(map[string]string)
	}
	s.logKVs[key] = value
	return s
}

func (s *Shell) ClearLogKV() *Shell {
	s.logKVs = make(map[string]string)
	return s
}

// Logger allows you to inject your own configured zerolog.Logger instance,
// overriding the library's default logger configuration.
func (s *Shell) Logger(logger zerolog.Logger) *Shell {
	s.log = logger
	return s
}

// Exec executes the configured command. It returns the standard output as a
// trimmed string and an error if the command fails. On success, it logs stdout
// as an info message. On failure, it logs stderr as an error message.
func (s *Shell) Exec() (string, error) {
	if s.command == "" {
		return "", fmt.Errorf("no command specified - use Arg() or Command() to set the command")
	}

	// Clean up HTTP writer when done
	defer func() {
		if s.httpWriter != nil {
			s.httpWriter.Close()
		}
	}()

	cmd := exec.Command(s.command, s.args...)

	if s.dir != "" {
		cmd.Dir = s.dir
	}
	if len(s.env) > 0 {
		cmd.Env = append(os.Environ(), s.env...)
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err := cmd.Run()

	stdout := strings.TrimSpace(stdoutBuf.String())
	stderr := strings.TrimSpace(stderrBuf.String())

	// Always log stderr if present (even on success, some commands write to stderr)
	if stderr != "" {
		logEvent := s.log.Error()
		for k, v := range s.logKVs {
			logEvent = logEvent.Str(k, v)
		}
		logEvent.Msg(stderr)
	}

	// Always log stdout if present
	if stdout != "" {
		logEvent := s.log.Info()
		for k, v := range s.logKVs {
			logEvent = logEvent.Str(k, v)
		}
		logEvent.Msg(stdout)
	}

	return stdout, err
}

// Stream executes the configured command with real-time output streaming.
// Unlike Exec(), this method streams stdout and stderr in real-time through
// zerolog, preserving the configured formatting and HTTP streaming settings.
// Returns an error if the command fails.
func (s *Shell) Stream() error {
	if s.command == "" {
		return fmt.Errorf("no command specified - use Arg() or Command() to set the command")
	}

	// Clean up HTTP writer when done
	defer func() {
		if s.httpWriter != nil {
			s.httpWriter.Close()
		}
	}()

	cmd := exec.Command(s.command, s.args...)

	if s.dir != "" {
		cmd.Dir = s.dir
	}
	if len(s.env) > 0 {
		cmd.Env = append(os.Environ(), s.env...)
	}

	// Create pipes for real-time streaming
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Use WaitGroup to handle concurrent streaming
	var wg sync.WaitGroup

	// Stream stdout through zerolog as info messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				logEvent := s.log.Info()
				for k, v := range s.logKVs {
					logEvent = logEvent.Str(k, v)
				}
				logEvent.Msg(line)
			}
		}
	}()

	// Stream stderr through zerolog as error messages
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			if line != "" {
				logEvent := s.log.Error()
				for k, v := range s.logKVs {
					logEvent = logEvent.Str(k, v)
				}
				logEvent.Msg(line)
			}
		}
	}()

	// Wait for all streaming to complete
	wg.Wait()

	// Wait for the command to complete and return its error status
	return cmd.Wait()
}
