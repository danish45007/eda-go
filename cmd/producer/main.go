package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"github.com/danish45007/eda-go/internal"
	"github.com/rabbitmq/amqp091-go"
)

func main() {
	produceConnection, err := internal.ConnectRabbitMQ("danish45007", "password", "localhost:5672", "customers",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/ca_certificate.pem",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/client_Danishs-MacBook-Pro.local_certificate.pem",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/client_Danishs-MacBook-Pro.local_key.pem",
	)
	if err != nil {
		panic(err)
	}
	consumeConnection, err := internal.ConnectRabbitMQ("danish45007", "password", "localhost:5672", "customers",
		"/Users/danishsharma/go/src/eda-go/tls-gen/basic/result/ca_certificate.pem",
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
	// unnamed callback queue
	queue, err := consumeClient.CreateQueue("", true, true)
	if err != nil {
		panic(err)
	}
	if err := consumeClient.CreateBinding(queue.Name, queue.Name, "customer_callbacks"); err != nil {
		panic(err)
	}
	callbackMessageBus, err := consumeClient.Consume(queue.Name, "client_api", true)
	if err != nil {
		panic(err)
	}
	var blockingChan chan struct{}
	go func() {
		for message := range callbackMessageBus {
			log.Printf("Callback message received: %s", message.CorrelationId)
		}
	}()
	// if err := client.CreateQueue("customers_created", true, false); err != nil {
	// 	panic(err)
	// }
	// if err := client.CreateQueue("customers_test", false, false); err != nil {
	// 	panic(err)
	// }
	// if err := client.CreateBinding("customers_created", "customers.created.*", "customer_events"); err != nil {
	// 	panic(err)
	// }
	// if err := client.CreateBinding("customers_test", "customers.*", "customer_events"); err != nil {
	// 	panic(err)
	// }
	// Create context to manage timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// send 10 messages
	for i := 0; i < 11; i++ {
		if err := produceClient.Send(ctx, "customer_events", "customers.created.us", amqp091.Publishing{
			ContentType:   "text/plain",
			DeliveryMode:  amqp091.Transient, // This tells rabbitMQ that this message can be deleted if no resources accepts it before a restart (non durable)
			ReplyTo:       queue.Name,
			CorrelationId: strconv.Itoa(i),
			Body:          []byte("Test Message 1"),
			MessageId:     strconv.Itoa(i),
		}); err != nil {
			panic(err)
		}
	}
	<-blockingChan
}
