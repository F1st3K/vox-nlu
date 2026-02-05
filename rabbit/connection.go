package rabbit

import (
	"fmt"

	"github.com/streadway/amqp"
)

func NewConnection(url string) (*amqp.Connection, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("amqp dial: %w", err)
	}
	return conn, nil
}
