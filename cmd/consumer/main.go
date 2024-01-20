package main

import (
	"context"
	"log"
	"time"

	"github.com/danish45007/eda-go/internal"
	"github.com/rabbitmq/amqp091-go"
	"golang.org/x/sync/errgroup"
)

func main() {
	produceConnection, err := internal.ConnectRabbitMQ("danish45007", "password", "localhost:5672", "customers", "/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/ca_certificate.pem",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/client_Danishs-MacBook-Pro.local_certificate.pem",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/client_Danishs-MacBook-Pro.local_key.pem")
	if err != nil {
		panic(err)
	}
	consumeConnection, err := internal.ConnectRabbitMQ("danish45007", "password", "localhost:5672", "customers", "/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/ca_certificate.pem",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/client_Danishs-MacBook-Pro.local_certificate.pem",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/client_Danishs-MacBook-Pro.local_key.pem")
	if err != nil {
		panic(err)
	}
	defer produceConnection.Close()
	defer consumeConnection.Close()
	produceClient, err := internal.NewRabbitMQClient(produceConnection)
	if err != nil {
		panic(err)
	}
	consumeClient, err := internal.NewRabbitMQClient(consumeConnection)
	if err != nil {
		panic(err)
	}
	queue, err := consumeClient.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}
	if err := consumeClient.CreateBinding(queue.Name, "", "customer_events"); err != nil {
		panic(err)
	}
	messageBus, err := consumeClient.Consume(queue.Name, "email_service", false)
	if err != nil {
		panic(err)
	}
	var blockingChan chan struct{}
	// set a timeout for 15 sec
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// errgroup is used to manage goroutines
	// if one of the goroutines returns an error, all goroutines are canceled
	g, _ := errgroup.WithContext(ctx)

	// Apply Qos to limit amount of messages to consume
	if err := consumeClient.ApplyQos(10, 0, true); err != nil {
		panic(err)
	}
	// 10 concurrent goroutines
	g.SetLimit(10)

	go func() {
		for message := range messageBus {
			// spawn a worker
			msg := message
			g.Go(func() error {
				log.Printf("Received message: %s", msg.Body)
				// simulate long running task
				time.Sleep(10 * time.Second)
				if err := msg.Ack(false); err != nil {
					log.Printf("Error acknowledging message : %s", err)
					return err
				}
				// use producer client to send back the callback
				if err := produceClient.Send(ctx, "customer_callbacks", msg.ReplyTo, amqp091.Publishing{
					ContentType:   "text/plain",      // The payload we send is plaintext, could be JSON or others..
					DeliveryMode:  amqp091.Transient, // This tells rabbitMQ to drop messages if restarted
					Body:          []byte("RPC Complete"),
					CorrelationId: msg.CorrelationId,
				}); err != nil {
					panic(err)
				}
				return nil
			})
		}
	}()
	log.Println("Consuming messages... Press CTRL+C to exit")
	<-blockingChan
}
