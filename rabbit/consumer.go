package rabbit

import (
	"context"
	"encoding/json"
	"log"

	"github.com/streadway/amqp"
)

type Consumer[T any] struct {
	url        string
	exchange   string
	queue      string
	routingKey string
	handler    func(T)

	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewConsumer[T any](
	url string,
	queue string,
	routingKey string,
	handler func(T),
) *Consumer[T] {
	return &Consumer[T]{
		url:        url,
		exchange:   "intents",
		queue:      queue,
		routingKey: routingKey,
		handler:    handler,
	}
}

func (c *Consumer[T]) Start(ctx context.Context) error {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return err
	}
	c.conn = conn

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	c.ch = ch

	// exchange
	err = ch.ExchangeDeclare(
		c.exchange,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// queue
	q, err := ch.QueueDeclare(
		c.queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	// bind
	err = ch.QueueBind(
		q.Name,
		c.routingKey,
		c.exchange,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	msgs, err := ch.Consume(
		q.Name,
		"",
		false, // manual ack
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	log.Printf(
		"Consumer started: queue=%s routing=%s",
		c.queue,
		c.routingKey,
	)

	go func() {
		for {
			select {
			case d, ok := <-msgs:
				if !ok {
					return
				}
				var e T
				if err := json.Unmarshal(d.Body, &e); err != nil {
					log.Printf("unmarshal error: %v", err)
					d.Nack(false, false)
					continue
				}
				c.handler(e)
				d.Ack(false)

			case <-ctx.Done():
				_ = c.ch.Close()
				_ = c.conn.Close()
				return
			}
		}
	}()

	<-ctx.Done()
	return nil
}
