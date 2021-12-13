package mqtt

import (
	"github.com/sirupsen/logrus"
)

// Config bundles configuration settings for MQTT.
type Config struct {
	Logger logrus.FieldLogger

	ClientID string
	Broker   string
	User     string
	Password string

	PublishBaseTopic   string
	SubscribeBaseTopic string
}
