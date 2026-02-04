package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"vox-nlu/rabbit"
	"vox-nlu/rasa"
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

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	// Инициализируем Rasa менеджер
	rasaMgr := rasa.NewManager(rasaPath)

	// Инициализируем RabbitMQ consumer
	startConsumer(rabbit.NewConsumer(
		rabbitURL,
		"intents.config.q",
		"intents.config",
		func(msg []rasa.Intent) {
			log.Println("Handling:")
			rasaMgr.Train(msg)
		}), wg, ctx)

	// startConsumer(rabbit.NewConsumer(
	// 	rabbitURL,
	// 	"intents.request.q",
	// 	"intents.request.*",
	// 	func(msg rabbit.Event) {
	// 		log.Println("Handling:", msg.Event)
	// 		store.Upsert(msg.Data)
	// 		rasaMgr.Train(store.All())
	// 	}), wg, ctx)

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
