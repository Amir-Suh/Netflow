package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	categorizedQueueName     = "netflow.api.categorized"
	arbitrageResultQueueName = "netflow.api.arbitrage.results"
)

// WorkerEvent is a categorization or arbitrage result delivered from a Python
// worker through RabbitMQ. UserID lets the WebSocket hub fan-out to the right
// browser session.
type WorkerEvent struct {
	EventType     string          `json:"event_type"`
	UserID        int64           `json:"user_id"`
	TransactionID int64           `json:"transaction_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

type Consumer struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	exchange string
}

func NewConsumer(url, exchange string) (*Consumer, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("consumer connect rabbitmq: %w", err)
	}
	channel, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("consumer open channel: %w", err)
	}
	if err := channel.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		channel.Close()
		conn.Close()
		return nil, fmt.Errorf("consumer declare exchange: %w", err)
	}
	for queueName, routingKey := range map[string]string{
		categorizedQueueName:     TransactionCategorizedRoutingKey,
		arbitrageResultQueueName: ArbitrageResultRoutingKey,
	} {
		if _, err := channel.QueueDeclare(queueName, true, false, false, false, nil); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("consumer declare queue %s: %w", queueName, err)
		}
		if err := channel.QueueBind(queueName, routingKey, exchange, false, nil); err != nil {
			channel.Close()
			conn.Close()
			return nil, fmt.Errorf("consumer bind queue %s: %w", queueName, err)
		}
	}
	return &Consumer{conn: conn, channel: channel, exchange: exchange}, nil
}

func NewConsumerWithRetry(ctx context.Context, url, exchange string, delay time.Duration, attempts int) (*Consumer, error) {
	var lastErr error
	for attempt := 1; attempts <= 0 || attempt <= attempts; attempt++ {
		consumer, err := NewConsumer(url, exchange)
		if err == nil {
			return consumer, nil
		}
		lastErr = err
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("consumer connect canceled: %w", ctx.Err())
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("consumer connect after %d attempts: %w", attempts, lastErr)
}

func (c *Consumer) Close() {
	if c.channel != nil {
		c.channel.Close()
	}
	if c.conn != nil {
		c.conn.Close()
	}
}

// Consume blocks until ctx is canceled, forwarding decoded WorkerEvents into
// the events channel. The categorized and arbitrage result queues are merged
// into a single stream tagged by EventType.
func (c *Consumer) Consume(ctx context.Context, events chan<- WorkerEvent) error {
	categorized, err := c.channel.Consume(categorizedQueueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume categorized: %w", err)
	}
	arbitrage, err := c.channel.Consume(arbitrageResultQueueName, "", false, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("consume arbitrage results: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-categorized:
			if !ok {
				return fmt.Errorf("categorized channel closed")
			}
			c.dispatch(ctx, events, msg, "transactions.categorized")
		case msg, ok := <-arbitrage:
			if !ok {
				return fmt.Errorf("arbitrage channel closed")
			}
			c.dispatch(ctx, events, msg, "transactions.arbitrage.results")
		}
	}
}

func (c *Consumer) dispatch(ctx context.Context, events chan<- WorkerEvent, msg amqp.Delivery, eventType string) {
	var inner struct {
		TransactionID int64     `json:"transaction_id"`
		UserID        int64     `json:"user_id"`
		OccurredAt    time.Time `json:"occurred_at"`
	}
	_ = json.Unmarshal(msg.Body, &inner)

	event := WorkerEvent{
		EventType:     eventType,
		UserID:        inner.UserID,
		TransactionID: inner.TransactionID,
		OccurredAt:    inner.OccurredAt,
		Payload:       json.RawMessage(msg.Body),
	}
	select {
	case <-ctx.Done():
		_ = msg.Nack(false, true)
	case events <- event:
		_ = msg.Ack(false)
	}
}
