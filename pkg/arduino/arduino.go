package arduino

import (
	"bufio"
	"github.com/cyrilix/robocar-arduino/pkg/tools"
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
	MinPwmThrottle = 972.0
	MaxPwmThrottle = 1954.0
)

var (
	serialLineRegex    = regexp.MustCompile(`(?P<timestamp>\d+),(?P<channel_1>\d+),(?P<channel_2>\d+),(?P<channel_3>\d+),(?P<channel_4>\d+),(?P<channel_5>\d+),(?P<channel_6>-?\d+),(?P<channel_7>\d+),(?P<channel_8>\d+),(?P<channel_9>\d+),(?P<frequency>\d+)`)
	DefaultPwmThrottle = PWMConfig{
		Min:    MinPwmThrottle,
		Max:    MaxPwmThrottle,
		Middle: MinPwmThrottle + (MaxPwmThrottle-MinPwmThrottle)/2,
	}
)

type Part struct {
	client                                                                                 mqtt.Client
	throttleTopic, steeringTopic, driveModeTopic, switchRecordTopic, throttleFeedbackTopic string
	pubFrequency                                                                           float64
	serial                                                                                 io.Reader
	mutex                                                                                  sync.Mutex
	steering, secondarySteering                                                            float32
	throttle, secondaryThrottle                                                            float32
	throttleFeedback                                                                       float32
	ctrlRecord                                                                             bool
	driveMode                                                                              events.DriveMode
	cancel                                                                                 chan interface{}

	useSecondaryRc             bool
	pwmSteeringConfig          *PWMConfig
	pwmSecondarySteeringConfig *PWMConfig
	pwmThrottleConfig          *PWMConfig
	pwmSecondaryThrottleConfig *PWMConfig

	throttleFeedbackThresholds *tools.ThresholdConfig
}

type PWMConfig struct {
	Min    int
	Max    int
	Middle int
}

func NewPWMConfig(min, max int) *PWMConfig {
	return &PWMConfig{
		Min:    min,
		Max:    max,
		Middle: min + (max-min)/2,
	}
}

func NewAsymetricPWMConfig(min, max, middle int) *PWMConfig {
	c := NewPWMConfig(min, max)
	c.Middle = middle
	return c
}

type Option func(p *Part)

func WithThrottleConfig(throttleConfig *PWMConfig) Option {
	return func(p *Part) {
		p.pwmThrottleConfig = throttleConfig
	}
}

func WithSteeringConfig(steeringConfig *PWMConfig) Option {
	return func(p *Part) {
		p.pwmSteeringConfig = steeringConfig
	}
}

func WithSecondaryRC(throttleConfig, steeringConfig *PWMConfig) Option {
	return func(p *Part) {
		p.pwmSecondaryThrottleConfig = throttleConfig
		p.pwmSecondarySteeringConfig = steeringConfig
	}
}

func WithThrottleFeedbackConfig(filename string) Option {
	return func(p *Part) {
		if filename == "" {
			p.throttleFeedbackThresholds = tools.NewThresholdConfig()
			return
		}
		tc, err := tools.NewThresholdConfigFromJson(filename)
		if err != nil {
			zap.S().Panicf("unable to load ThresholdConfig from file %v: %v", filename, err)
		}
		p.throttleFeedbackThresholds = tc

	}
}

