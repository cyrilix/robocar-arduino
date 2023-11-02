package main

import (
	"flag"
	"github.com/cyrilix/robocar-arduino/pkg/arduino"
	"github.com/cyrilix/robocar-base/cli"
	"go.uber.org/zap"
	"log"

	"os"
)

const (
	DefaultClientId = "robocar-arduino"

	SteeringLeftPWM  = 1004
	SteeringRightPWM = 1986

	ThrottleZeroPWM = 1260
	ThrottleMinPWM  = 972.0
	ThrottleMaxPWM  = 1954.0
)

var (
	SteeringCenterPWM = (SteeringRightPWM-SteeringLeftPWM)/2 + SteeringLeftPWM
)

func main() {
	var mqttBroker, username, password, clientId string
	var throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic, throttleFeedbackTopic, maxThrottleCtrlTopic string
	var feedbackConfig string
	var device string
	var baud int
	var pubFrequency float64

	mqttQos := cli.InitIntFlag("MQTT_QOS", 0)
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	cli.InitMqttFlags(DefaultClientId, &mqttBroker, &username, &password, &clientId, &mqttQos, &mqttRetain)

	var steeringLeftPWM, steeringRightPWM, steeringCenterPWM int
	if err := cli.SetIntDefaultValueFromEnv(&steeringLeftPWM, "STEERING_LEFT_PWM", SteeringLeftPWM); err != nil {
		zap.S().Warnf("unable to init steeringLeftPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&steeringRightPWM, "STEERING_RIGHT_PWM", SteeringRightPWM); err != nil {
		zap.S().Warnf("unable to init steeringRightPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&steeringCenterPWM, "STEERING_CENTER_PWM", SteeringCenterPWM); err != nil {
		zap.S().Warnf("unable to init steeringCenterPWM arg: %v", err)
	}

	var throttleMinPWM, throttleMaxPWM, throttleZeroPWM int
	if err := cli.SetIntDefaultValueFromEnv(&throttleMinPWM, "THROTTLE_MIN_PWM", arduino.DefaultPwmThrottle.Min); err != nil {
		zap.S().Warnf("unable to init throttleMinPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&throttleMaxPWM, "THROTTLE_MAX_PWM", arduino.DefaultPwmThrottle.Max); err != nil {
		zap.S().Warnf("unable to init throttleMaxPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&throttleZeroPWM, "THROTTLE_CENTER_PWM", arduino.DefaultPwmThrottle.Middle); err != nil {
		zap.S().Warnf("unable to init throttleZeroPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&throttleMinPWM, "THROTTLE_MIN_PWM", ThrottleMinPWM); err != nil {
		zap.S().Warnf("unable to init steeringLeftPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&throttleMaxPWM, "THROTTLE_MAX_PWM", ThrottleMaxPWM); err != nil {
		zap.S().Warnf("unable to init steeringRightPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&throttleZeroPWM, "THROTTLE_ZERO_PWM", ThrottleZeroPWM); err != nil {
		zap.S().Warnf("unable to init steeringRightPWM arg: %v", err)
	}

	flag.Float64Var(&pubFrequency, "mqtt-pub-frequency", 25., "Number of messages to publish per second")
	flag.StringVar(&throttleTopic, "mqtt-topic-throttle", os.Getenv("MQTT_TOPIC_THROTTLE"), "Mqtt topic where to publish throttle values, use MQTT_TOPIC_THROTTLE if args not set")
	flag.StringVar(&steeringTopic, "mqtt-topic-steering", os.Getenv("MQTT_TOPIC_STEERING"), "Mqtt topic where to publish steering values, use MQTT_TOPIC_STEERING if args not set")
	flag.StringVar(&driveModeTopic, "mqtt-topic-drive-mode", os.Getenv("MQTT_TOPIC_DRIVE_MODE"), "Mqtt topic where to publish drive mode state, use MQTT_TOPIC_DRIVE_MODE if args not set")
	flag.StringVar(&switchRecordTopic, "mqtt-topic-switch-record", os.Getenv("MQTT_TOPIC_SWITCH_RECORD"), "Mqtt topic where to publish switch record state, use MQTT_TOPIC_SWITCH_RECORD if args not set")
	flag.StringVar(&throttleFeedbackTopic, "mqtt-topic-throttle-feedback", os.Getenv("MQTT_TOPIC_THROTTLE_FEEDBACK"), "Mqtt topic where to publish throttle feedback, use MQTT_TOPIC_THROTTLE_FEEDBACK if args not set")
	flag.StringVar(&maxThrottleCtrlTopic, "mqtt-topic-max-throttle-ctrl", os.Getenv("MQTT_TOPIC_MAX_THROTTLE_CTRL"), "Mqtt topic where to publish max throttle value allowed, use MQTT_TOPIC_MAX_THROTTLE_CTRL if args not set")
	flag.StringVar(&device, "device", "/dev/serial0", "Serial device")
	flag.IntVar(&baud, "baud", 115200, "Serial baud")
	flag.StringVar(&feedbackConfig, "throttle-feedback-config", "", "config file that described thresholds to map pwm to percent the throttle feedback")

	flag.IntVar(&steeringLeftPWM, "steering-left-pwm", steeringLeftPWM, "maxPwm left value for steering PWM, STEERING_LEFT_PWM env if args not set")
	flag.IntVar(&steeringRightPWM, "steering-right-pwm", steeringRightPWM, "maxPwm right value for steering PWM, STEERING_RIGHT_PWM env if args not set")
	flag.IntVar(&steeringCenterPWM, "steering-center-pwm", steeringCenterPWM, "middlePwm value for steering PWM, STEERING_CENTER_PWM env if args not set")

	flag.IntVar(&throttleMinPWM, "throttle-min-pwm", throttleMinPWM, "maxPwm min value for throttle PWM, THROTTLE_MIN_PWM env if args not set")
	flag.IntVar(&throttleMaxPWM, "throttle-max-pwm", throttleMaxPWM, "maxPwm max value for throttle PWM, THROTTLE_MAX_PWM env if args not set")
	flag.IntVar(&throttleZeroPWM, "throttle-center-pwm", throttleZeroPWM, "middlePwm value for throttle PWM, THROTTLE_CENTER_PWM env if args not set")

	var ctrlThrottleMinPWM, ctrlThrottleMaxPWM int
	if err := cli.SetIntDefaultValueFromEnv(&ctrlThrottleMinPWM, "CTRL_THROTTLE_MIN_PWM", arduino.DefaultPwmThrottle.Min); err != nil {
		zap.S().Warnf("unable to init ctlThrottleMinPWM arg: %v", err)
	}
	if err := cli.SetIntDefaultValueFromEnv(&ctrlThrottleMaxPWM, "CTRL_THROTTLE_MAX_PWM", arduino.DefaultPwmThrottle.Max); err != nil {
		zap.S().Warnf("unable to init ctrlThrottleMaxPWM arg: %v", err)
	}
	flag.IntVar(&ctrlThrottleMinPWM, "ctrl-throttle-min-pwm", ctrlThrottleMinPWM, "maxPwm min value for control throttle PWM, CTRL_THROTTLE_MIN_PWM env if args not set")
	flag.IntVar(&ctrlThrottleMaxPWM, "ctrl-throttle-max-pwm", ctrlThrottleMaxPWM, "maxPwm max value for control throttle PWM, CTRL_THROTTLE_MAX_PWM env if args not set")

	logLevel := zap.LevelFlag("log", zap.InfoLevel, "log level")
	flag.Parse()

	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(*logLevel)
	lgr, err := config.Build()
	if err != nil {
		log.Fatalf("unable to init logger: %v", err)
	}
	defer func() {
		if err := lgr.Sync(); err != nil {
			log.Printf("unable to Sync logger: %v\n", err)
		}
	}()
	zap.ReplaceGlobals(lgr)

	client, err := cli.Connect(mqttBroker, username, password, clientId)
	if err != nil {
		zap.S().Fatalf("unable to connect to mqtt broker: %v", err)
	}
	defer client.Disconnect(10)

	sc := arduino.NewAsymetricPWMConfig(steeringLeftPWM, steeringRightPWM, steeringCenterPWM)
	ctrlThrottlrConfig := arduino.NewPWMConfig(ctrlThrottleMinPWM, ctrlThrottleMaxPWM)
	tc := arduino.NewAsymetricPWMConfig(throttleMinPWM, throttleMaxPWM, throttleZeroPWM)

	a := arduino.NewPart(client, device, baud, throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic,
		throttleFeedbackTopic, maxThrottleCtrlTopic,
		pubFrequency,
		arduino.WithThrottleFeedbackConfig(feedbackConfig),
		arduino.WithThrottleConfig(tc),
		arduino.WithSteeringConfig(sc),
		arduino.WithMaxThrottleCtrl(ctrlThrottlrConfig),
	)

	cli.HandleExit(a)

	err = a.Start()
	if err != nil {
		zap.S().Errorw("unable to start service", "error", err)
	}
}
