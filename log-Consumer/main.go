package main

import (
	"fmt"
	"log"

	"github.com/IBM/sarama"
)

func main() {

	brokers := []string{"kafka:9092"}
	//Set up our kafka
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	//Create a new consumer
	consumer, err := sarama.NewConsumer(brokers, config)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	//Start consuming from the logs topic
	partitionConsumer, err := consumer.ConsumePartition("logs", 0, sarama.OffsetOldest)
	if err != nil {
		log.Fatal(err)
	}

	defer partitionConsumer.Close()

	//Print consumed messages
	for msg := range partitionConsumer.Messages() {
		fmt.Printf("Consumed Log: %s\n", string(msg.Value))
	}
}
