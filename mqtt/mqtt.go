package mqtt

import (
	"fmt"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/sirupsen/logrus"
)

const (
	waitBeforeDisconnect = 250 // milliseconds
)

type Client struct {
	config *Config
	logger logrus.FieldLogger

	client pahomqtt.Client
}

func NewClient(c *Config) *Client {
	mqttLog := c.Logger.WithField("scope", "mqtt")
	pahomqtt.DEBUG = NewLogger(mqttLog, logrus.DebugLevel)
	pahomqtt.WARN = NewLogger(mqttLog, logrus.WarnLevel)
	pahomqtt.ERROR = NewLogger(mqttLog, logrus.ErrorLevel)
	pahomqtt.CRITICAL = NewLogger(mqttLog, logrus.FatalLevel)

	return &Client{
		config: c,
		logger: c.Logger,
	}
}

func (c *Client) Connect() error {
	opts := pahomqtt.NewClientOptions().
		AddBroker(c.config.Broker).
		SetClientID(c.config.ClientID).
		SetUsername(c.config.User).
		SetPassword(c.config.Password)
	opts.SetKeepAlive(2 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	c.client = pahomqtt.NewClient(opts)
	token := c.client.Connect()
	_ = token.Wait()
	if err := token.Error(); err != nil {
		return err
	}

	return nil
}

func (c *Client) Disconnect() {
	if c.client != nil {
		c.client.Disconnect(waitBeforeDisconnect)
	}
}

func (c *Client) PublishMessage(deviceName, message string) error {
	if c.client == nil {
		return fmt.Errorf("client not initialized")
	}

	token := c.client.Publish(c.config.PublishBaseTopic+"/"+deviceName, 0, false, message)
	_ = token.Wait()

	if err := token.Error(); err != nil {
		return err
	}

	return nil
}
