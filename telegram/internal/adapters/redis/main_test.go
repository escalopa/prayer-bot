package redis

import (
	"context"
	"log"
	"testing"

	"github.com/escalopa/gopray/telegram/test/testcon"
)

var testRedisURL string

func TestMain(m *testing.M) {
	url, terminate, err := testcon.NewRedisContainer(context.Background())
	if err != nil {
		log.Fatalf("failed to start redis container: %v", err)
	}
	testRedisURL = url
	defer func() {
		err = terminate()
		if err != nil {
			log.Fatalf("failed to terminate redis container: %v", err)
		}
	}()
	m.Run()
}

func testContext() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}
