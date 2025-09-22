package main

import (
	"log"

	"github.com/sanchitrk/gosh"
)

func main() {
	// Configure global zerolog settings
	gosh.ConfigureGlobals()

	err := gosh.New().
		Command("docker").
		Arg("build").
		Arg("-t").
		Arg("mcping:1").
		Arg("../../a0dotrun/mcping/").
		Stream()

	if err != nil {
		log.Printf("Error: %v", err)
	}
}
