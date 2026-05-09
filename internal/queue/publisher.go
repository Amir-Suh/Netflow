package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const TransactionIngestedRoutingKey = "transactions.ingested"

type TransactionEvent struct {
	EventType         string    `json:"event_type"`
	TransactionID     int64     `json:"transaction_id"`
	UserID            int64     `json:"user_id"`
	AccountID         int64     `json:"account_id"`
	SourceEnvironment string    `json:"source_environment"`
	OccurredAt        time.Time `json:"occurred_at"`
}

type Publisher struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

func NewPublisher(url, exchange, queueName string) (*Publisher, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open rabbitmq channel: %w", err)
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("declare exchange: %w", err)
	}
	if _, err := channel.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("declare queue: %w", err)
	}
	if err := channel.QueueBind(queueName, TransactionIngestedRoutingKey, exchange, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("bind queue: %w", err)
	}
	return &Publisher{conn: conn, channel: channel, exchange: exchange}, nil
}

func NewPublisherWithRetry(ctx context.Context, url, exchange, queueName string, delay time.Duration, attempts int) (*Publisher, error) {
	var lastErr error
	for attempt := 1; attempts <= 0 || attempt <= attempts; attempt++ {
		publisher, err := NewPublisher(url, exchange, queueName)
		if err == nil {
			return publisher, nil
		}
		lastErr = err

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("connect rabbitmq canceled: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("connect rabbitmq after %d attempts: %w", attempts, lastErr)
}

func (p *Publisher) Close() {
	if p.channel != nil {
		p.channel.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}
}

func (p *Publisher) PublishTransactionIngested(ctx context.Context, event TransactionEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal transaction event: %w", err)
	}
	return p.channel.PublishWithContext(ctx, p.exchange, TransactionIngestedRoutingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Timestamp:    time.Now().UTC(),
		Body:         body,
	})
}
