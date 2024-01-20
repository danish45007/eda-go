package internal

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitClient struct {
	// tcp connection used by the client
	conn *amqp.Connection
	// multiplexed connection on top of the tcp connection used for process and send messages
	ch *amqp.Channel
}

func ConnectRabbitMQ(username, password, host, vhost, caCert, clientCert, clientKey string) (*amqp.Connection, error) {
	ca, err := os.ReadFile(caCert)
	if err != nil {
		return nil, err
	}
	// Load the key pair
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}
	// Add the CA to the cert pool
	rootCAs := x509.NewCertPool()
	rootCAs.AppendCertsFromPEM(ca)

	tlsConf := &tls.Config{
		RootCAs:      rootCAs,
		Certificates: []tls.Certificate{cert},
	}
	// Setup the Connection to RabbitMQ host using AMQPs and Apply TLS config
	conn, err := amqp.DialTLS(fmt.Sprintf("amqps://%s:%s@%s/%s", username, password, host, vhost), tlsConf)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func NewRabbitMQClient(conn *amqp.Connection) (RabbitClient, error) {
	ch, err := conn.Channel()
	if err != nil {
		return RabbitClient{}, err
	}
	if err := ch.Confirm(false); err != nil {
		return RabbitClient{}, err
	}
	return RabbitClient{
		conn: conn,
		ch:   ch,
	}, nil
}

func (rc *RabbitClient) CloseChannel() error {
	return rc.ch.Close()
}

func (rc *RabbitClient) CreateQueue(queueName string, durable, autodelete bool) (amqp.Queue, error) {
	queue, err := rc.ch.QueueDeclare(queueName, durable, autodelete, false, false, nil)
	if err != nil {
		return amqp.Queue{}, err
	}
	return queue, nil
}

// CreateBinding creates a binding between a queue and an exchange using a routing key
func (rc *RabbitClient) CreateBinding(queueName, routingKey, exchangeName string) error {
	return rc.ch.QueueBind(queueName, routingKey, exchangeName, false, nil)
}

func (rc *RabbitClient) Send(ctx context.Context, exchangeName, routingKey string, options amqp.Publishing) error {
	confirmation, err := rc.ch.PublishWithDeferredConfirmWithContext(ctx, // context
		exchangeName, // exchange
		routingKey,   // routing key
		// Mandatory is used when we HAVE to have the message return an error, if there is no route or queue then
		// setting this to true will make the message bounce back
		// If this is False, and the message fails to deliver, it will be dropped
		true, // mandatory
		// immediate Removed in MQ 3 or up https://blog.rabbitmq.com/posts/2012/11/breaking-things-with-rabbitmq-3-0§
		false,   // immediate
		options, // amqp publishing struct)
	)
	if err != nil {
		return err
	}
	// wait for confirmation
	confirmation.Wait()
	return nil
}

func (rc *RabbitClient) Consume(queue, consumer string, autoAck bool) (<-chan amqp.Delivery, error) {
	return rc.ch.Consume(queue, consumer, autoAck, false, false, false, nil)
}

// ApplyQos is used to apply quality of service to the channel
// Prefetch count - How many messages the server will try to keep on the Channel
// prefetch Size - How many Bytes the server will try to keep on the channel
// global -- Any other Consumers on the connection in the future will apply the same rules if TRUE
func (rc RabbitClient) ApplyQos(count, size int, global bool) error {
	// Apply Quality of Service
	return rc.ch.Qos(
		count,
		size,
		global,
	)
}
