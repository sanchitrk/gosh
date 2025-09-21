# Gosh - Shell Command Builder with HTTP Log Streaming

A fluent, builder-pattern Go library for executing shell commands with structured logging and HTTP log streaming capabilities using zerolog.

## Features

- 🔧 **Fluent Builder Pattern**: Chain methods for readable command construction
- 📡 **HTTP Log Streaming**: Stream structured logs to HTTP endpoints using `io.Pipe`
- 📝 **Structured Logging**: Built on zerolog for consistent, structured log output
- 🎯 **Flexible Command Building**: Set commands and arguments in any order
- 🌍 **Environment Control**: Set working directories and environment variables
- ⚡ **Efficient Streaming**: Uses `io.Pipe` for memory-efficient HTTP streaming

## Installation

```bash
go get github.com/sanchitrk/gosh
```

## Quick Start

### Basic Usage

```go
package main

import (
    "log"
    "github.com/sanchitrk/gosh"
)

func main() {
    // Configure zerolog globals (optional, call once in main)
    gosh.ConfigureGlobals()
    
    // Execute a simple command
    output, err := gosh.New().
        Arg("echo").
        Arg("Hello, World!").
        Exec()
    
    if err != nil {
        log.Fatal(err)
    }
    
    fmt.Println("Output:", output)
}
```

### HTTP Log Streaming

First, start the included log server:

```bash
go run cmd/srv/srv.go
```

Then stream your command logs to it:

```go
// Stream logs to HTTP endpoint + stdout
output, err := gosh.New().
    WithHTTPStream("http://localhost:8080/logs").
    Arg("ls").
    Arg("-la").
    Exec()

// Stream logs only to HTTP endpoint (no stdout)
output, err := gosh.New().
    WithHTTPStreamOnly("http://localhost:8080/logs").
    Command("whoami").
    Exec()
```

## API Reference

### Creating a Shell Builder

```go
// Create a new shell builder
shell := gosh.New()
```

### Setting Commands and Arguments

The builder pattern allows flexible command construction:

```go
// Method 1: First Arg() sets command, subsequent calls add arguments
gosh.New().Arg("ls").Arg("-la").Arg("/tmp")

// Method 2: Use Args() with slice
gosh.New().Args("git", "log", "--oneline", "-n", "5")

// Method 3: Explicit command setting
gosh.New().Command("docker").Args("ps", "-a")

// Method 4: Mixed approach
gosh.New().Command("find").Arg(".").Args("-name", "*.go")
```

### HTTP Streaming

```go
// Stream to HTTP + local stdout
shell.WithHTTPStream("http://localhost:8080/logs")

// Stream only to HTTP (no local output)
shell.WithHTTPStreamOnly("http://localhost:8080/logs")
```

### Environment and Directory Control

```go
shell.Dir("/path/to/working/directory").
    Env("CUSTOM_VAR", "value").
    Env("ANOTHER_VAR", "another_value")
```

### Custom Logger

```go
customLogger := zerolog.New(os.Stderr).With().
    Str("service", "my-app").
    Timestamp().
    Logger()

shell.Logger(customLogger)
```

### Execution

```go
output, err := shell.Exec()
```

## Complete Example

```go
package main

import (
    "log"
    "github.com/sanchitrk/gosh"
)

func main() {
    gosh.ConfigureGlobals()
    
    // Complex command with all features
    output, err := gosh.New().
        WithHTTPStream("http://localhost:8080/logs").
        Command("git").
        Args("log", "--format=%h %s").
        Arg("-n").
        Arg("10").
        Dir("/path/to/repo").
        Env("PAGER", "cat").
        Exec()
    
    if err != nil {
        log.Printf("Command failed: %v", err)
        return
    }
    
    log.Printf("Git log output: %s", output)
}
```

## HTTP Log Server

The library includes a simple HTTP log server for testing:

```go
// cmd/srv/srv.go
package main

import (
    "io"
    "log"
    "net/http"
    "os"
)

func main() {
    http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
        log.Println("Received log stream...")
        io.Copy(os.Stdout, r.Body)
        w.WriteHeader(http.StatusOK)
    })
    log.Println("Log ingestor server starting on :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
```

Start it with:

```bash
go run cmd/srv/srv.go
```

## Log Structure

All logs use a clean, simplified JSON format:

**INFO logs** (stdout):
```json
{"timestamp": 1758477507, "level": "info", "msg": "Hello from stdout!"}
```

**ERROR logs** (stderr):
```json
{"timestamp": 1758477507, "level": "error", "msg": "ls: non-existent-dir: No such file or directory"}
```

Key points:
- **stdout** is logged as **INFO** level with the actual command output as the message
- **stderr** is logged as **ERROR** level with the actual error output as the message
- No extra metadata cluttering the logs - just timestamp, level, and the actual output
- Both stdout and stderr are logged regardless of command success/failure

## HTTP Streaming Implementation

The HTTP streaming uses Go's `io.Pipe` for efficient, memory-conscious streaming:

- **Efficient**: Uses streaming rather than buffering entire logs
- **Non-blocking**: HTTP failures don't block command execution
- **Concurrent**: Streaming happens in background goroutines
- **Clean**: Automatically closes connections when commands complete

## Migration from v1

If you're migrating from the old API:

```go
// Old way (deprecated)
gosh.New("ls", "-la")

// New way
gosh.New().Args("ls", "-la")
// or
gosh.New().Command("ls").Arg("-la")
```

The old constructor is still available as `NewLegacy()` for backward compatibility.

## Error Handling

Commands that fail will:
1. Return the error from `Exec()`
2. Log error details as structured JSON
3. Stream error logs to HTTP endpoint (if configured)

```go
output, err := gosh.New().Arg("false").Exec()
if err != nil {
    // Handle the error
    // Error details are also logged automatically
}
```

## Best Practices

1. **Call `ConfigureGlobals()` once** in your main function
2. **Use HTTP streaming** for centralized logging in distributed systems
3. **Chain methods** for readable command construction
4. **Handle errors** appropriately - they're automatically logged but should be handled
5. **Use `Dir()` and `Env()`** to ensure commands run in correct context

## Examples

See `cmd/example/main.go` for comprehensive usage examples.

## License

MIT License - see LICENSE file for details.