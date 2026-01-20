package main

import (
	"log"
	"os"

	"vox-nlu/rabbit"
	"vox-nlu/rasa"
)

func main() {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Fatal("RABBITMQ_URL not set")
	}

	// Инициализируем хранилище интентов
	store := rasa.NewStore()

	// Инициализируем Rasa менеджер
	rasaMgr := rasa.NewManager("/app/rasa")

	// Инициализируем RabbitMQ consumer
	consumer := rabbit.NewConsumer(rabbitURL, func(msg rabbit.Event) {
		switch msg.Event {
		case "nlu.command.upsert_intent":
			store.Upsert(msg.Data)
		case "nlu.command.train_all":
			rasaMgr.Train(store.All())
		}
	})

	log.Println("NLU adapter started")
	if err := consumer.Start(); err != nil {
		log.Fatal(err)
	}
}
