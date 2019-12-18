package main

import (
	"flag"
	"github.com/cyrilix/robocar-arduino/arduino"
	"github.com/cyrilix/robocar-base/cli"
	"github.com/cyrilix/robocar-base/mqttdevice"
	"log"
	"os"
)

const DefaultClientId = "robocar-arduino"

func setDefaultValueFromEnv(value *string, key string, defaultValue string) {
	if os.Getenv(key) != "" {
		*value = os.Getenv(key)
	} else {
		*value = defaultValue
	}
}

func main() {
	var mqttBroker, username, password, clientId, topicBase string
	var device string
	var baud int
	var pubFrequency float64
	var debug bool

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	flag.Float64Var(&pubFrequency, "mqtt-pub-frequency", 25., "Number of messages to publish per second")
	flag.StringVar(&topicBase, "mqtt-topic-base", os.Getenv("MQTT_TOPIC_BASE"), "Mqtt topic prefix, use MQTT_TOPIC_BASE if args not set")
	flag.StringVar(&device, "device", "/dev/serial0", "Serial device")
	flag.IntVar(&baud, "baud", 115200, "Serial baud")
	flag.BoolVar(&debug, "debug", false, "Display raw value to debug")

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	pubSub := mqttdevice.NewPahoMqttPubSub(mqttBroker, username, password, clientId, mqttQos, mqttRetain)
	defer func() {
		err := pubSub.Close()
		if err != nil {
			log.Printf("unable to close pubsub: %v", err)
		}
	}()

	a := arduino.NewPart(device, baud, pubSub, topicBase, pubFrequency, debug)
	err := a.Start()
	if err != nil {
		log.Printf("unable to start service: %v", err)
	}
}
