package arduino

import (
	"bufio"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/tarm/serial"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"io"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MinPwmAngle = 999.0
	MaxPwmAngle = 1985.0

	MinPwmThrottle = 972.0
	MaxPwmThrottle = 1954.0
)

var (
	serialLineRegex = regexp.MustCompile(`(?P<timestamp>\d+),(?P<channel_1>\d+),(?P<channel_2>\d+),(?P<channel_3>\d+),(?P<channel_4>\d+),(?P<channel_5>\d+),(?P<channel_6>\d+),(?P<frequency>\d+),(?P<distance_cm>\d+)?`)
)

type Part struct {
	client                                                          mqtt.Client
	throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic string
	pubFrequency                                                    float64
	serial                                                          io.Reader
	mutex                                                           sync.Mutex
	steering                                                        float32
	throttle                                                        float32
	ctrlRecord                                                      bool
	driveMode                                                       events.DriveMode
	cancel                                                          chan interface{}
}

func NewPart(client mqtt.Client, name string, baud int, throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic string, pubFrequency float64) *Part {
	c := &serial.Config{Name: name, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		zap.S().Panicw("unable to open serial port: %v", err)
	}
	return &Part{
		client:            client,
		serial:            s,
		throttleTopic:     throttleTopic,
		steeringTopic:     steeringTopic,
		driveModeTopic:    driveModeTopic,
		switchRecordTopic: switchRecordTopic,
		pubFrequency:      pubFrequency,
		driveMode:         events.DriveMode_INVALID,
		cancel:            make(chan interface{}),
	}
}

func (a *Part) Start() error {
	zap.S().Info("start arduino part")
	go a.publishLoop()
	for {
		buff := bufio.NewReader(a.serial)
		line, err := buff.ReadString('\n')
		if err == io.EOF || line == "" {
			zap.S().Error("remote connection closed")
			break
		}

		if !serialLineRegex.MatchString(line) {
			zap.S().Errorf("invalid line: '%v'", line)
			continue
		}
		values := strings.Split(strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r"), ",")

		a.updateValues(values)
	}
	return nil
}

func (a *Part) updateValues(values []string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.processChannel1(values[1])
	a.processChannel2(values[2])
	a.processChannel3(values[3])
	a.processChannel4(values[4])
	a.processChannel5(values[5])
	a.processChannel6(values[6])
}

func (a *Part) Stop() {
	zap.S().Info("stop ArduinoPart")
	close(a.cancel)
	switch s := a.serial.(type) {
	case io.ReadCloser:
		if err := s.Close(); err != nil {
			zap.S().Fatalf("unable to close serial port: %v", err)
		}
	}
}

func (a *Part) processChannel1(v string) {
	zap.L().Debug("process new value for channel1", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid value for channel1, should be an int: %v", err)
	}
	if value < MinPwmAngle {
		value = MinPwmAngle
	} else if value > MaxPwmAngle {
		value = MaxPwmAngle
	}
	a.steering = ((float32(value)-MinPwmAngle)/(MaxPwmAngle-MinPwmAngle))*2.0 - 1.0
}

func (a *Part) processChannel2(v string) {
	zap.L().Debug("process new value for channel2", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid value for channel2, should be an int: %v", err)
	}
	if value < MinPwmThrottle {
		value = MinPwmThrottle
	} else if value > MaxPwmThrottle {
		value = MaxPwmThrottle
	}
	a.throttle = ((float32(value)-MinPwmThrottle)/(MaxPwmThrottle-MinPwmThrottle))*2.0 - 1.0
}

func (a *Part) processChannel3(v string) {
	zap.L().Debug("process new value for channel3", zap.String("value", v))
}

func (a *Part) processChannel4(v string) {
	zap.L().Debug("process new value for channel4", zap.String("value", v))
}

func (a *Part) processChannel5(v string) {
	zap.L().Debug("process new value for channel5", zap.String("value", v))

	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid value for channel5, should be an int: %v", err)
	}

	if value < 1800 {
		if !a.ctrlRecord {
			zap.S().Infof("Update channel 5 with value %v, record: %v", true, false)
			a.ctrlRecord = true
		}
	} else {
		if a.ctrlRecord {
			zap.S().Infof("Update channel 5 with value %v, record: %v", false, true)
			a.ctrlRecord = false
		}
	}
}

func (a *Part) processChannel6(v string) {
	zap.L().Debug("process new value for channel6", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid value for channel6, should be an int: %v", err)
		return
	}
	if value > 1800 {
		if a.driveMode != events.DriveMode_PILOT {
			zap.S().Infof("Update channel 6 with value %v, new user_mode: %v", value, events.DriveMode_PILOT)
			a.driveMode = events.DriveMode_PILOT
		}
	} else {
		if a.driveMode != events.DriveMode_USER {
			zap.S().Infof("Update channel 6 with value %v, new user_mode: %v", value, events.DriveMode_USER)
		}
		a.driveMode = events.DriveMode_USER
	}
}

func (a *Part) publishLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(int(a.pubFrequency)))

	for {
		select {
		case <-ticker.C:
			a.publishValues()
		case <-a.cancel:
			ticker.Stop()
			return
		}
	}
}

func (a *Part) publishValues() {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.publishThrottle()
	a.publishSteering()
	a.publishDriveMode()
	a.publishSwitchRecord()
}

func (a *Part) publishThrottle() {
	throttle := events.ThrottleMessage{
		Throttle:   a.throttle,
		Confidence: 1.0,
	}
	throttleMessage, err := proto.Marshal(&throttle)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf throttle message: %v", err)
		return
	}
	zap.L().Debug("throttle channel", zap.Float32("throttle", a.throttle))
	publish(a.client, a.throttleTopic, &throttleMessage)
}

func (a *Part) publishSteering() {
	steering := events.SteeringMessage{
		Steering:   a.steering,
		Confidence: 1.0,
	}
	steeringMessage, err := proto.Marshal(&steering)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf steering message: %v", err)
		return
	}
	zap.L().Debug("steering channel", zap.Float32("steering", a.steering))
	publish(a.client, a.steeringTopic, &steeringMessage)
}

func (a *Part) publishDriveMode() {
	dm := events.DriveModeMessage{
		DriveMode: a.driveMode,
	}
	driveModeMessage, err := proto.Marshal(&dm)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf driveMode message: %v", err)
		return
	}
	publish(a.client, a.driveModeTopic, &driveModeMessage)
}

func (a *Part) publishSwitchRecord() {
	sr := events.SwitchRecordMessage{
		Enabled: !a.ctrlRecord,
	}
	switchRecordMessage, err := proto.Marshal(&sr)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf SwitchRecord message: %v", err)
		return
	}
	publish(a.client, a.switchRecordTopic, &switchRecordMessage)
}

var publish = func(client mqtt.Client, topic string, payload *[]byte) {
	client.Publish(topic, 0, false, *payload)
}
