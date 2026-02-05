package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"vox-nlu/rabbit"
	"vox-nlu/rasa"

	"github.com/streadway/amqp"
)

var Version = "dev"

func main() {
	rabbitURL := os.Getenv("RABBIT_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://guest:guest@localhost:5672/"
	}

	rasaPath := os.Getenv("RASA_PATH")
	if rasaPath == "" {
		rasaPath = "/rasa"
	}

	log.Println("Vox NLU version:", Version)

	// Инициализируем Rasa менеджер
	rasaMgr := rasa.NewManager("rasa", rasaPath)
	rasaMgr.Start()
	defer rasaMgr.Stop()

	// Инициализируем RabbitMQ connection
	conn, err := rabbit.NewConnection(rabbitURL)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Инициализируем RabbitMQ publisher
	pub, err := rabbit.NewPublisher(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer pub.Close()

	// Инициализируем RabbitMQ consumer
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	startConsumer(rabbit.NewConsumer(
		conn,
		"intents.config.q",
		"intents.config",
		func(msg []rasa.TrainIntents, data amqp.Delivery) bool {
			log.Printf("Handling: %s\n", data.RoutingKey)
			rasaMgr.Train(msg)
			return true
		}), wg, ctx)

	startConsumer(rabbit.NewConsumer(
		conn,
		"intents.request.q",
		"intents.request.*",
		func(msg rasa.ProcessText, data amqp.Delivery) bool {
			log.Printf("Handling: %s\n", data.RoutingKey)
			res, err := rasaMgr.Parse(msg)
			if err != nil {
				log.Println("Error on parse rasa:", err)
				return false
			}
			log.Println("Result:", res)
			pubRouting := strings.Replace(data.RoutingKey, "request", "response", 1)
			pub.PublishJSON(ctx, "intents", pubRouting, res)
			log.Println("published")
			return true
		}), wg, ctx)

	log.Println("NLU adapter started")

	// ждём SIGINT / SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	log.Println("shutting down...")
	cancel()
	wg.Wait()
}

func startConsumer[T any](c *rabbit.Consumer[T], wg *sync.WaitGroup, ctx context.Context) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := c.Start(ctx); err != nil {
			log.Println(err)
		}
	}()
}
