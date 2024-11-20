package main

import (
	"context"
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
	producer          sarama.SyncProducer
	processedProducer sarama.SyncProducer
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

func main() {
	//Setup our kafka brokers
	brokers := []string{"kafka:9092"}

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

	// Create our Processed Messages Producer
	processedProducer, err := sarama.NewSyncProducer(brokers, producerConfig)
	if err != nil {
		log.Fatalf("Failed to create processed messages producer: %v", err)
	}
	defer processedProducer.Close()

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
		producer:          dlqProducer,
		processedProducer: processedProducer,
	}

	//Subscribe to a topic
	for {
		if err := consumerGroup.Consume(context.TODO(), []string{"events"}, handler); err != nil {
			log.Fatal("Error while consuming:", err)
		}
	}

}

func (handler ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	// This method is called to process each message in the topic
	consumerIdString, _ := os.LookupEnv("CONSUMER_ID")
	var consumerId int = 0
	id, err := strconv.Atoi(consumerIdString)
	if err == nil {
		consumerId = id
	}

	fmt.Println("Consumer messages... ConsumerID: ", consumerId)

	for msg := range claim.Messages() {
		//One can do here some data transformations
		_, err := processMessage(msg, consumerId)
		if err != nil {
			sendToDeadLetterQueue(handler.producer, msg)
			continue
		}
		// Acknowledge the message to Kafka

		session.MarkMessage(msg, "")
		sendToProcessedMessagesQueue(handler.processedProducer, msg)
	}

	return nil
}

// processMessage unmarshals and processes the Kafka message
func processMessage(msg *sarama.ConsumerMessage, consumerId int) (LogEvent, error) {
	var logEvent LogEvent
	err := json.Unmarshal(msg.Value, &logEvent)
	if err != nil {
		return logEvent, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	if logEvent.ID == "30" {
		return logEvent, fmt.Errorf("oops snapped at %q", logEvent.ID)
	}
	// Simulate message processing (e.g., save to database or transform)
	log.Printf("Consumer: %d Processed LogEvent: %+v", consumerId, logEvent)
	log.Println()
	return logEvent, nil
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

// sendToDeadLetterQueue sends problematic messages to the DLQ topic
func sendToProcessedMessagesQueue(producer sarama.SyncProducer, msg *sarama.ConsumerMessage) {

	processedMessage := &sarama.ProducerMessage{
		Topic: "processed_messages",
		Key:   sarama.ByteEncoder(msg.Key),   // Preserve original key
		Value: sarama.ByteEncoder(msg.Value), // Preserve original value
	}

	partition, offset, err := producer.SendMessage(processedMessage)
	if err != nil {
		log.Printf("Failed to send message to DLQ: %v", err)
	} else {
		log.Printf("Message sent to DLQ: partition %d, offset %d", partition, offset)
	}
}
