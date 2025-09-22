package main

import (
	"fmt"
	"log"
	"time"

	"github.com/sanchitrk/gosh"
)

func main() {
	// Configure global zerolog settings
	gosh.ConfigureGlobals()

	fmt.Println("=== New Builder Pattern Examples ===")

	// Example 1: Basic usage - command is set with first Arg()
	fmt.Println("\n1. Basic command execution:")
	output, err := gosh.New().
		Arg("echo").
		Arg("Hello from gosh!").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Output: %s\n", output)
	}

	// Example 2: Using Args method
	fmt.Println("\n2. Using Args method:")
	output, err = gosh.New().
		Args("ls", "-la", "/tmp").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Directory listing: %d characters\n", len(output))
	}

	// Example 3: Using explicit Command method
	fmt.Println("\n3. Explicit command setting:")
	output, err = gosh.New().
		Command("date").
		Arg("+%Y-%m-%d %H:%M:%S").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Current time: %s\n", output)
	}

	// Example 4: Complex chaining with environment and directory
	fmt.Println("\n4. Complex chaining:")
	output, err = gosh.New().
		Command("printenv").
		Arg("CUSTOM_VAR").
		Dir("/tmp").
		Env("CUSTOM_VAR", "gosh_test_value").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Environment variable: %s\n", output)
	}

	fmt.Println("\n=== HTTP Streaming Examples ===")
	fmt.Println("Note: Make sure to run the log server first:")
	fmt.Println("  go run cmd/srv/srv.go")
	fmt.Println("  Server should be running on http://localhost:8080")

	// Example 5: HTTP streaming with dual output (stdout + HTTP)
	fmt.Println("\n5. HTTP streaming with dual output:")
	output, err = gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Arg("echo").
		Arg("This log goes to both stdout and HTTP endpoint").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Command completed. Check the log server output!\n")
	}

	// Give the HTTP stream a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Example 6: HTTP streaming only (no stdout)
	fmt.Println("\n6. HTTP streaming only:")
	output, err = gosh.New().
		WithHTTPStreamOnly("http://localhost:8080/logs").
		Arg("whoami").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Command completed. Output only sent to HTTP endpoint.\n")
	}

	// Give the HTTP stream a moment to complete
	time.Sleep(100 * time.Millisecond)

	// Example 7: Multiple commands with streaming
	fmt.Println("\n7. Multiple commands with streaming:")
	commands := [][]string{
		{"uname", "-a"},
		{"pwd"},
		{"date"},
		{"echo", "Batch processing complete"},
	}

	for i, cmd := range commands {
		fmt.Printf("Executing command %d/%d: %v\n", i+1, len(commands), cmd)
		
		shell := gosh.New().WithHTTPStream("http://localhost:8080/logs")
		for j, arg := range cmd {
			if j == 0 {
				shell = shell.Command(arg)
			} else {
				shell = shell.Arg(arg)
			}
		}
		
		output, err := shell.Exec()
		if err != nil {
			log.Printf("Command failed: %v", err)
		} else {
			fmt.Printf("  ✓ Success: %d chars output\n", len(output))
		}
		
		// Small delay between commands
		time.Sleep(50 * time.Millisecond)
	}

	// Example 8: Error handling with HTTP streaming
	fmt.Println("\n8. Error handling with streaming:")
	output, err = gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Arg("nonexistent-command-test").
		Exec()
	if err != nil {
		fmt.Printf("Expected error occurred (also streamed to HTTP): %v\n", err)
	}

	// Example 9: Long-running command with streaming
	fmt.Println("\n9. Long-running command with streaming:")
	output, err = gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Command("sh").
		Args("-c", "for i in 1 2 3; do echo Step $i; sleep 0.1; done").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Long-running command completed: %s\n", output)
	}

	fmt.Println("\n=== Builder Pattern Flexibility Examples ===")

	// Example 10: Building command step by step
	fmt.Println("\n10. Step-by-step command building:")
	shell := gosh.New()
	
	// Conditionally add HTTP streaming
	useHTTPLogging := true // This could come from config/flags
	if useHTTPLogging {
		shell = shell.WithHTTPStream("http://localhost:8080/logs")
	}
	
	// Set base command
	shell = shell.Command("ls")
	
	// Conditionally add flags
	showHidden := true
	if showHidden {
		shell = shell.Arg("-la")
	} else {
		shell = shell.Arg("-l")
	}
	
	// Set target directory
	shell = shell.Arg(".")
	
	output, err = shell.Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("Conditional command completed: %d chars\n", len(output))
	}

	// Example 11: Error case - no command specified
	fmt.Println("\n11. Error case - no command:")
	output, err = gosh.New().Dir("/tmp").Exec()
	if err != nil {
		fmt.Printf("Expected error: %v\n", err)
	}

	fmt.Println("\n=== Examples completed! ===")
	fmt.Println("Check your log server output to see the streamed logs.")
}