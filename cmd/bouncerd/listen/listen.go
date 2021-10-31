package listen

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var defaultSystemdNotify = false

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

	listenCmd.Flags().Bool("log-timestamp", true, "Prefix each log line with timestamp")
	listenCmd.Flags().String("log-level", "info", "Log level (one of panic, fatal, error, warn, info or debug)")
	listenCmd.Flags().BoolVar(&defaultSystemdNotify, "systemd-notify", defaultSystemdNotify, "Enable systemd sd_notify callback")

	return listenCmd
}

func listen(cmd *cobra.Command) error {
	logTimestamp, _ := cmd.Flags().GetBool("log-timestamp")
	logLevel, _ := cmd.Flags().GetString("log-level")

	logger, err := newLogger(!logTimestamp, logLevel)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	logger.Debugln("bouncer listening start")

	return fmt.Errorf("not implemented")
}
