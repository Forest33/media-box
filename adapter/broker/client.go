package broker

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"

	"media-box-ng/pkg/logger"
)

type Config struct {
	Host       string
	Port       int
	StateTopic string
	ClientID   string
	UserName   string
	Password   string
}

type Client struct {
	cfg                       *Config
	log                       *logger.Zerolog
	cli                       mqtt.Client
	subscribers               map[string]MessageHandler
	externalConnectHandler    ConnectHandler
	externalDisconnectHandler DisconnectHandler
}

type MessageHandler func(topic string, payload []byte)
type ConnectHandler func()
type DisconnectHandler func()

func NewBrokerClient(cfg *Config, log *logger.Zerolog) (*Client, error) {
	b := &Client{
		cfg:         cfg,
		log:         log,
		subscribers: make(map[string]MessageHandler, 1),
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", cfg.Host, cfg.Port))
	opts.SetClientID(fmt.Sprintf("%s-%d", cfg.ClientID, time.Now().Unix()))
	opts.SetUsername(cfg.UserName)
	opts.SetPassword(cfg.Password)
	opts.SetAutoReconnect(true)
	opts.SetDefaultPublishHandler(b.messagePubHandler)
	opts.OnConnect = b.connectHandler
	opts.OnConnectionLost = b.connectLostHandler
	b.cli = mqtt.NewClient(opts)

	return b, nil
}

func (b *Client) Start() error {
	if token := b.cli.Connect(); token.Wait() && token.Error() != nil {
		return token.Error()
	}
	return nil
}

func (b *Client) PublishState(data []byte) {
	b.cli.Publish(b.cfg.StateTopic, 0, false, data).Wait()
}

func (b *Client) Subscribe(topic string, handler MessageHandler) {
	b.subscribers[topic] = handler
	b.cli.Subscribe(topic, 1, b.messageHandler).Wait()
}

func (b *Client) Close() {
	b.cli.Disconnect(1000)
}

func (b *Client) SetConnectHandler(h ConnectHandler) {
	b.externalConnectHandler = h
}

func (b *Client) SetDisconnectHandler(h DisconnectHandler) {
	b.externalDisconnectHandler = h
}

func (b *Client) messageHandler(client mqtt.Client, msg mqtt.Message) {
	if handler, ok := b.subscribers[msg.Topic()]; ok {
		handler(msg.Topic(), msg.Payload())
	}
}

func (b *Client) messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	b.log.Debug().Msgf("MQTT received message: %s from topic: %s\n", msg.Payload(), msg.Topic())
}

func (b *Client) connectHandler(client mqtt.Client) {
	b.log.Info().Msgf("MQTT connected")
	if b.externalConnectHandler != nil {
		b.externalConnectHandler()
	}
}

func (b *Client) connectLostHandler(client mqtt.Client, err error) {
	b.log.Error().Msgf("MQTT connect lost: %v", err)
}