func NewPart(client mqtt.Client, name string, baud int, throttleTopic, steeringTopic, driveModeTopic,
	switchRecordTopic, throttleFeedbackTopic string, pubFrequency float64, options ...Option) *Part {
	c := &serial.Config{Name: name, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		zap.S().Panicw("unable to open serial port: %v", err)
	}
	p := &Part{
		client:                client,
		serial:                s,
		throttleTopic:         throttleTopic,
		steeringTopic:         steeringTopic,
		driveModeTopic:        driveModeTopic,
		switchRecordTopic:     switchRecordTopic,
		throttleFeedbackTopic: throttleFeedbackTopic,
		pubFrequency:          pubFrequency,
		driveMode:             events.DriveMode_INVALID,
		cancel:                make(chan interface{}),

		pwmSteeringConfig:          &DefaultPwmThrottle,
		pwmSecondarySteeringConfig: &DefaultPwmThrottle,
		pwmThrottleConfig:          &DefaultPwmThrottle,
		pwmSecondaryThrottleConfig: &DefaultPwmThrottle,

		throttleFeedbackThresholds: tools.NewThresholdConfig(),
	}

	for _, o := range options {
		o(p)
	}
	return p
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

		zap.L().Debug("raw line: %s", zap.String("raw", line))
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
	a.processChannel7(values[7])
	a.processChannel8(values[8])
	a.processChannel9(values[9])
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
	zap.L().Debug("process new value for steering on channel1", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid steering value for channel1, should be an int: %v", err)
	}
	a.steering = convertPwmSteeringToPercent(value, a.pwmSteeringConfig)
}

func convertPwmSteeringToPercent(value int, c *PWMConfig) float32 {
	if value < c.Min {
		value = c.Min
	} else if value > c.Max {
		value = c.Max
	}
	if value == c.Middle {
		return 0.
	}
	if value < c.Middle {
		return (float32(value) - float32(c.Middle)) / float32(c.Middle-c.Min)
	}
	//  middle < value < max
	return (float32(value) - float32(c.Middle)) / float32(c.Max-c.Middle)
}

func (a *Part) processChannel2(v string) {
	zap.L().Debug("process new throttle value on channel2", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid throttle value for channel2, should be an int: %v", err)
	}
	if value < a.pwmThrottleConfig.Min {
		value = a.pwmThrottleConfig.Min
	} else if value > a.pwmThrottleConfig.Max {
		value = a.pwmThrottleConfig.Max
	}

	throttle := 0.
	if value > a.pwmThrottleConfig.Middle {
		throttle = (float64(value) - float64(a.pwmThrottleConfig.Middle)) / float64(a.pwmThrottleConfig.Max-a.pwmThrottleConfig.Middle)
	}
	if value < a.pwmThrottleConfig.Middle {
		throttle = -1. * (float64(a.pwmThrottleConfig.Middle) - float64(value)) / (float64(a.pwmThrottleConfig.Middle - a.pwmThrottleConfig.Min))
	}

	a.throttle = float32(throttle)
}

func (a *Part) processChannel3(v string) {
	zap.L().Debug("process new value for channel3", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid throttle value for channel2, should be an int: %v", err)
	}
	a.useSecondaryRc = value > 1900
}

func (a *Part) processChannel4(v string) {
	zap.L().Debug("process new value for channel4", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid throttle value for channel2, should be an int: %v", err)
	}
	a.throttleFeedback = a.convertPwmFeedBackToPercent(value)
}

func (a *Part) processChannel5(v string) {
	zap.L().Debug("process new value for channel5", zap.String("value", v))

	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid value for channel5 'record', should be an int: %v", err)
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
		zap.S().Errorf("invalid value for channel6 'drive-mode', should be an int: %v", err)
		return
	}
	if value < 0 {
		// No value, ignore it
		return
	}
	if value <= 1800 && value > 1200 {
		if a.driveMode != events.DriveMode_COPILOT {
			zap.S().Infof("Update channel 6 'drive-mode' with value %v, new user_mode: %v", value, events.DriveMode_PILOT)
			a.driveMode = events.DriveMode_COPILOT
		}
	} else if value > 1800 {
		if a.driveMode != events.DriveMode_PILOT {
			zap.S().Infof("Update channel 6 'drive-mode' with value %v, new user_mode: %v", value, events.DriveMode_PILOT)
			a.driveMode = events.DriveMode_PILOT
		}
	} else {
		if a.driveMode != events.DriveMode_USER {
			zap.S().Infof("Update channel 6 'drive-mode' with value %v, new user_mode: %v", value, events.DriveMode_USER)
		}
		a.driveMode = events.DriveMode_USER
	}
}

