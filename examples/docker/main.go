package main

import (
	"log"

	"github.com/sanchitrk/gosh"
)

func main() {
	gosh.ConfigureGlobals()
	gs := gosh.New()

	httpURL := "https://localapi.a0.run/internal/v1/deployments/log"

	headers := map[string]string{
		"Authorization": "Bearer Token",
		"Content-Type":  "application/json",
	}
	logKVs := map[string]string{
		"deploymentId": "g7qossr9gdmcts4u5rfz7",
	}

	for k, v := range headers {
		gs.AddHTTPHeader(k, v)
	}
	for k, v := range logKVs {
		gs.LogKV(k, v)
	}

	gs = gs.WithHTTPStream(httpURL)

	err := gs.
		Command("docker").
		Arg("build").
		Arg("-t").
		Arg("mcping:latest").
		Arg("../../a0dotrun/mcping/").
		Stream()

	if err != nil {
		log.Printf("Error: %v", err)
	}
}
