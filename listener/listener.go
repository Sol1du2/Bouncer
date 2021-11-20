package listener

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"tinygo.org/x/bluetooth"

	"github.com/sol1du2/bouncer/mqtt"
)

// TODO(sol1du2): make this configurable.
const (
	home = "home"
	away = "not_home"
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
	dMutex    sync.RWMutex

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

	var wg sync.WaitGroup

	// Start listening for BT devices.
	wg.Add(1)
	go func() {
		defer wg.Done()

		select {
		case <-listenCtx.Done():
			return
		case <-readyCh:
		}

		if l.config.OnReady != nil {
			l.config.OnReady(l)
		}

		// Enable BLE interface.
		if err := l.btAdapter.Enable(); err != nil {
			errCh <- fmt.Errorf("failed to enable BLE stack: %s", err)
			return
		}

		logger.Infoln("beacon listener started")
		// Start scanning, this blocks.
		err := l.btAdapter.Scan(func(adapter *bluetooth.Adapter, btDevice bluetooth.ScanResult) {
			address := btDevice.Address.String()
			l.dMutex.RLock()
			if d, ok := l.devices[address]; ok {
				// We found the device and need to edit it, acquire a write
				// lock.
				l.dMutex.RUnlock()
				l.dMutex.Lock()
				defer l.dMutex.Unlock()

				logger.Debugln("found", d.name, address, btDevice.RSSI, btDevice.LocalName())
				d.expiration = time.Now().Add(l.config.DeviceExpiration)

				// If we were already home don't bother sending the message
				// again.
				if !d.isHome {
					logger.Debugln(d.name, "has arrived")
					if err := l.connectAndPublish(d.name, home); err != nil {
						logger.WithError(err).Errorln("failed to send", home, "message")
					} else {
						d.isHome = true
					}
				}
			} else {
				// Only read unlock if the condition was NOT true.
				l.dMutex.RUnlock()
			}
		})

		if err != nil {
			errCh <- fmt.Errorf("failed to scan devices: %s", err)
			return
		}
	}()

	// Check if devices are no longer valid
	wg.Add(1)
	go func() {
		defer wg.Done()

		logger.Infoln("beacon presence check started")
		for {
			l.dMutex.RLock()
			for _, d := range l.devices {
				// If the time hasn't expired yet, or we are already away,
				// don't bother sending messages.
				if !time.Now().After(d.expiration) || !d.isHome {
					continue
				}

				// Time has expired and the device is still set as home,
				// swap lock for a write lock.
				l.dMutex.RUnlock()
				l.dMutex.Lock()

				logger.Debugln(d.name, "left")
				if err := l.connectAndPublish(d.name, away); err != nil {
					logger.WithError(err).Errorln("failed to send", home, "message")
				} else {
					d.isHome = false
				}

				// Don't forget to swap the lock to a read lock again.
				l.dMutex.Unlock()
				l.dMutex.RLock()
			}

			l.dMutex.RUnlock()

			select {
			case <-listenCtx.Done():
				logger.Infoln("beacon presence check stopped")
				return
			case <-time.After(5 * time.Second):
			}
		}
	}()

	// TODO(sol1du2): implement proper ready.
	go func() {
		close(readyCh)
	}()

	// Wait for all services to stop before closing the exit channel
	go func() {
		wg.Wait()
		close(exitCh)
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

	logger.Infoln("clean shutdown start")

	// Stop Scanner.
	if stopErr := l.btAdapter.StopScan(); stopErr != nil {
		logger.Debugln(stopErr)
	}
	logger.Infoln("beacon listener stopped")

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

// connectAndPublish connects to the MQTT broker, publishes a message and
// disconnects.
func (l *Listener) connectAndPublish(deviceName, message string) error {
	if err := l.mqttConn.Connect(); err != nil {
		return err
	}
	defer l.mqttConn.Disconnect()
	return l.mqttConn.PublishMessage(deviceName, message)
}
