package main

import (
	"fmt"
	"os"

	"github.com/sol1du2/bouncer/cmd"
	"github.com/sol1du2/bouncer/cmd/bouncerd/listen"
)

func main() {
	cmd.RootCmd.Use = "bouncerd"

	cmd.RootCmd.AddCommand(cmd.CommandVersion())
	cmd.RootCmd.AddCommand(listen.CommandListen())

	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
