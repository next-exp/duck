package main

import (
	"errors"
	"testing"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
)

func TestScrubSentryEvent(t *testing.T) {
	event := &sentry.Event{
		Request: &sentry.Request{
			Data:        `{"password":"secret"}`,
			QueryString: "token=secret",
			Cookies:     "session=secret",
			Headers: map[string]string{
				"Authorization": "Bearer secret",
				"Cookie":        "session=secret",
				"X-Api-Key":     "secret",
				"Accept":        "application/json",
			},
		},
		Extra: map[string]interface{}{
			"api_token": "secret",
			"password":  "secret",
			"detail":    "safe",
		},
	}

	scrubbed := scrubSentryEvent(event, nil)
	if scrubbed.Request.Data != "" || scrubbed.Request.QueryString != "" || scrubbed.Request.Cookies != "" {
		t.Fatal("request body, query string, or cookies were not cleared")
	}
	for _, header := range []string{"Authorization", "Cookie", "X-Api-Key"} {
		if got := scrubbed.Request.Headers[header]; got != "[Filtered]" {
			t.Errorf("header %s = %q, want [Filtered]", header, got)
		}
	}
	if got := scrubbed.Request.Headers["Accept"]; got != "application/json" {
		t.Errorf("safe header changed to %q", got)
	}
	if got := scrubbed.Extra["api_token"]; got != "[Filtered]" {
		t.Errorf("api_token = %q, want [Filtered]", got)
	}
	if got := scrubbed.Extra["password"]; got != "[Filtered]" {
		t.Errorf("password = %q, want [Filtered]", got)
	}
	if got := scrubbed.Extra["detail"]; got != "safe" {
		t.Errorf("safe extra changed to %q", got)
	}
}

func TestShouldReportConnectError(t *testing.T) {
	tests := []struct {
		code connect.Code
		want bool
	}{
		{connect.CodeUnknown, true},
		{connect.CodeInternal, true},
		{connect.CodeDataLoss, true},
		{connect.CodeInvalidArgument, false},
		{connect.CodeNotFound, false},
		{connect.CodeUnavailable, false},
	}

	for _, test := range tests {
		t.Run(test.code.String(), func(t *testing.T) {
			err := connect.NewError(test.code, errors.New("test error"))
			if got := shouldReportConnectError(err); got != test.want {
				t.Errorf("shouldReportConnectError(%s) = %t, want %t", test.code, got, test.want)
			}
		})
	}
}

func TestRecoverToErrorWithHubCapturesPanic(t *testing.T) {
	transport := &sentry.MockTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	hub := sentry.NewHub(client, sentry.NewScope())
	runTransition := &RunTransition{}

	func() {
		defer recoverToErrorWithHub(runTransition, hub, "start-run")
		panic("boom")
	}()

	state, message := runTransition.Status()
	if state != TransitionError || message != "panic: boom" {
		t.Fatalf("transition status = (%v, %q), want (%v, %q)", state, message, TransitionError, "panic: boom")
	}
	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d events, want 1", len(events))
	}
	if got := events[0].Tags["operation"]; got != "start-run" {
		t.Errorf("operation tag = %q, want start-run", got)
	}
}

func TestRecoverBackgroundPanicCapturesPanic(t *testing.T) {
	transport := &sentry.MockTransport{}
	client, err := sentry.NewClient(sentry.ClientOptions{
		Dsn:       "https://public@example.com/1",
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}
	hub := sentry.NewHub(client, sentry.NewScope())

	func() {
		defer recoverBackgroundPanic(hub, map[string]string{"operation": "test-background"})
		panic("boom")
	}()

	events := transport.Events()
	if len(events) != 1 {
		t.Fatalf("captured %d events, want 1", len(events))
	}
	if got := events[0].Tags["operation"]; got != "test-background" {
		t.Errorf("operation tag = %q, want test-background", got)
	}
}
