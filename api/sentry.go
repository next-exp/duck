package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/getsentry/sentry-go"
)

const sentryFlushTimeout = 2 * time.Second

// release is set at build time with -X main.release=<git commit>.
var release = "development"

func initializeSentry() bool {
	dsn := strings.TrimSpace(os.Getenv("SENTRY_DSN"))
	if dsn == "" {
		return false
	}

	transport := sentry.NewHTTPTransport()
	transport.Timeout = 3 * time.Second
	transport.BufferSize = 100

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		Environment:      os.Getenv("SENTRY_ENVIRONMENT"),
		Release:          release,
		AttachStacktrace: true,
		SendDefaultPII:   false,
		EnableTracing:    false,
		TracesSampleRate: 0,
		MaxBreadcrumbs:   50,
		Transport:        transport,
		Tags:             map[string]string{"service": "api"},
		BeforeSend:       scrubSentryEvent,
	})
	if err != nil {
		// Do not use DuckLogger here: Sentry availability must never become a
		// stopping error or prevent the API from starting.
		log.Printf("Sentry disabled: initialization failed")
		return false
	}
	log.Printf("Sentry error reporting enabled")
	return true
}

func scrubSentryEvent(event *sentry.Event, _ *sentry.EventHint) *sentry.Event {
	if event == nil {
		return nil
	}
	if event.Request != nil {
		event.Request.Data = ""
		event.Request.QueryString = ""
		event.Request.Cookies = ""
		for name := range event.Request.Headers {
			switch strings.ToLower(name) {
			case "authorization", "cookie", "set-cookie", "x-api-key":
				event.Request.Headers[name] = "[Filtered]"
			}
		}
	}
	for key := range event.Extra {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "password") || strings.Contains(lower, "passwd") ||
			strings.Contains(lower, "token") || strings.Contains(lower, "secret") ||
			strings.Contains(lower, "dsn") {
			event.Extra[key] = "[Filtered]"
		}
	}
	return event
}

func cloneSentryHub(ctx context.Context) *sentry.Hub {
	if ctx != nil {
		if hub := sentry.GetHubFromContext(ctx); hub != nil {
			return hub.Clone()
		}
	}
	return sentry.CurrentHub().Clone()
}

func captureSentryException(ctx context.Context, err error, tags map[string]string) {
	if err == nil {
		return
	}
	hub := cloneSentryHub(ctx)
	hub.WithScope(func(scope *sentry.Scope) {
		for key, value := range tags {
			scope.SetTag(key, value)
		}
		hub.CaptureException(err)
	})
}

func captureFatalSentryError(err error, stage string) {
	captureSentryException(context.Background(), err, map[string]string{"stage": stage, "fatal": "true"})
	sentry.Flush(sentryFlushTimeout)
}

func shouldReportConnectError(err error) bool {
	switch connect.CodeOf(err) {
	case connect.CodeUnknown, connect.CodeInternal, connect.CodeDataLoss:
		return true
	default:
		return false
	}
}

func sentryConnectInterceptor() connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			response, err := next(ctx, req)
			if err != nil && shouldReportConnectError(err) {
				captureSentryException(ctx, err, map[string]string{
					"procedure":    req.Spec().Procedure,
					"connect_code": connect.CodeOf(err).String(),
				})
			}
			return response, err
		}
	})
}

func recoverBackgroundPanic(hub *sentry.Hub, tags map[string]string) {
	if recovered := recover(); recovered != nil {
		if hub == nil {
			hub = sentry.CurrentHub().Clone()
		}
		hub.WithScope(func(scope *sentry.Scope) {
			for key, value := range tags {
				scope.SetTag(key, value)
			}
			hub.Recover(recovered)
		})
		// Standard logging only: a monitoring path must not emit a Duck
		// stopping-error message.
		log.Printf("recovered background panic in %s", tags["operation"])
	}
}
