# Bouncer
A golang service that monitors Bluetooth beacons in its the vicinity and reports their presence via MQTT messages. Designed to be used with a Raspberry PI Zero W.

My goal for Bouncer is to use it with [Home Assistant](https://www.home-assistant.io/) to track the tiles I have attached to each person's key. When a device is found a home message is published to an MQTT broker. If the device is not detected after a certain amount of time an away message is published.

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

## MQTT message layout

At the moment all MQTT messages are sent as a simple string without any json payload.

Bouncer supports 2 base topics, which are configurable.
- **The publish base topic** is used to publish a message indicating a change in the device's presence. This message is a string which can be either `home`, if the device was detected, or `not_home` if the device was not detected after the set timeout expired. Please note that this message is only published if the device presence status **changes**. Meaning, if the device was already home (or already away), no new messages are published. This helps avoid spamming the MQTT broker. For the time being these strings can not be configured. The topic format is `PUBLISH_BASE_TOPIC/DEVICE_NAME`. For example, using the default base topic and the device name `tile_alice`, when the device is detected at home a message `home` will be sent to `bouncer/presence/tile_alice`

- **The subscribe base topic** is used to listen to requests. For the time being the only available request is to ask for the current presence status of all devices. The topic format is `SUBSCRIBE_BASE_TOPIC/presence`. When a message is published to this topic (no payload necessary), Bouncer will send a message to the `PUBLISH_BASE_TOPIC/DEVICE` for every device it's tracking.

## Home Assistant Configuration

As mentioned above this service was designed to be used with [Home Assistant](https://www.home-assistant.io/). The messages were chosen to be as compatible with it as possible so we get minimal configuration. For more information on how Home Assistant works please consult their docs. Here I will only explain the necessary configuration to listen to the messages from Bouncer.

Essentially all you need to do is add an MQTT device tracker to your configuration file. Here is an example:

```yaml
device_tracker:
  - platform: mqtt
    devices:
      tile_device_1: "bouncer/presence/device_1"
   	  tile_device_2: "bouncer/presence/device_2"
      tile_device_3: "bouncer/presence/device_3"
      tile_device_4: "bouncer/presence/device_4"
    qos: 0
    payload_home: "home"
    payload_not_home: "not_home"
```
This assumes the default `PUBLISH_BASE_TOPIC`. Consult the Bouncer configuration for more details.

Now all you have to do is associate these device trackers with specific persons on your system. For example, via the configuration yaml file:

```yaml
person:
  - name: Person1
    device_trackers:
      - device_tracker.device_1
  - name: Person2
    device_trackers:
      - device_tracker.device_2
  - name: Person3
    device_trackers:
      - device_tracker.device_3
  - name: Person4
    device_trackers:
      - device_tracker.device_4
```

Now Home Assistant will set the presence status of each person whenever an MQTT message is received.

This is enough to track the presence of the devices but if, for example, you need to restart Home Assistant, once it's back up again, it will not know what the current status of the person is and, by default, it will set them to `away`. Since Bouncer will not publish anything if the status does not change, we need to publish a message from Home Assistant to request the current status of all devices. This can be done via the subscribe topic mentioned above. We could, for example, create an automation that sends a message to this topic once Home Assistant is up and running:

```yaml
- id: on_start_request_persons_presence
  trigger:
    - platform: homeassistant
      event: start
  action:
    - service: mqtt.publish
      data:
        topic: bouncer/request/presence
```
This assumes the default `SUBSCRIBE_BASE_TOPIC`. Consult the Bouncer configuration for more details.

## Bouncer Configuration

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
| `BOUNCER_MQTT_CLIENT`     | bouncer                 | The client ID that Bouncer will use when connecting to the MQTT broker.         |
| `BOUNCER_MQTT_BROKER`     | ""                 | The network address for the MQTT Broker. The port should be separated by a colon (`:`).          |
| `BOUNCER_MQTT_USER`       | ""                 | The MQTT user to authenticate with the broker.          |
| `BOUNCER_MQTT_PASSWORD`   | ""                 | The user's password for authentication.          |
| `BOUNCER_MQTT_PUBLISH_BASE_TOPIC` | bouncer/presence | The base topic when publishing the presence of a device. The device name will be appended to this topic. E.g. If a device is named `tile_solidus`, the topic will be `bouncer/presence/tile_solidus` :)          |
| `BOUNCER_MQTT_SUBSCRIBE_BASE_TOPIC` | bouncer/request | The base topic bouncer will subscribe to. This will be used for any requests Bouncer will listen to. For now only the status request (sending the status of all devices) is available. |
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
  desired_device_name_1: "MAC ADDRESS 1"
  desired_device_name_2: "MAC ADDRESS 2"
  ...
```

You can use any device name you want, just make sure they're unique. The MAC Addresses should also be unique, otherwise they'll be treated as one device and the last one on the list will be its name.

Lastly, if you wish you can add any of the environment variables to this file (except the file's location). Just make sure you understand the precedent rules. When adding the env variables to the config file remove the `BOUNCER_` prefix. So, for example, if you wanted to add the `BOUNCER_MQTT_BASE_TOPIC` to the config file you'd do it like this:

```yaml
MQTT_BASE_TOPIC: "yaml_bouncer/presence"
```

Thats it. Enjoy. :)
