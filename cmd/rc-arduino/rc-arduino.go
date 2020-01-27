package main

import (
	"flag"
	"github.com/cyrilix/robocar-arduino/arduino"
	"github.com/cyrilix/robocar-base/cli"
	"log"
	"os"
)

const DefaultClientId = "robocar-arduino"

func main() {
	var mqttBroker, username, password, clientId string
	var throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic string
	var device string
	var baud int
	var pubFrequency float64
	var debug bool

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	flag.Float64Var(&pubFrequency, "mqtt-pub-frequency", 25., "Number of messages to publish per second")
	flag.StringVar(&throttleTopic, "mqtt-topic-throttle", os.Getenv("MQTT_TOPIC_THROTTLE"), "Mqtt topic where to publish throttle values, use MQTT_TOPIC_THROTTLE if args not set")
	flag.StringVar(&steeringTopic, "mqtt-topic-steering", os.Getenv("MQTT_TOPIC_STEERING"), "Mqtt topic where to publish steering values, use MQTT_TOPIC_STEERING if args not set")
	flag.StringVar(&driveModeTopic, "mqtt-topic-drive-mode", os.Getenv("MQTT_TOPIC_DRIVE_MODE"), "Mqtt topic where to publish drive mode state, use MQTT_TOPIC_DRIVE_MODE if args not set")
	flag.StringVar(&switchRecordTopic, "mqtt-topic-switch-record", os.Getenv("MQTT_TOPIC_SWITCH_RECORD"), "Mqtt topic where to publish switch record state, use MQTT_TOPIC_SWITCH_RECORD if args not set")
	flag.StringVar(&device, "device", "/dev/serial0", "Serial device")
	flag.IntVar(&baud, "baud", 115200, "Serial baud")
	flag.BoolVar(&debug, "debug", false, "Display raw value to debug")

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		log.Fatalf("unable to connect to mqtt broker: %v", err)
	}
	defer client.Disconnect(10)

	a := arduino.NewPart(client, device, baud, throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic, pubFrequency, debug)
	err = a.Start()
	if err != nil {
		log.Printf("unable to start service: %v", err)
	}
}
