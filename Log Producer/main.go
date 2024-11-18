package main

import (
	"fmt"
	"log"
	"time"

	"github.com/IBM/sarama"
)

// Log simulator aggregatror to Kafka
// This will send logs to our Kafka

func main() {

	// Set up our kafka configuration
	config := sarama.NewConfig()
	config.Producer.Return.Successes = true

	//Create a new producer
	// A synhrounous producer waits for confirmation of message delivery before proceeding
	producer, err := sarama.NewSyncProducer([]string{"localhost:9092"}, config)
	if err != nil {
		log.Fatal(err)
	}
	defer producer.Close() //Close our producer once the function completes ie our main function

	// Simulate log aggregation and sending logs to Kafka
	for i := 0; i < 100; i++ {
		// Set up our message
		msg := &sarama.ProducerMessage{
			Topic: "logs",
			Value: sarama.StringEncoder(fmt.Sprintf("Log message %d at %s", i, time.Now().String())),
		}

		//Send the log message
		partition, offset, err := producer.SendMessage(msg)
		if err != nil {
			log.Printf("Failed to send message: %v", err)
		} else {
			log.Printf("Message sent to partition %d at offset %d", partition, offset)
		}

		//Simulate a delay
		time.Sleep(15 * time.Second)
	}
}
