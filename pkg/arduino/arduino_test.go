package arduino

import (
	"bufio"
	"fmt"
	"github.com/cyrilix/robocar-protobuf/go/events"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"google.golang.org/protobuf/proto"
	"math"
	"net"
	"sync"
	"testing"
	"time"
)

const (
	MinPwmAngle = 999.0
	MaxPwmAngle = 1985.0
)

var (
	MiddlePwmAngle = int((MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle)
)

func TestArduinoPart_Update(t *testing.T) {
	oldPublish := publish
	defer func() { publish = oldPublish }()

	publish = func(client mqtt.Client, topic string, payload *[]byte) {}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Fatalf("unable to init connection for test")
	}
	defer func() {
		if err := ln.Close(); err != nil {
			t.Errorf("unable to close resource: %v", err)
		}
	}()

	serialClient, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Fatalf("unable to init connection for test")
	}
	defer func() {
		if err := serialClient.Close(); err != nil {
			t.Errorf("unable to close resource: %v", err)
		}
	}()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("unable to init connection for test")
	}
	defer func() {
		if err := conn.Close(); err != nil {
			t.Errorf("unable to close resource: %v", err)
		}
	}()

	a := Part{client: nil, serial: conn, pubFrequency: 100, pwmSteeringConfig: NewAsymetricPWMSteeringConfig(MinPwmAngle, MaxPwmAngle, MiddlePwmAngle)}
	go func() {
		err := a.Start()
		if err != nil {
			t.Errorf("unsable to start part: %v", err)
			t.Fail()
		}
	}()

	channel1, channel2, channel3, channel4, channel5, channel6, distanceCm := 678, 910, 1112, 1678, 1910, 112, 128
	cases := []struct {
		name, content                      string
		expectedThrottle, expectedSteering float32
		expectedDriveMode                  events.DriveMode
		expectedSwitchRecord               bool
	}{
		{"Good value",
			fmt.Sprintf("12345,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, false},
		{"Unparsable line",
			"12350,invalid line\n",
			-1., -1., events.DriveMode_USER, false},
		{"Switch record on",
			fmt.Sprintf("12355,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 998, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, true},

		{"Switch record off",
			fmt.Sprintf("12360,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1987, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, false},
		{"Switch record off",
			fmt.Sprintf("12365,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1850, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, false},
		{"Switch record on",
			fmt.Sprintf("12370,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1003, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, true},

		{"DriveMode: user",
			fmt.Sprintf("12375,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 998, distanceCm),
			-1., -1., events.DriveMode_USER, false},
		{"DriveMode: pilot",
			fmt.Sprintf("12380,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1987, distanceCm),
			-1., -1., events.DriveMode_PILOT, false},
		{"DriveMode: pilot",
			fmt.Sprintf("12385,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1850, distanceCm),
			-1., -1., events.DriveMode_PILOT, false},

		// DriveMode: user
		{"DriveMode: user",
			fmt.Sprintf("12390,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1003, distanceCm),
			-1., -1., events.DriveMode_USER, false},

		{"Sterring: over left",
			fmt.Sprintf("12395,%d,%d,%d,%d,%d,%d,50,%d\n", 99, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, false},
		{"Sterring: left",
			fmt.Sprintf("12400,%d,%d,%d,%d,%d,%d,50,%d\n", int(MinPwmAngle+40), channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -0.92, events.DriveMode_USER, false},
		{"Sterring: middle",
			fmt.Sprintf("12405,%d,%d,%d,%d,%d,%d,50,%d\n", 1450, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -0.09, events.DriveMode_USER, false},
		{"Sterring: right",
			fmt.Sprintf("12410,%d,%d,%d,%d,%d,%d,50,%d\n", 1958, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., 0.95, events.DriveMode_USER, false},
		{"Sterring: over right",
			fmt.Sprintf("12415,%d,%d,%d,%d,%d,%d,50,%d\n", 2998, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., 1., events.DriveMode_USER, false},

		{"Throttle: over down",
			fmt.Sprintf("12420,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 99, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., events.DriveMode_USER, false},
		{"Throttle: down",
			fmt.Sprintf("12425,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 998, channel3, channel4, channel5, channel6, distanceCm),
			-0.95, -1., events.DriveMode_USER, false},
		{"Throttle: stop",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 1450, channel3, channel4, channel5, channel6, distanceCm),
			-0.03, -1., events.DriveMode_USER, false},
		{"Throttle: up",
			fmt.Sprintf("12435,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 1948, channel3, channel4, channel5, channel6, distanceCm),
			0.99, -1., events.DriveMode_USER, false},
		{"Throttle: over up",
			fmt.Sprintf("12440,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 2998, channel3, channel4, channel5, channel6, distanceCm),
			1., -1., events.DriveMode_USER, false},
	}

	for _, c := range cases {
		w := bufio.NewWriter(serialClient)
		_, err := w.WriteString(c.content)
		if err != nil {
			t.Errorf("unable to send test content: %v", c.content)
		}
		err = w.Flush()
		if err != nil {
			t.Error("unable to flush content")
		}

		time.Sleep(10 * time.Millisecond)
		a.mutex.Lock()
		if fmt.Sprintf("%0.2f", a.throttle) != fmt.Sprintf("%0.2f", c.expectedThrottle) {
			t.Errorf("%s: bad throttle value, expected: %0.2f, actual: %.2f", c.name, c.expectedThrottle, a.throttle)
		}
		if fmt.Sprintf("%0.2f", a.steering) != fmt.Sprintf("%0.2f", c.expectedSteering) {
			t.Errorf("%s: bad steering value, expected: %0.2f, actual: %.2f", c.name, c.expectedSteering, a.steering)
		}
		if a.driveMode != c.expectedDriveMode {
			t.Errorf("%s: bad drive mode, expected: %v, actual:%v", c.name, c.expectedDriveMode, a.driveMode)
		}
		if a.ctrlRecord != c.expectedSwitchRecord {
			t.Errorf("%s: bad switch record, expected: %v, actual:%v", c.name, c.expectedSwitchRecord, a.ctrlRecord)
		}
		a.mutex.Unlock()
	}
}

func TestPublish(t *testing.T) {
	oldPublish := publish
	defer func() { publish = oldPublish }()

	var muPublishedEvents sync.Mutex
	pulishedEvents := make(map[string][]byte)
	publish = func(client mqtt.Client, topic string, payload *[]byte) {
		muPublishedEvents.Lock()
		defer muPublishedEvents.Unlock()
		pulishedEvents[topic] = *payload
	}

	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		t.Fatalf("unable to init connection for test")
	}
	defer ln.Close()

	client, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		t.Fatalf("unable to init connection for test")
	}
	defer client.Close()

	conn, err := ln.Accept()
	if err != nil {
		t.Fatalf("unable to init connection for test")
	}
	defer conn.Close()

	pubFrequency := 100.
	a := Part{
		client:            nil,
		serial:            conn,
		pubFrequency:      pubFrequency,
		throttleTopic:     "car/part/arduino/throttle",
		steeringTopic:     "car/part/arduino/steering",
		driveModeTopic:    "car/part/arduino/drive_mode",
		switchRecordTopic: "car/part/arduino/switch_record",
		cancel:            make(chan interface{}),
	}
	go a.Start()
	defer a.Stop()

	cases := []struct {
		throttle, steering   float32
		driveMode            events.DriveMode
		switchRecord         bool
		expectedThrottle     events.ThrottleMessage
		expectedSteering     events.SteeringMessage
		expectedDriveMode    events.DriveModeMessage
		expectedSwitchRecord events.SwitchRecordMessage
	}{
		{-1, 1, events.DriveMode_USER, true,
			events.ThrottleMessage{Throttle: -1., Confidence: 1.},
			events.SteeringMessage{Steering: 1.0, Confidence: 1.},
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SwitchRecordMessage{Enabled: false},
		},
		{0, 0, events.DriveMode_PILOT, false,
			events.ThrottleMessage{Throttle: 0., Confidence: 1.},
			events.SteeringMessage{Steering: 0., Confidence: 1.},
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SwitchRecordMessage{Enabled: true},
		},
		{0.87, -0.58, events.DriveMode_PILOT, false,
			events.ThrottleMessage{Throttle: 0.87, Confidence: 1.},
			events.SteeringMessage{Steering: -0.58, Confidence: 1.},
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SwitchRecordMessage{Enabled: true},
		},
	}

	for _, c := range cases {
		a.mutex.Lock()
		a.throttle = c.throttle
		a.steering = c.steering
		a.driveMode = c.driveMode
		a.ctrlRecord = c.switchRecord
		a.mutex.Unlock()

		time.Sleep(time.Second / time.Duration(int(pubFrequency)))
		time.Sleep(500 * time.Millisecond)

		var throttleMsg events.ThrottleMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/throttle"], &throttleMsg)
		muPublishedEvents.Unlock()
		if throttleMsg.String() != c.expectedThrottle.String() {
			t.Errorf("msg(car/part/arduino/throttle): %v, wants %v", throttleMsg, c.expectedThrottle)
		}

		var steeringMsg events.SteeringMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/steering"], &steeringMsg)
		muPublishedEvents.Unlock()
		if steeringMsg.String() != c.expectedSteering.String() {
			t.Errorf("msg(car/part/arduino/steering): %v, wants %v", steeringMsg, c.expectedSteering)
		}

		var driveModeMsg events.DriveModeMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/drive_mode"], &driveModeMsg)
		muPublishedEvents.Unlock()
		if driveModeMsg.String() != c.expectedDriveMode.String() {
			t.Errorf("msg(car/part/arduino/drive_mode): %v, wants %v", driveModeMsg, c.expectedDriveMode)
		}

		var switchRecordMsg events.SwitchRecordMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/switch_record"], &switchRecordMsg)
		muPublishedEvents.Unlock()
		if switchRecordMsg.String() != c.expectedSwitchRecord.String() {
			t.Errorf("msg(car/part/arduino/switch_record): %v, wants %v", switchRecordMsg, c.expectedSwitchRecord)
		}
	}
}

func unmarshalMsg(t *testing.T, payload []byte, msg proto.Message) {
	err := proto.Unmarshal(payload, msg)
	if err != nil {
		t.Errorf("unable to unmarshal protobuf content to %T: %v", msg, err)
	}
}

func Test_convertPwmSteeringToPercent(t *testing.T) {
	type args struct {
		value  int
		middle int
		min    int
		max    int
	}
	tests := []struct {
		name string
		args args
		want float32
	}{
		{
			name: "middle",
			args: args{
				value:  (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 0.,
		},
		{
			name: "left",
			args: args{
				value:  MinPwmAngle,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: -1.,
		},
		{
			name: "mid-left",
			args: args{
				value:  int(math.Round((MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle - (MaxPwmAngle-MinPwmAngle)/4)),
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: -0.4989858,
		},
		{
			name: "over left",
			args: args{
				value:  MinPwmAngle - 100,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: -1.,
		},
		{
			name: "right",
			args: args{
				value:  MaxPwmAngle,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 1.,
		},
		{
			name: "mid-right",
			args: args{
				value:  int(math.Round((MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + (MaxPwmAngle-MinPwmAngle)/4)),
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 0.5010142,
		},
		{
			name: "over right",
			args: args{
				value:  MaxPwmAngle + 100,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 1.,
		},
		{
			name: "asymetric middle",
			args: args{
				value:  (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 0.,
		},
		{
			name: "asymetric mid-left",
			args: args{
				value:  int(math.Round(((MaxPwmAngle-MinPwmAngle)/2+MinPwmAngle+100-MinPwmAngle)/2) + MinPwmAngle),
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: -0.49915683,
		},
		{
			name: "asymetric left",
			args: args{
				value:  MinPwmAngle,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: -1.,
		},
		{
			name: "asymetric mid-right",
			args: args{
				value:  int(math.Round((MaxPwmAngle - (MaxPwmAngle-((MaxPwmAngle-MinPwmAngle)/2+MinPwmAngle+100))/2))),
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 0.50127226,
		},
		{
			name: "asymetric right",
			args: args{
				value:  MaxPwmAngle,
				middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				min:    MinPwmAngle,
				max:    MaxPwmAngle,
			},
			want: 1.,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertPwmSteeringToPercent(tt.args.value, tt.args.min, tt.args.max, tt.args.middle); got != tt.want {
				t.Errorf("convertPwmSteeringToPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}