func (a *Part) processChannel7(v string) {
	zap.L().Debug("process new value for secondary steering on channel7", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid steering value for channel7, should be an int: %v", err)
	}
	a.secondarySteering = convertPwmSteeringToPercent(value, a.pwmSecondarySteeringConfig)
}

func (a *Part) processChannel8(v string) {
	zap.L().Debug("process new throttle value on channel8", zap.String("value", v))
	value, err := strconv.Atoi(v)
	if err != nil {
		zap.S().Errorf("invalid throttle value for channel8, should be an int: %v", err)
	}
	if value < a.pwmSecondaryThrottleConfig.Min {
		value = a.pwmSecondaryThrottleConfig.Min
	} else if value > a.pwmSecondaryThrottleConfig.Max {
		value = a.pwmSecondaryThrottleConfig.Max
	}
	a.secondaryThrottle = ((float32(value)-float32(a.pwmSecondaryThrottleConfig.Min))/float32(a.pwmSecondaryThrottleConfig.Max-a.pwmSecondaryThrottleConfig.Min))*2.0 - 1.0
}

func (a *Part) processChannel9(v string) {
	zap.L().Debug("process new value for channel9", zap.String("value", v))
}

func (a *Part) Throttle() float32 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.useSecondaryRc {
		return a.secondaryThrottle
	}
	return a.throttle
}

func (a *Part) ThrottleFeedback() float32 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.throttleFeedback
}

func (a *Part) Steering() float32 {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	if a.useSecondaryRc {
		return a.secondarySteering
	}
	return a.steering
}

func (a *Part) DriveMode() events.DriveMode {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.driveMode
}

func (a *Part) SwitchRecord() bool {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.ctrlRecord
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
	a.publishThrottle()
	a.publishThrottleFeedback()
	a.publishSteering()
	a.publishDriveMode()
	a.publishSwitchRecord()
}

func (a *Part) publishThrottle() {
	throttle := events.ThrottleMessage{
		Throttle:   a.Throttle(),
		Confidence: 1.0,
	}
	throttleMessage, err := proto.Marshal(&throttle)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf throttle message: %v", err)
		return
	}
	zap.L().Debug("throttle channel", zap.Float32("throttle", a.throttle))
	publish(a.client, a.throttleTopic, throttleMessage)
}

func (a *Part) publishSteering() {
	steering := events.SteeringMessage{
		Steering:   a.Steering(),
		Confidence: 1.0,
	}
	steeringMessage, err := proto.Marshal(&steering)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf steering message: %v", err)
		return
	}
	zap.L().Debug("steering channel", zap.Float32("steering", a.steering))
	publish(a.client, a.steeringTopic, steeringMessage)
}

func (a *Part) publishThrottleFeedback() {
	tm := events.ThrottleMessage{
		Throttle:   a.ThrottleFeedback(),
		Confidence: 1.,
	}
	tfMessage, err := proto.Marshal(&tm)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf throttleFeedback message: %v", err)
		return
	}
	publish(a.client, a.throttleFeedbackTopic, tfMessage)
}

func (a *Part) publishDriveMode() {
	dm := events.DriveModeMessage{
		DriveMode: a.DriveMode(),
	}
	driveModeMessage, err := proto.Marshal(&dm)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf driveMode message: %v", err)
		return
	}
	publish(a.client, a.driveModeTopic, driveModeMessage)
}

func (a *Part) publishSwitchRecord() {
	sr := events.SwitchRecordMessage{
		Enabled: !a.SwitchRecord(),
	}
	switchRecordMessage, err := proto.Marshal(&sr)
	if err != nil {
		zap.S().Errorf("unable to marshal protobuf SwitchRecord message: %v", err)
		return
	}
	publish(a.client, a.switchRecordTopic, switchRecordMessage)
}

func (a *Part) convertPwmFeedBackToPercent(value int) float32 {
	return float32(a.throttleFeedbackThresholds.ValueOf(value))
}

var publish = func(client mqtt.Client, topic string, payload []byte) {
	client.Publish(topic, 0, false, payload)
}
