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

	fmt.Println("=== Testing Exec and Stream methods with HTTP streaming ===")
	fmt.Println("Note: Make sure the log server is running on http://localhost:8080")
	fmt.Println("Start it with: go run cmd/srv/srv.go")
	fmt.Println()

	// Test 1: Exec method with HTTP streaming
	fmt.Println("1. Testing Exec() method with WithHTTPStream:")
	_, err := gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Arg("echo").
		Arg("Exec method: This should appear in console AND HTTP endpoint").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Exec completed successfully\n")
	}
	time.Sleep(100 * time.Millisecond)

	// Test 2: Exec method with HTTP streaming only
	fmt.Println("\n2. Testing Exec() method with WithHTTPStreamOnly:")
	_, err = gosh.New().
		WithHTTPStreamOnly("http://localhost:8080/logs").
		Arg("echo").
		Arg("Exec method: This should appear ONLY at HTTP endpoint").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Exec completed (output sent only to HTTP)\n")
	}
	time.Sleep(100 * time.Millisecond)

	// Test 3: Stream method with HTTP streaming
	fmt.Println("\n3. Testing Stream() method with WithHTTPStream:")
	err = gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Command("sh").
		Args("-c", "echo 'Stream method: Line 1'; echo 'Stream method: Line 2'; echo 'Stream method: Line 3'").
		Stream()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Stream completed successfully\n")
	}
	time.Sleep(100 * time.Millisecond)

	// Test 4: Stream method with HTTP streaming only
	fmt.Println("\n4. Testing Stream() method with WithHTTPStreamOnly:")
	err = gosh.New().
		WithHTTPStreamOnly("http://localhost:8080/logs").
		Command("sh").
		Args("-c", "echo 'Stream method ONLY: Line 1'; echo 'Stream method ONLY: Line 2'").
		Stream()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Stream completed (output sent only to HTTP)\n")
	}
	time.Sleep(100 * time.Millisecond)

	// Test 5: Stream method with error output
	fmt.Println("\n5. Testing Stream() method with stderr:")
	err = gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Command("sh").
		Args("-c", "echo 'Stream stdout message'; echo 'Stream stderr message' >&2").
		Stream()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Stream with stderr completed\n")
	}
	time.Sleep(100 * time.Millisecond)

	// Test 6: Long-running command with Stream
	fmt.Println("\n6. Testing Stream() method with long-running command:")
	err = gosh.New().
		WithHTTPStream("http://localhost:8080/logs").
		Command("sh").
		Args("-c", "for i in 1 2 3 4 5; do echo 'Stream real-time: Step '$i; sleep 0.2; done").
		Stream()
	if err != nil {
		log.Printf("Error: %v", err)
	} else {
		fmt.Printf("✓ Long-running Stream completed\n")
	}

	// Test 7: Compare formatting between Exec and Stream
	fmt.Println("\n7. Comparing Exec vs Stream formatting:")

	fmt.Println("  Exec method:")
	_, err = gosh.New().
		Arg("echo").
		Arg("Exec: Formatted output test").
		Exec()
	if err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println("  Stream method:")
	err = gosh.New().
		Command("echo").
		Arg("Stream: Formatted output test").
		Stream()
	if err != nil {
		log.Printf("Error: %v", err)
	}

	fmt.Println("\n=== All tests completed! ===")
	fmt.Println("Check the log server output to verify HTTP streaming worked correctly.")
	fmt.Println("Both methods should now produce consistent zerolog formatting.")
}
