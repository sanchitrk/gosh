package gosh

import (
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
	url    string
	client *http.Client
	buffer bytes.Buffer
	mutex  sync.Mutex
}

// NewHTTPStreamWriter creates a new HTTP stream writer
func NewHTTPStreamWriter(url string) *HTTPStreamWriter {
	return &HTTPStreamWriter{
		url:    url,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

// Write implements io.Writer interface
func (w *HTTPStreamWriter) Write(p []byte) (n int, err error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	
	// Send immediately to HTTP endpoint
	go func() {
		req, err := http.NewRequest("POST", w.url, bytes.NewBuffer(p))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		
		resp, err := w.client.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
	}()
	
	return len(p), nil
}

// Close closes the writer (no-op for this implementation)
func (w *HTTPStreamWriter) Close() error {
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
}

// New creates a new Shell builder instance.
// The first call to Arg() will set the command, subsequent calls add arguments.
func New() *Shell {
	return &Shell{
		log: zerolog.New(os.Stdout).With().Timestamp().Logger(),
	}
}

// WithHTTPStream configures the Shell to stream logs to an HTTP endpoint.
// This uses io.Pipe for efficient streaming of logs to the HTTP endpoint.
func (s *Shell) WithHTTPStream(url string) *Shell {
	s.streamingURL = url
	s.httpWriter = NewHTTPStreamWriter(url)
	
	// Create a multi-writer to send logs both to stdout and HTTP endpoint
	multiWriter := io.MultiWriter(os.Stdout, s.httpWriter)
	s.log = zerolog.New(multiWriter).With().Timestamp().Logger()
	
	return s
}

// WithHTTPStreamOnly configures the Shell to stream logs only to an HTTP endpoint.
// This sends logs exclusively to the HTTP endpoint without local stdout output.
func (s *Shell) WithHTTPStreamOnly(url string) *Shell {
	s.streamingURL = url
	s.httpWriter = NewHTTPStreamWriter(url)
	s.log = zerolog.New(s.httpWriter).With().Timestamp().Logger()
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

	// Log the command being executed
	s.log.Info().
		Str("command", s.command).
		Strs("args", s.args).
		Str("dir", s.dir).
		Strs("env", s.env).
		Msg("executing command")

	err := cmd.Run()

	stdout := strings.TrimSpace(stdoutBuf.String())
	stderr := strings.TrimSpace(stderrBuf.String())

	if err != nil {
		// On error, log stderr as the error message.
		if stderr != "" {
			s.log.Error().
				Str("command", s.command).
				Strs("args", s.args).
				Str("stderr", stderr).
				Err(err).
				Msg("command failed")
		} else {
			// Fallback if stderr is empty but an error still occurred
			s.log.Error().
				Str("command", s.command).
				Strs("args", s.args).
				Err(err).
				Msg("command failed without stderr output")
		}
		return stdout, err
	}

	// On success, log stdout as the info message.
	s.log.Info().
		Str("command", s.command).
		Strs("args", s.args).
		Str("stdout", stdout).
		Msg("command completed successfully")

	return stdout, nil
}

// Legacy constructor for backward compatibility
// Deprecated: Use New().Command(command).Args(args...) instead
func NewLegacy(command string, args ...string) *Shell {
	return &Shell{
		command: command,
		args:    args,
		log:     zerolog.New(os.Stdout).With().Timestamp().Logger(),
	}
}