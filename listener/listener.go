package listener

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"

	"github.com/sol1du2/bouncer/mqtt"
)

type device struct {
	name       string
	expiration time.Time
	isHome     bool
}

// Listener takes care of listening for bluetooth devices and tracking them.
type Listener struct {
	config *Config

	logger logrus.FieldLogger

	btAdapter *bluetooth.Adapter
	devices   map[string]*device

	mqttConn *mqtt.Client
}

// New constructs a listener from the provided parameters.
func New(c *Config) (*Listener, error) {
	if len(c.MACAddresses) == 0 {
		return nil, fmt.Errorf("mac address list empty, nothing to track")
	}

	l := &Listener{
		config: c,
		logger: c.Logger,

		btAdapter: bluetooth.DefaultAdapter,
		devices:   make(map[string]*device),
	}

	for key, value := range c.MACAddresses {
		l.logger.Debugln("Tracking", value, "as", key)
		// Set the value as the key so that we can easily lookup by address.
		l.devices[value] = &device{
			name:       key,
			expiration: time.Now(),
			isHome:     false,
		}
	}

	l.mqttConn = mqtt.NewClient(&mqtt.Config{
		Logger: c.Logger,

		ClientID:  c.MQTTClient,
		Broker:    c.MQTTBroker,
		User:      c.MQTTUser,
		Password:  c.MQTTPassword,
		BaseTopic: c.MQTTBaseTopic,
	})

	return l, nil
}

// Listen starts all the associated resources and blocks forever until signals
// or error occurs.
func (l *Listener) Listen(ctx context.Context) error {
	listenCtx, listenCtxCancel := context.WithCancel(ctx)
	defer listenCtxCancel()

	logger := l.logger

	errCh := make(chan error, 2)
	exitCh := make(chan struct{}, 1)
	signalCh := make(chan os.Signal, 1)
	readyCh := make(chan struct{}, 1)
	triggerCh := make(chan bool, 1)

	// Start listening.
	go func() {
		select {
		case <-listenCtx.Done():
			return
		case <-readyCh:
		}

		logger.Infoln("beacon listener ready")

		if l.config.OnReady != nil {
			l.config.OnReady(l)
		}

		// Enable BLE interface.
		if err := l.btAdapter.Enable(); err != nil {
			errCh <- fmt.Errorf("failed to enable BLE stack: %s", err)
			return
		}

		// Reset expiration time.
		for _, value := range l.devices {
			value.expiration = time.Now().Add(5 * time.Minute)
		}

		// Start scanning.
		err := l.btAdapter.Scan(func(adapter *bluetooth.Adapter, btDevice bluetooth.ScanResult) {
			address := btDevice.Address.String()
			if d, ok := l.devices[address]; ok {
				logger.Debugln("found", d.name, address, btDevice.RSSI, btDevice.LocalName())
				d.expiration = time.Now().Add(5 * time.Minute)

				// If we were already home don't bother sending the message
				// again.
				if !d.isHome {
					// Connect to broker
					if err := l.mqttConn.Connect(); err != nil {
						logger.Errorln(err)
					} else {
						defer l.mqttConn.Disconnect()
						if err := l.mqttConn.PublishHome(d.name); err != nil {
							logger.Errorln(err)
						} else {
							d.isHome = true
						}
					}
				}
			}
		})

		if err != nil {
			errCh <- fmt.Errorf("failed to scan devices: %s", err)
			return
		}

		// Check every minute if any of the devices left
		for {
			for _, d := range l.devices {
				// If we were not at home don't bother sending the message
				// again.
				if time.Now().After(d.expiration) && d.isHome {
					logger.Debugln(d.name, "left")

					if err := l.mqttConn.PublishAway(d.name); err != nil {
						logger.Errorln(err)
					} else {
						d.isHome = false
					}
				}
			}

			time.Sleep(time.Minute)
		}
	}()

	// TODO(sol1du2): implement proper ready.
	go func() {
		close(readyCh)
	}()

	// Wait for exit or error, with support for HUP to reload
	err := func() error {
		signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		for {
			select {
			case errFromChannel := <-errCh:
				return errFromChannel
			case reason := <-signalCh:
				if reason == syscall.SIGHUP {
					logger.Infoln("reload signal received, scanning licenses")
					select {
					case triggerCh <- true:
					default:
					}
					continue
				}
				logger.WithField("signal", reason).Warnln("received signal")
				return nil
			}
		}
	}()

	// Shutdown, listener will stop to accept new connections.
	logger.Infoln("clean shutdown start")
	close(exitCh)

	// Cancel our own context.
	listenCtxCancel()
	func() {
		for {
			select {
			case <-exitCh:
				logger.Infoln("clean shutdown complete, exiting")
				return
			default:
				// Some services still running.
				logger.Infoln("waiting services to exit")
			}
			select {
			case reason := <-signalCh:
				logger.WithField("signal", reason).Warn("received signal")
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
	}()

	return err
}
