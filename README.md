# Bouncer
Simple golang service that monitors Bluetooth beacons in its the vicinity. Designed to be used with a Raspberry PI Zero W.

My main goal with Bouncer is to use it with [Home Assistant](https://www.home-assistant.io/) to track the tiles I have attached to each person's key. When a device is found a home message is published to an MQTT broker. If the device is not detected after a certain amount of time an away message is published.

## Dependencies
- [golang 1.17](https://golang.org/dl/)
- bluez

## Install bluez and configure its permissions

First let's install bluez:

```bash
sudo apt update
sudo apt install bluez
```

At the time of writing, by default, the pi user in raspbian does not have enough permissions to use Bouncer's scan function.

The solution is to either add the permissions directly to the user or add them to a group the user belongs too. By default there is a bluetooth group and that is what I prefer to use. First lets add the pi user to the bluetooth group:

```bash
sudo usermod -G bluetooth -a pi
```

The bluetooth group still does not have enough permissions, so let's add them. Open the following with your favorite editor:

```bash
sudo vim /etc/dbus-1/system.d/bluetooth.conf
```

And add the following to the bluetooth group:

```bash
<policy group="bluetooth">
	<allow send_destination="org.bluez"/>
    <allow send_interface="org.bluez.Agent1"/>
    <allow send_interface="org.bluez.GattCharacteristic1"/>
    <allow send_interface="org.bluez.GattDescriptor1"/>
    <allow send_interface="org.freedesktop.DBus.ObjectManager"/>
    <allow send_interface="org.freedesktop.DBus.Properties"/>
</policy>
```

## Build
To build the project simply run `./build.sh`.

## Run
To start Bouncer run the following command:

```bash
./bin/bouncerd listen [flags]
```

## Configuration

The configuration for Bouncer can be done via command line flags, environment variables or a configuration file. All file types supported by the [viper project](https://github.com/spf13/viper) are supported. E.g. `JSON`, `TOML`, `YAML`, `HCL`, `INI`.

Bouncer uses the following precedence order. Each item takes precedence over the item below it:

- flag
- env
- config
- default

The available command line flags can be looked at by running the command:

```
./bin/bouncerd listen --help
```

All flags are also available via the following envs:

| Variable                | Default            | Function |
|-------------------------|--------------------|----------|
| `BOUNCER_LOG_TIMESTAMP`   | true               | If Bouncer should log the timestamp with each log line.         |
| `BOUNCER_LOG_LEVEL`       | info               | The log level (debug, info, warn, error).         |
| `BOUNCER_SYSTEMD_NOTIFY`  | false              |          |
| `BOUNCER_CONFIG_FILE`     | ~/.bouncer.yaml    | The location and name of the config file, including its extension. This file can contain any of the env variables but its main purpose is to configure the list of MAC Addresses to track. See below for more information.        |
| `BOUNCER_MQTT_CLIENT`     | bouncer                 | The client ID the Bouncer will use when connecting to the MQTT broker.         |
| `BOUNCER_MQTT_BROKER`     | ""                 | The network address for the MQTT Broker. The port should be separated by a colon (`:`).          |
| `BOUNCER_MQTT_USER`       | ""                 | The MQTT user to authenticate with the broker.          |
| `BOUNCER_MQTT_PASSWORD`   | ""                 | The user's password for authentication.          |
| `BOUNCER_MQTT_PUBLISH_BASE_TOPIC` | bouncer/presence | The base topic when publishing the presence of a device. The device name will be appended to this topic. E.g. If a device is named `tile_solidus`, the topic will be `bouncer/presence/tile_solidus` :)          |
| `BOUNCER_MQTT_SUBSCRIBE_BASE_TOPIC` | bouncer/request | The base topic bouncer will subscribe to. This will be used for any requests the bouncer will listen to. For now only the status request (sending the status of all devices) is available. |
| `BOUNCER_DEVICE_EXPIRATION` | 60 seconds | The amount of time (in seconds) without detecting a device for it to be considered away          |

### MAC Address List

In order to run Bouncer you need to give it a list of MAC Addresses to track. An empty list will result in an error. This needs to be set in a configuration file. Currently there is no support for flags or environment variables for this. I will use `YAML` to explain this file but, as mentioned above, several types are supported.

First create a yaml file:

```bash
touch ~/.bouncer.yaml
```

If you choose a different location don't forget to change it with either the command line flag, or the environment variable.

Then, add the following with your favorite editor:

```yaml
MAC_ADDRESSES:
  desired_device_name: "MAC ADDRESS"
  desired_device_name: "MAC ADDRESS"
```

You can use any device name you want, just make sure they're unique. The MAC Addresses should also be unique, otherwise they'll be treated as one device and the last one on the list will be its name.

Lastly, if you wish you can add any of the environment variables to this file (except the file's location). Just make sure you understand the precedent rules. When adding the env variables to the config file remove the `BOUNCER_` prefix. So, for example, if you wanted to add the `BOUNCER_MQTT_BASE_TOPIC` to the config file you'd do it like this:

```yaml
MQTT_BASE_TOPIC: "not_bouncer/presence"
```

Thats it. Enjoy. :)
