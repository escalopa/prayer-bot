package botprofile

import (
	"fmt"
	"testing"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func TestIsRateLimited(t *testing.T) {
	rateErr := &bot.TooManyRequestsError{Message: "too many requests", RetryAfter: 43240}
	wrapped := fmt.Errorf("setMyName: %w", rateErr)

	if !isRateLimited(rateErr) {
		t.Fatal("expected direct TooManyRequestsError to be rate limited")
	}
	if !isRateLimited(wrapped) {
		t.Fatal("expected wrapped TooManyRequestsError to be rate limited")
	}
	if isRateLimited(fmt.Errorf("network timeout")) {
		t.Fatal("expected unrelated error not to be rate limited")
	}
}

func TestWrapProfileErr(t *testing.T) {
	rateErr := &bot.TooManyRequestsError{RetryAfter: 60}
	if err := wrapProfileErr("setMyName", rateErr); err != nil {
		t.Fatalf("expected nil for rate limit, got %v", err)
	}
	if err := wrapProfileErr("setMyName", fmt.Errorf("bad request")); err == nil {
		t.Fatal("expected error for non-rate-limit failure")
	}
}

func TestIsTransient(t *testing.T) {
	transient := []error{
		fmt.Errorf("getMyName: error decode response body for method getMyName, , unexpected end of JSON input"),
		fmt.Errorf("read tcp: connection reset by peer"),
		&bot.TooManyRequestsError{RetryAfter: 30},
	}
	for _, err := range transient {
		if !isTransient(err) {
			t.Fatalf("expected transient error: %v", err)
		}
	}

	permanent := []error{
		fmt.Errorf("bad request: chat not found"),
		fmt.Errorf("owner id is required"),
	}
	for _, err := range permanent {
		if isTransient(err) {
			t.Fatalf("expected non-transient error: %v", err)
		}
	}

	if isTransient(nil) {
		t.Fatal("nil error must not be transient")
	}
}

func TestCommandsEqual(t *testing.T) {
	a := []models.BotCommand{{Command: "start", Description: "Start"}}
	b := []models.BotCommand{{Command: "start", Description: "Start"}}
	c := []models.BotCommand{{Command: "help", Description: "Help"}}

	if !commandsEqual(a, b) {
		t.Fatal("expected identical command lists to be equal")
	}
	if commandsEqual(a, c) {
		t.Fatal("expected different command lists not to be equal")
	}
	if commandsEqual(a, append(a, c...)) {
		t.Fatal("expected different lengths not to be equal")
	}
}
