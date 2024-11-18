package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

// Define a struct to represent the data you want to send
type LogEvent struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message string `json:"message"`
	User    User   `json:"user"`
}

// Define a struct to represent the data you want to send
type User struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Level string `json:"level"`
}

func main() {
	//Set up our kafka using sarama
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	//Create a new producer
	producer, err := sarama.NewSyncProducer([]string{"kafka:9092"}, config)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close()

	//Simulate sending events to Kafka
	for i := 0; i < 100; i++ {

		logEvent := LogEvent{
			ID:      string(i),
			Type:    "UserCreation",
			Message: "User just created bla bla",
			User: User{
				ID:    "userID::" + fmt.Sprint(i),
				Email: fmt.Sprintf("user%d@web.com", i),
				Level: "Admin",
			},
		}

		//Marshall our object to JSON
		logEventJSON, err := json.Marshal(logEvent)
		if err != nil {
			log.Fatal(err)
		}

		msg := &sarama.ProducerMessage{
			Topic: "events",
			Value: sarama.ByteEncoder(logEventJSON),
		}

		//Send event message
		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("Failed to send event: %v", err)
		} else {
			log.Printf("Event sent to partition %d at offset %d", partition, offset)
		}

		//Simulate a 15second delay
		time.Sleep(15 * time.Second)
	}

}
