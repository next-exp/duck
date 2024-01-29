package main

import (
	"flag"
	"fmt"

	duck "github.com/jmbenlloch/next_duck/pkg"
)

//type Message struct {
//	Timestamp     time.Time   `json:"timestamp"`
//	Host          string      `json:"host"`
//	Type          MessageType `json:"type"`
//	Value         string      `json:"value"`
//	StopProcesses bool        `json:"stop_processes"`
//}

// Assuming duck.Message is defined in the imported package, define the String method correctly.

func main() {
	configFilename := flag.String("config", "", "Configuration file path")
	flag.Parse()

	configuration, err := duck.ReadConfigurationFile(*configFilename)
	if err != nil {
		panic(err)
	}

	userCentrifugal := "readLogs"
	fn := func(message *duck.Message) {
		fmt.Printf("%s\n", message)
		if message.Type == duck.MessageMetric {
			return
		}
	}
	_, err = duck.CreateNewSubscriptionWithFnReadout(configuration.Centrifugal, userCentrifugal, fn)
	if err != nil {
		panic(err)
	}

	for {
	}
}
