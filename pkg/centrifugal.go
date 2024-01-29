package duck

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	_ "net/http/pprof"

	"github.com/centrifugal/centrifuge-go"
	"github.com/golang-jwt/jwt"
)

const CENTRIFUGE_TOPIC = "duck"

// MessagePublisher defines the interface for publishing messages to a subscription.
// This allows for mocking in tests without requiring a real Centrifuge connection.
type MessagePublisher interface {
	Publish(ctx context.Context, data []byte) error
}

// CentrifugePublisher wraps a centrifuge.Subscription to implement MessagePublisher.
type CentrifugePublisher struct {
	Sub *centrifuge.Subscription
}

// Publish sends data to the Centrifuge subscription.
func (p *CentrifugePublisher) Publish(ctx context.Context, data []byte) error {
	_, err := p.Sub.Publish(ctx, data)
	return err
}

type Metrics struct {
	EventCounter    int
	ByteCounter     int
	CurrentTrgRate  float64
	CurrentDataRate float64
	AvgTrgRate      float64
	AvgDataRate     float64
}

type State struct {
	State string `json:"state"`
}

type OutputFile struct {
	Server string `json:"server"`
	Subrun int    `json:"subrun"`
}

func ConnToken(user string, exp int64, token string) string {
	// NOTE that JWT must be generated on backend side of your application!
	// Here we are generating it on client side only for example simplicity.
	claims := jwt.MapClaims{"sub": user}
	if exp > 0 {
		claims["exp"] = exp
	}
	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(token))
	if err != nil {
		panic(err)
	}
	return t
}

func CentrifugeConnection(config CentrifugalConfiguration, user string) (*centrifuge.Client, error) {
	address := fmt.Sprintf("ws://%s:%d/connection/websocket", config.Host, config.Port)

	// If token is empty, don't send any token (for insecure mode)
	clientConfig := centrifuge.Config{}
	if config.Token != "" {
		clientConfig.Token = ConnToken(user, 0, config.Token)
	}

	client := centrifuge.NewJsonClient(address, clientConfig)

	//client.OnConnecting(func(e centrifuge.ConnectingEvent) {
	//	log.Printf("Connecting - %d (%s)", e.Code, e.Reason)
	//})
	//client.OnConnected(func(e centrifuge.ConnectedEvent) {
	//	log.Printf("Connected with ID %s", e.ClientID)
	//})
	//client.OnDisconnected(func(e centrifuge.DisconnectedEvent) {
	//	log.Printf("Disconnected: %d (%s)", e.Code, e.Reason)
	//})

	client.OnError(func(e centrifuge.ErrorEvent) {
		log.Printf("Error: %s", e.Error.Error())
	})

	//client.OnMessage(func(e centrifuge.MessageEvent) {
	//	log.Printf("Message from server: %s", string(e.Data))
	//})

	//client.OnSubscribed(func(e centrifuge.ServerSubscribedEvent) {
	//	log.Printf("Subscribed to server-side channel %s: (was recovering: %v, recovered: %v)", e.Channel, e.WasRecovering, e.Recovered)
	//})
	//client.OnSubscribing(func(e centrifuge.ServerSubscribingEvent) {
	//	log.Printf("Subscribing to server-side channel %s", e.Channel)
	//})
	//client.OnUnsubscribed(func(e centrifuge.ServerUnsubscribedEvent) {
	//	log.Printf("Unsubscribed from server-side channel %s", e.Channel)
	//})

	//client.OnPublication(func(e centrifuge.ServerPublicationEvent) {
	//	log.Printf("Publication from server-side channel %s: %s (offset %d)", e.Channel, e.Data, e.Offset)
	//})
	//client.OnJoin(func(e centrifuge.ServerJoinEvent) {
	//	log.Printf("Join to server-side channel %s: %s (%s)", e.Channel, e.User, e.Client)
	//})
	//client.OnLeave(func(e centrifuge.ServerLeaveEvent) {
	//	log.Printf("Leave from server-side channel %s: %s (%s)", e.Channel, e.User, e.Client)
	//})

	err := client.Connect()
	return client, err
}

func SubscribeCentrifuge(client *centrifuge.Client, fn func(*Message)) (*centrifuge.Subscription, error) {
	sub, err := client.NewSubscription(CENTRIFUGE_TOPIC, centrifuge.SubscriptionConfig{
		Recoverable: true,
		JoinLeave:   true,
	})
	if err != nil {
		log.Fatalln("error creating subscription:", err)
		return nil, err
	}

	//sub.OnSubscribing(func(e centrifuge.SubscribingEvent) {
	//	log.Printf("Subscribing on channel %s - %d (%s)", sub.Channel, e.Code, e.Reason)
	//})
	//sub.OnSubscribed(func(e centrifuge.SubscribedEvent) {
	//	log.Printf("Subscribed on channel %s, (was recovering: %v, recovered: %v)", sub.Channel, e.WasRecovering, e.Recovered)
	//})
	//sub.OnUnsubscribed(func(e centrifuge.UnsubscribedEvent) {
	//	log.Printf("Unsubscribed from channel %s - %d (%s)", sub.Channel, e.Code, e.Reason)
	//})

	sub.OnError(func(e centrifuge.SubscriptionErrorEvent) {
		log.Printf("Subscription error %s: %s", sub.Channel, e.Error)
	})

	if fn != nil {
		sub.OnPublication(func(e centrifuge.PublicationEvent) {
			var message *Message
			err := json.Unmarshal(e.Data, &message)
			if err != nil {
				return
			}
			fn(message)
		})
	}
	//sub.OnJoin(func(e centrifuge.JoinEvent) {
	//	log.Printf("Someone joined %s: user id %s, client id %s", sub.Channel, e.User, e.Client)
	//})
	//sub.OnLeave(func(e centrifuge.LeaveEvent) {
	//	log.Printf("Someone left %s: user id %s, client id %s", sub.Channel, e.User, e.Client)
	//})

	err = sub.Subscribe()
	if err != nil {
		log.Fatalln("error subscribing: ", err)
	}
	return sub, err
}

func CreateNewSubscription(config CentrifugalConfiguration, user string) (*centrifuge.Subscription, error) {
	centrifugeConn, err := CentrifugeConnection(config, user)
	if err != nil {
		return nil, err
	}
	sub, err := SubscribeCentrifuge(centrifugeConn, nil)
	return sub, err
}

func CreateNewSubscriptionWithFnReadout(config CentrifugalConfiguration, user string, fn func(*Message)) (*centrifuge.Subscription, error) {
	centrifugeConn, err := CentrifugeConnection(config, user)
	if err != nil {
		return nil, err
	}
	sub, err := SubscribeCentrifuge(centrifugeConn, fn)
	return sub, err
}

func PublishMessage(publisher MessagePublisher, message Message) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	return publisher.Publish(context.Background(), data)
}
