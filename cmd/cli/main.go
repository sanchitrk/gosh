package main

import (
	"log"

	"github.com/sanchitrk/gosh"
)

func main() {
	// Call this once to set up the desired logging format globally.
	gosh.ConfigureGlobals()

	_, err := gosh.New("echo", "Hello from gosh!").Exec()
	if err != nil {
		log.Fatal(err)
	}

	_, err = gosh.New("ls", "non-existent-dir").Exec()
	if err != nil {
	}
}
