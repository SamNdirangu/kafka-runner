package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

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

// ConsumerGroupHandler handles consumed messages
type ConsumerGroupHandler struct {
	producer sarama.SyncProducer
}

func (ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	// TODO read more: Not needed for now
	fmt.Println("Consumer group setup complete")
	return nil
}
func (ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	// TODO read more: Cleaing up after consumer group is closed
	fmt.Println("Consumer group cleanup complete")
	return nil
}

func (handler *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// This method is called to process each message in the topic
	consumerIdString, nil := os.LookupEnv("CONSUMER_ID")
	var consumerId int = 0
	if id, err := strconv.Atoi(consumerIdString); err == nil {
		consumerId = id
	}

	fmt.Println("Consumer messages... ConsumerID: ", consumerId)

	for msg := range claim.Messages() {
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
			sendToDeadLetterQueue(handler, msg)
			continue
		}
		// Acknowledge the message to Kafka

		fmt.Printf("Consumer %d received LogEvnet: %s\n", consumerId, logEvent)
		session.MarkMessage(msg, "")
	}

	return nil
}

func main() {
	//Setup our kafka brokers
	brokers := []string{"kafka1:9093", "kafka2:9093", "kafka3:9093"}

	// Prodoucer Sections -------------------------------------------
	producerConfig := sarama.NewConfig()
	producerConfig.Producer.Return.Successes = true
	producerConfig.Producer.Return.Errors = true

	// Create our DLQ Producer
	dlqProducer, err := sarama.NewSyncProducer(brokers, producerConfig)
	if err != nil {
		log.Fatalf("Failed to create DLQ producer: %v", err)
	}
	defer dlqProducer.Close()

	// Consumer Section -----------------------------------------------------------
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true
	config.Consumer.Offsets.AutoCommit.Enable = false // Disable auto commit for manual offset management

	//Create our consumer group
	consumerGroup, err := sarama.NewConsumerGroup(brokers, "event-001", config)
	if err != nil {
		log.Fatal(err)
	}
	defer consumerGroup.Close()

	// ------------------------------------------------------------------

	//Define a hanlder for processing messages
	handler := ConsumerGroupHandler{
		producer: dlqProducer,
	}

	//Subscribe to a topic
	for {
		if err := consumerGroup.Consume(nil, []string{"events"}, handler); err != nil {
			log.Fatal("Error while consuming:", err)
		}
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
func (handler *ConsumerGroupHandler) sendToDeadLetterQueue(producer sarama.SyncProducer, msg *sarama.ConsumerMessage) {

	dlqMessage := &sarama.ProducerMessage{
		Topic: "dead_letter_queue",
		Key:   sarama.ByteEncoder(msg.Key),   // Preserve original key
		Value: sarama.ByteEncoder(msg.Value), // Preserve original value
	}

	partition, offset, err := handler.producer.SendMessage(dlqMessage)
	if err != nil {
		log.Printf("Failed to send message to DLQ: %v", err)
	} else {
		log.Printf("Message sent to DLQ: partition %d, offset %d", partition, offset)
	}
}
