package main

import (
	"encoding/json"
	"fmt"
	"log"

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

	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.AutoCommit.Enable = false // Disable auto commit for manual offset management

	brokers := []string{"kafka:9092"}
	//Setup our consumer
	consumer, err := sarama.NewConsumer(brokers, config)

	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	//Setup our dead Letter Producer
	dlqProducer, err := sarama.NewSyncProducer(brokers, nil)
	if err != nil {
		log.Fatalf("Failed to create DLQ producer: %v", err)
	}
	defer dlqProducer.Close()

	//We spawn multiple consumers to process our events
	for i := 0; i < 1; i++ {
		go func(consumerId int) {
			// Start consuming from the data-pipeline topic
			partitionConsumer, err := consumer.ConsumePartition("events", 0, sarama.OffsetOldest)
			if err != nil {
				log.Fatal(err)
			}
			defer partitionConsumer.Close()

			for msg := range partitionConsumer.Messages() {
				var logEvent LogEvent
				err := json.Unmarshal(msg.Value, &logEvent)
				if err != nil {
					log.Fatal(err)
				}
				if logEvent.ID == "50" {
					log.Fatal("Simulated uncaught error at event 30")
				}
				//One can do here some data transformations
				err = processMessage(msg, consumerId)
				if err != nil {
					sendToDeadLetterQueue(dlqProducer, msg)
					continue
				}

				fmt.Printf("Consumer %d received LogEvnet: %s\n", consumerId, logEvent)
			}

			// Handle errors from the consumer
			go func() {
				for err := range partitionConsumer.Errors() {
					log.Printf("Consumer error: %v", err)
				}
			}()

		}(i)

	}

}

// processMessage unmarshals and processes the Kafka message
func processMessage(msg *sarama.ConsumerMessage, consumerId int) error {
	var logEvent LogEvent
	err := json.Unmarshal(msg.Value, &logEvent)
	if err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if logEvent.ID == "30" {
		log.Fatal("OOOps snapped at 30")
	}

	// Simulate message processing (e.g., save to database or transform)
	log.Printf("Consumer: %d Processed LogEvent: %+v", consumerId, logEvent)
	return nil
}

// sendToDeadLetterQueue sends problematic messages to the DLQ topic
func sendToDeadLetterQueue(producer sarama.SyncProducer, msg *sarama.ConsumerMessage) {
	dlqMessage := &sarama.ProducerMessage{
		Topic: "dead_letter_queue",
		Key:   sarama.ByteEncoder(msg.Key),   // Preserve original key
		Value: sarama.ByteEncoder(msg.Value), // Preserve original value
	}

	partition, offset, err := producer.SendMessage(dlqMessage)
	if err != nil {
		log.Printf("Failed to send message to DLQ: %v", err)
	} else {
		log.Printf("Message sent to DLQ: partition %d, offset %d", partition, offset)
	}
}
