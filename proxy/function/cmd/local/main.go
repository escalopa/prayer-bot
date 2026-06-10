package main

import (
	"log"
	"os"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/escalopa/prayer-bot/proxy/function"
)

func main() {
	funcframework.RegisterHTTPFunction("/", function.WebhookProxy)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := funcframework.Start(port); err != nil {
		log.Fatal(err)
	}
}
