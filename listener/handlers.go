package listener

import (
	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

// handlePresenceRequest publishes the presence status of all devices being tracked.
func (l *Listener) handlePresenceRequest(_ pahomqtt.Client, _ pahomqtt.Message) {
	l.logger.Debugln("received presence request")

	l.dMutex.RLock()
	defer l.dMutex.RUnlock()
	for _, d := range l.devices {
		var msg string
		if d.isHome {
			msg = home
		} else {
			msg = away
		}

		// The handling of the messages should not block, publishing messages
		// should be done in a separate thread in case they take too long to
		// send.
		go func(deviceName string) {
			if err := l.mqttConn.PublishMessage(deviceName, msg); err != nil {
				l.logger.WithError(err).Errorln("failed to send", msg, "message")
			}
		}(d.name)
	}
}
