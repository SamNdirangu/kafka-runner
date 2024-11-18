package main

import (
	"fmt"
	"log"

	"github.com/IBM/sarama"
)

func main() {
	//Set up our kafka
	config := sarama.NewConfig()
	config.Consumer.Return.Errors = true

	//Create a new consumer
	consumer, err := sarama.NewConsumer([]string{"localhost:9093"}, config)
	if err != nil {
		log.Fatal(err)
	}
	defer consumer.Close()

	//Start consuming from the logs topic
	partitionConsumer, err := consumer.ConsumePartition("logs", 0, sarama.OffsetNewest)
	if err != nil {
		log.Fatal(err)
	}

	defer partitionConsumer.Close()

	//Print consumed messages
	for msg := range partitionConsumer.Messages() {
		fmt.Printf("Consumed Log: %s\n", string(msg.Value))
	}
}
