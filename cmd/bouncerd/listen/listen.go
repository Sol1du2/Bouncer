package listen

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sol1du2/bouncer/cmd/bouncerd/common"
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
	if err := common.ApplyConfiguration(cmd); err != nil {
		return fmt.Errorf("failed to apply configuration: %w", err)
	}

	logger, err := newLogger(!common.LogTimestamp, common.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}

	logger.Debugln("bouncer listening start")

	return fmt.Errorf("not implemented")
}
