package rabbit

import (
	"log"

	"github.com/streadway/amqp"
)

type Event struct {
	Event string
	Data  map[string]interface{}
}

type Consumer struct {
	url     string
	handler func(Event)
	conn    *amqp.Connection
	ch      *amqp.Channel
	queue   string
}

func NewConsumer(url string, handler func(Event)) *Consumer {
	return &Consumer{
		url:     url,
		handler: handler,
		queue:   "nlu.commands",
	}
}

func (c *Consumer) Start() error {
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

	msgs, err := ch.Consume(
		q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	forever := make(chan bool)
	go func() {
		for d := range msgs {
			// TODO: unmarshal JSON into Event
			log.Printf("Received message: %s", d.Body)
		}
	}()
	log.Println("Waiting for messages...")
	<-forever
	return nil
}
