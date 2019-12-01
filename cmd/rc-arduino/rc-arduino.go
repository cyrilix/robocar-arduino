package main

import (
	"flag"
	"github.com/cyrilix/robocar-arduino/arduino"
	"github.com/cyrilix/robocar-base/mqttdevice"
	"log"
	"os"
	"strconv"
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
	var mqttBroker, username, password, qos, clientId, topicBase string
	var device string
	var baud int
	var pubFrequency float64

	setDefaultValueFromEnv(&clientId, "MQTT_CLIENT_ID", DefaultClientId)
	setDefaultValueFromEnv(&mqttBroker, "MQTT_BROKER", "tcp://127.0.0.1:1883")
	setDefaultValueFromEnv(&qos, "MQTT_QOS", "0")
	mqttQos, err := strconv.Atoi(qos)
	if err != nil {
		log.Panicf("invalid mqtt qos value: %v", qos)
	}
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	flag.StringVar(&mqttBroker, "mqtt-broker", mqttBroker, "Broker Uri, use MQTT_BROKER env if arg not set")
	flag.StringVar(&username, "mqtt-username", os.Getenv("MQTT_USERNAME"), "Broker Username, use MQTT_USERNAME env if arg not set")
	flag.StringVar(&password, "mqtt-password", os.Getenv("MQTT_PASSWORD"), "Broker Password, MQTT_PASSWORD env if args not set")
	flag.StringVar(&clientId, "mqtt-client-id", clientId, "Mqtt client id, use MQTT_CLIENT_ID env if args not set")
	flag.IntVar(&mqttQos, "mqtt-qos", mqttQos, "Qos to pusblish message, use MQTT_QOS env if arg not set")
	flag.StringVar(&topicBase, "mqtt-topic-base", os.Getenv("MQTT_TOPIC_BASE"), "Mqtt topic prefix, use MQTT_TOPIC_BASE if args not set")
	flag.BoolVar(&mqttRetain, "mqtt-retain", mqttRetain, "Retain mqtt message, if not set, true if MQTT_RETAIN env variable is set")
	flag.Float64Var(&pubFrequency, "mqtt-pub-frequency", 25., "Number of messages to publish per second")
	flag.StringVar(&device, "device", "/dev/serial0", "Serial device")
	flag.IntVar(&baud, "baud", 115200, "Serial baud")

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	pubSub := mqttdevice.NewPahoMqttPubSub(mqttBroker, username, password, clientId, mqttQos, mqttRetain)
	defer pubSub.Close()

	a := arduino.NewArduinoPart(device, baud, pubSub,topicBase, pubFrequency)
	a.Start()
}
