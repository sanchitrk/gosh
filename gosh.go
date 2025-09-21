package gosh

import (
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/rs/zerolog"
)

// ConfigureGlobals sets the package-level zerolog configuration for field names
// and timestamp format. This should be called once at the start of your main() func.
func ConfigureGlobals() {
	zerolog.TimestampFieldName = "timestamp"
	zerolog.MessageFieldName = "msg"
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
}

// Shell is the builder for executing shell commands.
type Shell struct {
	command string
	args    []string
	dir     string
	env     []string
	log     zerolog.Logger
}

// New creates a new Shell builder instance for a given command.
// It defaults to a structured JSON logger writing to os.Stdout.
func New(command string, args ...string) *Shell {
	return &Shell{
		command: command,
		args:    args,
		log:     zerolog.New(os.Stdout).With().Timestamp().Logger(),
	}
}

// Arg adds a single argument to the command.
func (s *Shell) Arg(arg string) *Shell {
	s.args = append(s.args, arg)
	return s
}

// Args adds multiple arguments to the command from a slice.
func (s *Shell) Args(args ...string) *Shell {
	s.args = append(s.args, args...)
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
// overriding the library's default JSON logger.
func (s *Shell) Logger(logger zerolog.Logger) *Shell {
	s.log = logger
	return s
}

// Exec executes the configured command. It returns the standard output as a
// trimmed string and an error if the command fails. On success, it logs stdout
// as an info message. On failure, it logs stderr as an error message.
func (s *Shell) Exec() (string, error) {
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

	if err != nil {
		// On error, log stderr as the error message.
		if stderr != "" {
			s.log.Error().Msg(stderr)
		} else {
			// Fallback if stderr is empty but an error still occurred
			s.log.Error().Err(err).Msg("command failed without stderr output")
		}
		return stdout, err
	}

	// On success, log stdout as the info message.
	s.log.Info().Msg(stdout)

	return stdout, nil
}
