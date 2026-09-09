package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jmbenlloch/next_duck/pkg/database"
	amqp "github.com/rabbitmq/amqp091-go"
)

type topiPublisher interface {
	Publish(context.Context, database.Topiparam, []byte) error
}
type amqpTopiPublisher struct{}

func (amqpTopiPublisher) Publish(ctx context.Context, p database.Topiparam, body []byte) error {
	u := &url.URL{Scheme: "amqp", Host: net.JoinHostPort(p.RabbitmqAddress, strconv.Itoa(int(p.RabbitmqPort))), Path: p.RabbitmqVhost}
	u.User = url.UserPassword(p.RabbitmqUser, p.RabbitmqPassword)
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := amqp.DialConfig(u.String(), amqp.Config{Dial: func(network, addr string) (net.Conn, error) {
		c, err := dialer.DialContext(ctx, network, addr)
		if err == nil {
			if deadline, ok := ctx.Deadline(); ok {
				err = c.SetDeadline(deadline)
			}
		}
		return c, err
	}})
	if err != nil {
		return fmt.Errorf("connect to RabbitMQ: %w", err)
	}
	defer conn.Close()
	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	defer ch.Close()
	if err = ch.ExchangeDeclare(p.ExchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if _, err = ch.QueueDeclare(p.ControlQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}
	if err = ch.QueueBind(p.ControlQueue, p.ControlQueue, p.ExchangeName, false, nil); err != nil {
		return fmt.Errorf("bind queue: %w", err)
	}
	if err = ch.Confirm(false); err != nil {
		return fmt.Errorf("enable publisher confirms: %w", err)
	}
	confirms := ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	if err = ch.PublishWithContext(ctx, p.ExchangeName, p.ControlQueue, false, false, amqp.Publishing{ContentType: "application/json", DeliveryMode: amqp.Persistent, Body: body}); err != nil {
		return err
	}
	select {
	case confirmation := <-confirms:
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ rejected publication")
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *DuckAPIServer) notifyTopi(operation string, run int32) {
	if s.topiPublisher == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p, err := s.queries.GetTopiParams(ctx)
		if err != nil {
			if err != context.Canceled {
				logger.Slog.Warn("TOPI notification skipped", "operation", operation, "run", run, "error", err)
			}
			return
		}
		if !p.Enabled {
			return
		}
		msg := struct {
			RunNumber int32  `json:"run_number"`
			Operation string `json:"operation"`
			Config    string `json:"config,omitempty"`
		}{RunNumber: run, Operation: operation}
		if operation == "start" {
			msg.Config = p.SelectedConfiguration
			if msg.Config == "" {
				logger.Slog.Warn("TOPI start notification skipped: no configuration selected", "run", run)
				return
			}
		}
		body, err := json.Marshal(msg)
		if err != nil {
			return
		}
		if err := s.topiPublisher.Publish(ctx, p, body); err != nil {
			logger.Slog.Warn("TOPI notification failed; run control is unaffected", "operation", operation, "run", run, "error", err)
		}
	}()
}
