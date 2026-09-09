package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/centrifugal/centrifuge-go"
	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	_ "github.com/go-sql-driver/mysql"
	duck "github.com/jmbenlloch/next_duck/pkg"
	pbconnect "github.com/jmbenlloch/next_duck/rpc/api/apiconnect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var logger duck.DuckLogger
var configFilename string

// corsMiddleware adds CORS headers to all responses
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Connect-Protocol-Version, Connect-Timeout-Ms")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Expose-Headers", "Connect-Protocol-Version, Connect-Timeout-Ms")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("method=%s, uri=%s, remote_ip=%s", r.Method, r.RequestURI, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

func shouldForceStopFromMessage(message *duck.Message, localHost string) bool {
	if message == nil {
		return false
	}
	if message.Type != duck.MessageError || !message.StopProcesses {
		return false
	}
	// The API subscribes to the global error stream. Ignore its own error
	// publications to avoid recursive force-stop loops from local control errors.
	return message.Host != localHost
}

func main() {
	configFilenameFlag := flag.String("config", "", "Configuration file path")
	printMessages := flag.Bool("messages", false, "Print to stdout Centrifuge messages")
	debug := flag.Bool("debug", false, "Change port to 1324 instead of 1323")
	portFlag := flag.Int("port", 0, "Port to listen on (default: 1323, or 1324 if debug)")
	flag.Parse()
	sentryEnabled := initializeSentry()
	if sentryEnabled {
		defer sentry.Flush(sentryFlushTimeout)
	}

	configFilename = *configFilenameFlag
	configuration, err := duck.ReadConfigurationFile(configFilename)
	if err != nil {
		captureFatalSentryError(err, "read-configuration")
		panic(err)
	}
	// Use retry logic with 10 retries and 1 second initial delay
	// Total max wait time: ~63 seconds (1+2+4+8+16+32)
	db, err := duck.ParseDatabaseConfigurationWithRetry(configuration.Database, 10, 1*time.Second, nil, nil)
	if err != nil {
		captureFatalSentryError(err, "connect-database")
		panic(err)
	}

	userCentrifugal := "api"

	var expiration int64 = 0
	user := "js"
	centrifugalToken := duck.ConnToken(user, expiration, configuration.Centrifugal.Token)

	apiServer := NewDuckAPIServer(db, configFilename, centrifugalToken, *debug)

	fn := func(message *duck.Message) {
		if message.Type == duck.MessageMetric {
			return
		}
		if *printMessages {
			log.Printf("Message: %s", message)
		}
		if shouldForceStopFromMessage(message, "api") {
			log.Printf(
				"force-stop triggered by message host=%s type=%s stop_processes=%t value=%q run=%d",
				message.Host,
				message.Type,
				message.StopProcesses,
				message.Value,
				message.RunNumber,
			)
			ok, _ := apiServer.runTransition.tryBeginStop()
			if ok {
				hub := sentry.CurrentHub().Clone()
				go func() {
					defer recoverToErrorWithHub(&apiServer.runTransition, hub, "automatic-force-stop")
					forceStopProcesses(apiServer)
					apiServer.runTransition.setDone()
				}()
			}
		}
	}
	var sub *centrifuge.Subscription
	if configuration.Centrifugal.Port != 0 {
		var errSub error
		sub, errSub = duck.CreateNewSubscriptionWithFnReadout(configuration.Centrifugal, userCentrifugal, fn)
		if errSub != nil {
			captureFatalSentryError(errSub, "connect-centrifuge")
			panic(errSub)
		}
	}
	logger = duck.NewDuckLogger("api", sub, configuration.LogLevel)
	path, handler := pbconnect.NewDuckAPIHandler(
		apiServer,
		connect.WithInterceptors(sentryConnectInterceptor()),
	)

	// Create HTTP mux and mount ConnectRPC handler
	mux := http.NewServeMux()

	// Mount ConnectRPC handler with /daq prefix for reverse proxy compatibility
	// Use StripPrefix to remove /daq before passing to the ConnectRPC handler
	pathPrefix := "/daq"
	mux.Handle(pathPrefix+path, http.StripPrefix(pathPrefix, handler))

	// Add health check endpoint for Docker healthchecks (with prefix)
	mux.HandleFunc(pathPrefix+"/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Also keep health check at root for backward compatibility
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Apply middleware
	handlerWithMiddleware := loggingMiddleware(corsMiddleware(mux))
	sentryMiddleware := sentryhttp.New(sentryhttp.Options{
		Repanic:         true,
		WaitForDelivery: false,
	})
	instrumentedHandler := sentryMiddleware.Handle(handlerWithMiddleware)

	// Start server with HTTP/2 support for ConnectRPC
	port := 1323
	if *debug {
		port = 1324
	}
	if *portFlag != 0 {
		port = *portFlag
	}

	// Use h2c (HTTP/2 Cleartext) to support both HTTP/1.1 and HTTP/2 without TLS
	h2s := &http2.Server{}
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: h2c.NewHandler(instrumentedHandler, h2s),
	}

	log.Printf("Starting server on :%d with ConnectRPC endpoints", port)
	err = server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		captureFatalSentryError(err, "serve-http")
		log.Fatal(err)
	}
}
