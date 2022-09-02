package arduino

import (
	"bufio"
	"fmt"
	"github.com/cyrilix/robocar-arduino/pkg/tools"
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

	publish = func(client mqtt.Client, topic string, payload []byte) {}

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

	defaultPwmThrottleConfig := NewPWMConfig(MinPwmThrottle, MaxPwmThrottle)
	a := Part{client: nil, serial: conn, pubFrequency: 100,
		pwmSteeringConfig:          NewAsymetricPWMConfig(MinPwmAngle, MaxPwmAngle, MiddlePwmAngle),
		pwmThrottleConfig:          &DefaultPwmThrottle,
		pwmSecondaryThrottleConfig: &DefaultPwmThrottle,
		pwmSecondarySteeringConfig: NewPWMConfig(MinPwmThrottle, MaxPwmThrottle),
		throttleFeedbackThresholds: tools.NewThresholdConfig(),
	}
	go func() {
		err := a.Start()
		if err != nil {
			t.Errorf("unsable to start part: %v", err)
			t.Fail()
		}
	}()

	channel1, channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9 := 678, 910, 1012, 1678, 1910, 112, 0, 0, 0
	cases := []struct {
		name, content                      string
		throttlePwmConfig                  *PWMConfig
		expectedThrottle, expectedSteering float32
		expectedDriveMode                  events.DriveMode
		expectedSwitchRecord               bool
	}{
		{"Good value",
			fmt.Sprintf("12345,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1.,
			events.DriveMode_USER, false},
		{"Invalid line",
			"12350,invalid line\n", defaultPwmThrottleConfig,
			-1., -1., events.DriveMode_INVALID, false},
		{"Switch record on",
			fmt.Sprintf("12355,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, 998, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, true},

		{"Switch record off",
			fmt.Sprintf("12360,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, 1987, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},
		{"Switch record off",
			fmt.Sprintf("12365,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, 1850, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},
		{"Switch record on",
			fmt.Sprintf("12370,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, 1003, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, true},

		{"DriveMode: user",
			fmt.Sprintf("12375,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, channel5, 998, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},
		{"DriveMode: pilot",
			fmt.Sprintf("12380,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, channel5, 1987, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_PILOT, false},
		{"DriveMode: pilot",
			fmt.Sprintf("12385,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, channel5, 1850, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_PILOT, false},

		// DriveMode: user
		{"DriveMode: user",
			fmt.Sprintf("12390,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel2, channel3, channel4, channel5, 1003, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},

		{"Sterring: over left", fmt.Sprintf("12395,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", 99, channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},
		{"Sterring: left",
			fmt.Sprintf("12400,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", int(MinPwmAngle+40), channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -0.92, events.DriveMode_USER, false},
		{"Sterring: middle",
			fmt.Sprintf("12405,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", 1450, channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -0.09, events.DriveMode_USER, false},
		{"Sterring: right",
			fmt.Sprintf("12410,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", 1958, channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., 0.95, events.DriveMode_USER, false},
		{"Sterring: over right",
			fmt.Sprintf("12415,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", 2998, channel2, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., 1., events.DriveMode_USER, false},

		{"Throttle: over down",
			fmt.Sprintf("12420,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, 99, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},
		{"Throttle: down",
			fmt.Sprintf("12425,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, 998, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -0.95, -1., events.DriveMode_USER, false},
		{"Throttle: stop",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, 1450, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			NewPWMConfig(1000, 1900), 0.0, -1., events.DriveMode_USER, false},
		{"Throttle: up",
			fmt.Sprintf("12435,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, 1948, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, 0.99, -1., events.DriveMode_USER, false},
		{"Throttle: over up",
			fmt.Sprintf("12440,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, 2998, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			defaultPwmThrottleConfig, 1., -1., events.DriveMode_USER, false},
		{"Throttle: zero not middle",
			fmt.Sprintf("12440,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, 1600, channel3, channel4, channel5, channel6, channel7, channel8, channel9),
			&PWMConfig{1000, 1700, 1500},
			0.5, -1., events.DriveMode_USER, false},
		{"Use 2nd rc: use channels 7 and 8",
			fmt.Sprintf("12440,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", 1000, 1000, 1950, channel4, channel5, channel6, 2000, 2008, channel9),
			defaultPwmThrottleConfig, 1., 1, events.DriveMode_USER, false},
		{"Drive Mode: user",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel6, channel3, channel4, channel5, 900, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_USER, false},
		{"Drive Mode: pilot",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel6, channel3, channel4, channel5, 1950, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_PILOT, false},
		{"Drive Mode: no value",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,%d,%d,%d,50\n", channel1, channel6, channel3, channel4, channel5, -1, channel7, channel8, channel9),
			defaultPwmThrottleConfig, -1., -1., events.DriveMode_INVALID, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := bufio.NewWriter(serialClient)
			_, err := w.WriteString(c.content)
			if err != nil {
				t.Errorf("unable to send test content: %v", c.content)
			}
			err = w.Flush()
			if err != nil {
				t.Error("unable to flush content")
			}

			a.pwmThrottleConfig = c.throttlePwmConfig
			a.driveMode = events.DriveMode_INVALID

			time.Sleep(10 * time.Millisecond)
			a.mutex.Lock()
			a.mutex.Unlock()
			if fmt.Sprintf("%0.2f", a.Throttle()) != fmt.Sprintf("%0.2f", c.expectedThrottle) {
				t.Errorf("%s: bad throttle value, expected: %0.2f, actual: %.2f", c.name, c.expectedThrottle, a.Throttle())
			}
			if fmt.Sprintf("%0.2f", a.Steering()) != fmt.Sprintf("%0.2f", c.expectedSteering) {
				t.Errorf("%s: bad steering value, expected: %0.2f, actual: %.2f", c.name, c.expectedSteering, a.Steering())
			}
			if a.DriveMode() != c.expectedDriveMode {
				t.Errorf("%s: bad drive mode, expected: %v, actual:%v", c.name, c.expectedDriveMode, a.DriveMode())
			}
			if a.SwitchRecord() != c.expectedSwitchRecord {
				t.Errorf("%s: bad switch record, expected: %v, actual:%v", c.name, c.expectedSwitchRecord, a.SwitchRecord())
			}
		})
	}
}

func TestPublish(t *testing.T) {
	oldPublish := publish
	defer func() { publish = oldPublish }()

	var muPublishedEvents sync.Mutex
	pulishedEvents := make(map[string][]byte)
	publish = func(client mqtt.Client, topic string, payload []byte) {
		muPublishedEvents.Lock()
		defer muPublishedEvents.Unlock()
		pulishedEvents[topic] = payload
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
		client:                nil,
		serial:                conn,
		pubFrequency:          pubFrequency,
		throttleTopic:         "car/part/arduino/throttle/target",
		steeringTopic:         "car/part/arduino/steering",
		driveModeTopic:        "car/part/arduino/drive_mode",
		switchRecordTopic:     "car/part/arduino/switch_record",
		throttleFeedbackTopic: "car/part/arduino/throttle/feedback",
		cancel:                make(chan interface{}),
	}
	go a.Start()
	defer a.Stop()

	cases := []struct {
		throttle, steering       float32
		driveMode                events.DriveMode
		throttleFeedback         float32
		switchRecord             bool
		expectedThrottle         events.ThrottleMessage
		expectedSteering         events.SteeringMessage
		expectedDriveMode        events.DriveModeMessage
		expectedSwitchRecord     events.SwitchRecordMessage
		expectedThrottleFeedback events.ThrottleMessage
	}{
		{-1, 1, events.DriveMode_USER, 0.3, true,
			events.ThrottleMessage{Throttle: -1., Confidence: 1.},
			events.SteeringMessage{Steering: 1.0, Confidence: 1.},
			events.DriveModeMessage{DriveMode: events.DriveMode_USER},
			events.SwitchRecordMessage{Enabled: false},
			events.ThrottleMessage{Throttle: 0.3, Confidence: 1.},
		},
		{0, 0, events.DriveMode_PILOT, 0.4, false,
			events.ThrottleMessage{Throttle: 0., Confidence: 1.},
			events.SteeringMessage{Steering: 0., Confidence: 1.},
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SwitchRecordMessage{Enabled: true},
			events.ThrottleMessage{Throttle: 0.4, Confidence: 1.},
		},
		{0.87, -0.58, events.DriveMode_PILOT, 0.5, false,
			events.ThrottleMessage{Throttle: 0.87, Confidence: 1.},
			events.SteeringMessage{Steering: -0.58, Confidence: 1.},
			events.DriveModeMessage{DriveMode: events.DriveMode_PILOT},
			events.SwitchRecordMessage{Enabled: true},
			events.ThrottleMessage{Throttle: 0.5, Confidence: 1.},
		},
	}

	for _, c := range cases {
		a.mutex.Lock()
		a.throttle = c.throttle
		a.steering = c.steering
		a.driveMode = c.driveMode
		a.ctrlRecord = c.switchRecord
		a.throttleFeedback = c.throttleFeedback
		a.mutex.Unlock()

		time.Sleep(time.Second / time.Duration(int(pubFrequency)) * 2)

		var throttleMsg events.ThrottleMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/throttle/target"], &throttleMsg)
		muPublishedEvents.Unlock()
		if throttleMsg.String() != c.expectedThrottle.String() {
			t.Errorf("msg(car/part/arduino/throttle/target): %v, wants %v", throttleMsg.String(), c.expectedThrottle.String())
		}

		var steeringMsg events.SteeringMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/steering"], &steeringMsg)
		muPublishedEvents.Unlock()
		if steeringMsg.String() != c.expectedSteering.String() {
			t.Errorf("msg(car/part/arduino/steering): %v, wants %v", steeringMsg.String(), c.expectedSteering.String())
		}

		var driveModeMsg events.DriveModeMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/drive_mode"], &driveModeMsg)
		muPublishedEvents.Unlock()
		if driveModeMsg.String() != c.expectedDriveMode.String() {
			t.Errorf("msg(car/part/arduino/drive_mode): %v, wants %v", driveModeMsg.String(), c.expectedDriveMode.String())
		}

		var switchRecordMsg events.SwitchRecordMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/switch_record"], &switchRecordMsg)
		muPublishedEvents.Unlock()
		if switchRecordMsg.String() != c.expectedSwitchRecord.String() {
			t.Errorf("msg(car/part/arduino/switch_record): %v, wants %v", switchRecordMsg.String(), c.expectedSwitchRecord.String())
		}

		var throttleFeedbackMsg events.ThrottleMessage
		muPublishedEvents.Lock()
		unmarshalMsg(t, pulishedEvents["car/part/arduino/throttle/feedback"], &throttleFeedbackMsg)
		muPublishedEvents.Unlock()
		if throttleFeedbackMsg.String() != c.expectedThrottleFeedback.String() {
			t.Errorf("msg(car/part/arduino/throttle/feedback): %v, wants %v", throttleFeedbackMsg.String(), c.expectedThrottleFeedback.String())
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
		value          int
		steeringConfig *PWMConfig
	}
	tests := []struct {
		name string
		args args
		want float32
	}{
		{
			name: "middle",
			args: args{
				value: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 0.,
		},
		{
			name: "left",
			args: args{
				value: MinPwmAngle,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: -1.,
		},
		{
			name: "mid-left",
			args: args{
				value: int(math.Round((MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle - (MaxPwmAngle-MinPwmAngle)/4)),
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: -0.4989858,
		},
		{
			name: "over left",
			args: args{
				value: MinPwmAngle - 100,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: -1.,
		},
		{
			name: "right",
			args: args{
				value: MaxPwmAngle,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 1.,
		},
		{
			name: "mid-right",
			args: args{
				value: int(math.Round((MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + (MaxPwmAngle-MinPwmAngle)/4)),
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 0.5010142,
		},
		{
			name: "over right",
			args: args{
				value: MaxPwmAngle + 100,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 1.,
		},
		{
			name: "asymetric middle",
			args: args{
				value: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 0.,
		},
		{
			name: "asymetric mid-left",
			args: args{
				value: int(math.Round(((MaxPwmAngle-MinPwmAngle)/2+MinPwmAngle+100-MinPwmAngle)/2) + MinPwmAngle),
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: -0.49915683,
		},
		{
			name: "asymetric left",
			args: args{
				value: MinPwmAngle,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: -1.,
		},
		{
			name: "asymetric mid-right",
			args: args{
				value: int(math.Round((MaxPwmAngle - (MaxPwmAngle-((MaxPwmAngle-MinPwmAngle)/2+MinPwmAngle+100))/2))),
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 0.50127226,
		},
		{
			name: "asymetric right",
			args: args{
				value: MaxPwmAngle,
				steeringConfig: &PWMConfig{
					Middle: (MaxPwmAngle-MinPwmAngle)/2 + MinPwmAngle + 100,
					Min:    MinPwmAngle,
					Max:    MaxPwmAngle,
				},
			},
			want: 1.,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertPwmSteeringToPercent(tt.args.value, tt.args.steeringConfig); got != tt.want {
				t.Errorf("convertPwmSteeringToPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPart_convertPwmFeedBackToPercent(t *testing.T) {
	type fields struct {
	}
	type args struct {
		value int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   float32
	}{
		{
			name: "big value -> 0%",
			args: args{value: 1234567},
			want: 0.,
		},
		{
			name: "very slow",
			args: args{10000},
			want: 0.,
		},
		{
			name: "0.07 limit",
			args: args{8700},
			want: 0.07,
		},
		{
			name: "0.075",
			args: args{value: 8700 - (8700-4800)/2},
			want: 0.075,
		},
		{
			name: "0.08",
			args: args{value: 4800},
			want: 0.08,
		},
		{
			name: "1.0",
			args: args{value: 548},
			want: 1.,
		},
		{
			name: "under lower limit",
			args: args{value: 520},
			want: 1.,
		},
		{
			name: "invalid value",
			args: args{value: 499},
			want: 0.,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Part{throttleFeedbackThresholds: tools.NewThresholdConfig()}
			if got := a.convertPwmFeedBackToPercent(tt.args.value); got != tt.want {
				t.Errorf("convertPwmFeedBackToPercent() = %v, want %v", got, tt.want)
			}
		})
	}
}
