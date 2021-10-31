package listener

import (
	"github.com/sirupsen/logrus"
)

// Config bundles configuration settings for Listener.
type Config struct {
	Logger logrus.FieldLogger

	MACAddresses map[string]string

	MQTTClient    string
	MQTTBroker    string
	MQTTUser      string
	MQTTPassword  string
	MQTTBaseTopic string

	OnReady func(*Listener)
}
