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

	presenceTopic = "presence"
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

		if err := l.scanBeacons(); err != nil {
			errCh <- err
		}
	}()

	// Check if devices are no longer valid.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-listenCtx.Done():
			return
		case <-readyCh:
		}

		l.checkBeaconPresence(listenCtx)
	}()

	// Setup and then wait until services close before closing the exit channel.
	go func() {
		if err := l.setup(); err != nil {
			errCh <- err
		} else {
			close(readyCh)
		}

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
	l.mqttConn.Disconnect()

	// Cancel our own context and stop context sensitive services.
	listenCtxCancel()

	// Wait for the exitCh to be closed indicating all services have exited.
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

// setup Initializes the BT adapter as well as the MQTT client and the devices
// map.
// If set, calls the OnReady function when done.
func (l *Listener) setup() error {
	c := l.config

	// Initiate devices map.
	for key, value := range c.MACAddresses {
		l.logger.Debugln("Tracking", value, "as", key)
		// Set the value as the key so that we can easily lookup by address.
		l.devices[value] = &device{
			name:       key,
			expiration: time.Now(),
			isHome:     false,
		}
	}

	// Initiate MQTT client.
	l.mqttConn = mqtt.NewClient(&mqtt.Config{
		Logger: c.Logger,

		ClientID:           c.MQTTClient,
		Broker:             c.MQTTBroker,
		User:               c.MQTTUser,
		Password:           c.MQTTPassword,
		PublishBaseTopic:   c.MQTTPublishBaseTopic,
		SubscribeBaseTopic: c.MQTTSubscribeBaseTopic,
	})

	// Enable BLE interface.
	if err := l.btAdapter.Enable(); err != nil {
		return fmt.Errorf("failed to enable BLE stack: %s", err)
	}

	// Connect to MQTT server
	if err := l.mqttConn.Connect(); err != nil {
		return fmt.Errorf("failed to connect to MQTT server: %s", err)
	}

	// Add new subscriptions here, as needed.
	if err := l.mqttConn.Subscribe(presenceTopic, l.handleStatusRequest); err != nil {
		return fmt.Errorf("failed to subscribe to %s topic: %s", presenceTopic, err)
	}

	// Notify ready.
	if l.config.OnReady != nil {
		l.config.OnReady(l)
	}

	return nil
}

// scanBeacons Starts scanning for the beacons in the devices map.
// If a device that was previously marked as away is found, an MQTT home message
// is sent. If the device was already marked as home, no new messages are sent.
//
// Blocks forever until the bt adaptor's StopScan function is called.
func (l *Listener) scanBeacons() error {
	logger := l.logger

	logger.Infoln("beacon scan started")
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
				if err := l.mqttConn.PublishMessage(d.name, home); err != nil {
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
		return fmt.Errorf("failed to scan devices: %s", err)
	}

	logger.Infoln("beacon scan stopped")
	return nil
}

// checkBeaconPresence checks if any of the beacon devices has been away for
// longer than the set expiration time. The check, for all devices, is performed
// every 5 seconds.
// If the device was previously marked as being at home an MQTT away message is
// sent. If the device was already marked as away, no new messages are sent.
//
// Blocks forever until the ctx is canceled.
func (l *Listener) checkBeaconPresence(ctx context.Context) {
	logger := l.logger

	logger.Infoln("beacon presence check start")
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
			if err := l.mqttConn.PublishMessage(d.name, away); err != nil {
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
		case <-ctx.Done():
			logger.Infoln("beacon presence check stopped")
			return
		case <-time.After(5 * time.Second):
		}
	}
}
