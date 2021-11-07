package listen

import (
	"context"
	"fmt"
	"os"

	systemDaemon "github.com/coreos/go-systemd/v22/daemon"
	"github.com/spf13/cobra"

	"github.com/sol1du2/bouncer/cmd/bouncerd/common"
	"github.com/sol1du2/bouncer/listener"
)

func CommandListen() *cobra.Command {
	listenCmd := &cobra.Command{
		Use:   "listen",
		Short: "Start service",
		Run: func(cmd *cobra.Command, args []string) {
			// Add listener
			if err := listen(cmd); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	common.SetDefaults(listenCmd)

	return listenCmd
}

func listen(cmd *cobra.Command) error {
	ctx := context.Background()

	if err := common.ApplyConfiguration(cmd); err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	logger, err := newLogger(!common.LogTimestamp, common.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	logger.Debugln("bouncer listening start")

	cfg := &listener.Config{
		Logger: logger,

		OnReady: func(listener *listener.Listener) {
			if common.SystemdNotify {
				ok, notifyErr := systemDaemon.SdNotify(false, systemDaemon.SdNotifyReady)
				logger.WithField("ok", ok).Debugln("called systemd sd_notify ready")
				if notifyErr != nil {
					logger.WithError(notifyErr).Errorln("failed to trigger systemd sd_notify")
				}
			}
		},

		MACAddresses: common.MACAddresses,

		MQTTClient:    common.MQTTClient,
		MQTTBroker:    common.MQTTBroker,
		MQTTUser:      common.MQTTUser,
		MQTTPassword:  common.MQTTPassword,
		MQTTBaseTopic: common.MQTTBaseTopic,

		DeviceExpiration: common.DeviceExpiration,
	}

	l, err := listener.New(cfg)
	if err != nil {
		return err
	}

	return l.Listen(ctx)
}
