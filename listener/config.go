package listener

import (
	"github.com/sirupsen/logrus"
)

// Config bundles configuration settings.
type Config struct {
	Logger logrus.FieldLogger

	MACAddresses map[string]string

	OnReady func(*Listener)
}
