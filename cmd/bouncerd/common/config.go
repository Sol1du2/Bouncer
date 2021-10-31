package common

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Generic settings.
	LogTimestamp  bool
	LogLevel      string
	SystemdNotify bool

	// Mqtt settings.
	MQTTBroker    string
	MQTTUser      string
	MQTTPassword  string
	MQTTBaseTopic string

	MACAddresses = map[string]string{}
)

func SetDefaults(cmd *cobra.Command) {
	// Defaults
	viper.SetDefault("LOG_TIMESTAMP", true)
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("SYSTEMD_NOTIFY", false)
	viper.SetDefault("CONFIG_FILE", "~/.bouncer.yaml")

	viper.SetDefault("MQTT_BROKER", "")
	viper.SetDefault("MQTT_USER", "")
	viper.SetDefault("MQTT_PASSWORD", "")
	viper.SetDefault("MQTT_BASE_TOPIC", "bouncer/presence")

	viper.SetDefault("MAC_ADDRESSES", map[string]string{})

	// Command line flags
	cmd.Flags().Bool("log-timestamp", true, "Prefix each log line with timestamp")
	cmd.Flags().String("log-level", "info", "Log level (one of panic, fatal, error, warn, info or debug)")
	cmd.Flags().Bool("systemd-notify", false, "Enable systemd sd_notify callback")
	cmd.Flags().String(
		"config-file", "~/.bouncer.yaml", "Configuration file location. This is where a list of mac addresses should be included")

	cmd.Flags().String("mqtt-broker", "", "MQTT Broker to connect to")
	cmd.Flags().String("mqtt-user", "", "MQTT Username")
	cmd.Flags().String("mqtt-password", "", "MQTT Password")
	cmd.Flags().String("mqtt-base-topic", "bouncer/presence", "MQTT Base topic. Device name in mac address list will be appended to the topic")

	_ = viper.BindPFlag("LOG_TIMESTAMP", cmd.Flags().Lookup("log-timestamp"))
	_ = viper.BindPFlag("LOG_LEVEL", cmd.Flags().Lookup("log-level"))
	_ = viper.BindPFlag("SYSTEMD_NOTIFY", cmd.Flags().Lookup("systemd-notify"))
	_ = viper.BindPFlag("CONFIG_FILE", cmd.Flags().Lookup("config-file"))

	_ = viper.BindPFlag("MQTT_BROKER", cmd.Flags().Lookup("mqtt-broker"))
	_ = viper.BindPFlag("MQTT_USER", cmd.Flags().Lookup("mqtt-user"))
	_ = viper.BindPFlag("MQTT_PASSWORD", cmd.Flags().Lookup("mqtt-password"))
	_ = viper.BindPFlag("MQTT_BASE_TOPIC", cmd.Flags().Lookup("mqtt-base-topic"))

	// Setup env
	viper.SetEnvPrefix("bouncer")
	viper.AutomaticEnv()
}

func ApplyConfiguration(cmd *cobra.Command) error {
	envFile := viper.GetString("CONFIG_FILE")
	viper.SetConfigFile(envFile)
	if err := viper.ReadInConfig(); err != nil {
		// If file does not exist continue with only env variables and flags.
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	LogTimestamp = viper.GetBool("LOG_TIMESTAMP")
	LogLevel = viper.GetString("LOG_LEVEL")
	SystemdNotify = viper.GetBool("SYSTEMD_NOTIFY")

	MQTTBroker = viper.GetString("MQTT_BROKER")
	MQTTUser = viper.GetString("MQTT_USER")
	MQTTPassword = viper.GetString("MQTT_PASSWORD")
	MQTTBaseTopic = viper.GetString("MQTT_BASE_TOPIC")

	MACAddresses = viper.GetStringMapString("MAC_ADDRESSES")

	return nil
}
