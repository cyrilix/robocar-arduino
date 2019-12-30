package arduino

import (
	"bufio"
	"github.com/cyrilix/robocar-base/mqttdevice"
	"github.com/cyrilix/robocar-base/types"
	"github.com/tarm/serial"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MinPwmAngle = 960.0
	MaxPwmAngle = 1980.0

	MinPwmThrottle = 972.0
	MaxPwmThrottle = 1954.0
)

var (
	serialLineRegex = regexp.MustCompile(`(?P<timestamp>\d+),(?P<channel_1>\d+),(?P<channel_2>\d+),(?P<channel_3>\d+),(?P<channel_4>\d+),(?P<channel_5>\d+),(?P<channel_6>\d+),(?P<frequency>\d+),(?P<distance_cm>\d+)?`)
)

type Part struct {
	pub          mqttdevice.Publisher
	topicBase    string
	pubFrequency float64
	serial       io.Reader
	mutex        sync.Mutex
	steering     float64
	throttle     float64
	distanceCm   int
	ctrlRecord   bool
	driveMode    types.DriveMode
	debug        bool
}

func NewPart(name string, baud int, pub mqttdevice.Publisher, topicBase string, pubFrequency float64, debug bool) *Part {
	c := &serial.Config{Name: name, Baud: baud}
	s, err := serial.OpenPort(c)
	if err != nil {
		log.Panicf("unable to open serial port: %v", err)
	}
	return &Part{serial: s, pub: pub, topicBase: topicBase, pubFrequency: pubFrequency, driveMode: types.DriveModeInvalid, debug: debug}
}

func (a *Part) Start() error {
	log.Printf("start arduino part")
	go a.publishLoop()
	for {
		buff := bufio.NewReader(a.serial)
		line, err := buff.ReadString('\n')
		if err == io.EOF || line == "" {
			log.Println("remote connection closed")
			break
		}

		if !serialLineRegex.MatchString(line) {
			log.Printf("invalid line: '%v'", line)
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
	a.processDistanceCm(values[8])
}

func (a *Part) Stop() {
	log.Printf("stop ArduinoPart")
	switch s := a.serial.(type) {
	case io.ReadCloser:
		if err := s.Close(); err != nil {
			log.Fatalf("unable to close serial port: %v", err)
		}
	}
}

func (a *Part) processChannel1(v string) {
	if a.debug {
		log.Printf("channel1: %v", v)
	}
	value, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid value for channel1, should be an int: %v", err)
	}
	if value < MinPwmAngle {
		value = MinPwmAngle
	} else if value > MaxPwmAngle {
		value = MaxPwmAngle
	}
	a.steering = ((float64(value)-MinPwmAngle)/(MaxPwmAngle-MinPwmAngle))*2.0 - 1.0
}

func (a *Part) processChannel2(v string) {
	if a.debug {
		log.Printf("channel2: %v", v)
	}
	value, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid value for channel2, should be an int: %v", err)
	}
	if value < MinPwmThrottle {
		value = MinPwmThrottle
	} else if value > MaxPwmThrottle {
		value = MaxPwmThrottle
	}
	a.throttle = ((float64(value)-MinPwmThrottle)/(MaxPwmThrottle-MinPwmThrottle))*2.0 - 1.0
}

func (a *Part) processChannel3(v string) {
	if a.debug {
		log.Printf("channel3: %v", v)
	}
}

func (a *Part) processChannel4(v string) {
	if a.debug {
		log.Printf("channel4: %v", v)
	}
}

func (a *Part) processChannel5(v string) {
	if a.debug {
		log.Printf("channel5: %v", v)
	}

	value, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid value for channel5, should be an int: %v", err)
	}

	if value < 1800 {
		if !a.ctrlRecord {
			log.Printf("Update channel 5 with value %v, record: %v", true, false)
			a.ctrlRecord = true
		}
	} else {
		if a.ctrlRecord {
			log.Printf("Update channel 5 with value %v, record: %v", false, true)
			a.ctrlRecord = false
		}
	}
}

func (a *Part) processChannel6(v string) {
	if a.debug {
		log.Printf("channel6: %v", v)
	}
	value, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid value for channel6, should be an int: %v", err)
		return
	}
	if value > 1800 {
		if a.driveMode != types.DriveModePilot {
			log.Printf("Update channel 6 with value %v, new user_mode: %v", value, types.DriveModeUser)
			a.driveMode = types.DriveModePilot
		}
	} else {
		if a.driveMode != types.DriveModeUser {
			log.Printf("Update channel 6 with value %v, new user_mode: %v", value, types.DriveModeUser)
		}
		a.driveMode = types.DriveModeUser
	}
}

func (a *Part) processDistanceCm(v string) {
	value, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid value for distanceCm, should be an int: %v", err)
		return
	}
	a.distanceCm = value
}

func (a *Part) publishLoop() {
	prefix := strings.TrimSuffix(a.topicBase, "/")
	for {
		a.publishValues(prefix)
		time.Sleep(time.Second / time.Duration(int(a.pubFrequency)))
	}
}

func (a *Part) publishValues(prefix string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.pub.Publish(prefix+"/throttle", mqttdevice.NewMqttValue(types.Throttle{Value: a.throttle, Confidence: 1.}))
	a.pub.Publish(prefix+"/steering", mqttdevice.NewMqttValue(types.Steering{Value: a.steering, Confidence: 1.}))
	a.pub.Publish(prefix+"/drive_mode", mqttdevice.NewMqttValue(types.ToString(a.driveMode)))
	a.pub.Publish(prefix+"/switch_record", mqttdevice.NewMqttValue(a.ctrlRecord))
	a.pub.Publish(prefix+"/distance_cm", mqttdevice.NewMqttValue(a.distanceCm))
}
