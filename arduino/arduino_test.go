package arduino

import (
	"bufio"
	"fmt"
	"github.com/cyrilix/robocar-base/mqttdevice"
	"github.com/cyrilix/robocar-base/testtools"
	"github.com/cyrilix/robocar-base/types"
	"net"
	"testing"
	"time"
)

func TestArduinoPart_Update(t *testing.T) {
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

	a := Part{serial: conn, pubFrequency: 100, pub: testtools.NewFakePublisher()}
	go a.Start()

	channel1, channel2, channel3, channel4, channel5, channel6, distanceCm := 678, 910, 1112, 1678, 1910, 112, 128
	cases := []struct {
		name, content                      string
		expectedThrottle, expectedSteering float32
		expectedDriveMode                  types.DriveMode
		expectedSwitchRecord               bool
		expectedDistanceCm                 int
	}{
		{"Good value",
			fmt.Sprintf("12345,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"Unparsable line",
			"12350,invalid line\n",
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"Switch record on",
			fmt.Sprintf("12355,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 998, channel6, distanceCm),
			-1., -1., types.DriveModeUser, true, distanceCm},

		{"Switch record off",
			fmt.Sprintf("12360,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1987, channel6, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"Switch record off",
			fmt.Sprintf("12365,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1850, channel6, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"Switch record on",
			fmt.Sprintf("12370,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1003, channel6, distanceCm),
			-1., -1., types.DriveModeUser, true, distanceCm},

		{"DriveMode: user",
			fmt.Sprintf("12375,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 998, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"DriveMode: pilot",
			fmt.Sprintf("12380,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1987, distanceCm),
			-1., -1., types.DriveModePilot, false, distanceCm},
		{"DriveMode: pilot",
			fmt.Sprintf("12385,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1850, distanceCm),
			-1., -1., types.DriveModePilot, false, distanceCm},

		// DriveMode: user
		{"DriveMode: user",
			fmt.Sprintf("12390,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1003, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},

		{"Sterring: over left",
			fmt.Sprintf("12395,%d,%d,%d,%d,%d,%d,50,%d\n", 99, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"Sterring: left",
			fmt.Sprintf("12400,%d,%d,%d,%d,%d,%d,50,%d\n", 998, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -0.93, types.DriveModeUser, false, distanceCm},
		{"Sterring: middle",
			fmt.Sprintf("12405,%d,%d,%d,%d,%d,%d,50,%d\n", 1450, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -0.04, types.DriveModeUser, false, distanceCm},
		{"Sterring: right",
			fmt.Sprintf("12410,%d,%d,%d,%d,%d,%d,50,%d\n", 1958, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., 0.96, types.DriveModeUser, false, distanceCm},
		{"Sterring: over right",
			fmt.Sprintf("12415,%d,%d,%d,%d,%d,%d,50,%d\n", 2998, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., 1., types.DriveModeUser, false, distanceCm},

		{"Throttle: over down",
			fmt.Sprintf("12420,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 99, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., types.DriveModeUser, false, distanceCm},
		{"Throttle: down",
			fmt.Sprintf("12425,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 998, channel3, channel4, channel5, channel6, distanceCm),
			-0.95, -1., types.DriveModeUser, false, distanceCm},
		{"Throttle: stop",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 1450, channel3, channel4, channel5, channel6, distanceCm),
			-0.03, -1., types.DriveModeUser, false, distanceCm},
		{"Throttle: up",
			fmt.Sprintf("12435,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 1948, channel3, channel4, channel5, channel6, distanceCm),
			0.99, -1., types.DriveModeUser, false, distanceCm},
		{"Throttle: over up",
			fmt.Sprintf("12440,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 2998, channel3, channel4, channel5, channel6, distanceCm),
			1., -1., types.DriveModeUser, false, distanceCm},

		{"Distance cm",
			fmt.Sprintf("12445,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, channel6, 43),
			-1., -1., types.DriveModeUser, false, 43},
		{"Distance cm with \r",
			fmt.Sprintf("12450,%d,%d,%d,%d,%d,%d,50,%d\r\n", channel1, channel2, channel3, channel4, channel5, channel6, 43),
			-1., -1., types.DriveModeUser, false, 43},
	}

	for _, c := range cases {
		w := bufio.NewWriter(client)
		_, err := w.WriteString(c.content)
		if err != nil {
			t.Errorf("unable to send test content: %v", c.content)
		}
		err = w.Flush()
		if err != nil {
			t.Error("unable to flush content")
		}

		time.Sleep(5 * time.Millisecond)
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
		if a.distanceCm != c.expectedDistanceCm {
			t.Errorf("%s: bad distanceCm, expected: %v"+
				", actual:%v", c.name, c.expectedDistanceCm, a.distanceCm)
		}
		a.mutex.Unlock()
	}
}

func TestPublish(t *testing.T) {
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
	p := testtools.NewFakePublisher()
	a := Part{serial: conn, pub: p, pubFrequency: pubFrequency, topicBase: "car/part/arduino/"}
	go a.Start()
	defer a.Stop()

	cases := []struct {
		throttle, steering                                                                            float64
		driveMode                                                                                     types.DriveMode
		switchRecord                                                                                  bool
		distanceCm                                                                                    int
		expectedThrottle, expectedSteering, expectedDriveMode, expectedSwitchRecord, expectedDistance mqttdevice.MqttValue
	}{
		{-1, 1, types.DriveModeUser, false, 55,
			mqttdevice.NewMqttValue(types.Throttle{Value: -1., Confidence: 1.}),
			mqttdevice.NewMqttValue(types.Steering{Value: 1.0, Confidence: 1.}),
			mqttdevice.NewMqttValue("user"),
			mqttdevice.NewMqttValue("OFF"),
			mqttdevice.NewMqttValue("55")},
		{0, 0, types.DriveModePilot, true, 43,
			mqttdevice.NewMqttValue(types.Throttle{Confidence: 1.}),
			mqttdevice.NewMqttValue(types.Steering{Confidence: 1.}),
			mqttdevice.NewMqttValue("pilot"),
			mqttdevice.NewMqttValue("ON"),
			mqttdevice.NewMqttValue("43")},
		{0.87, -0.58, types.DriveModePilot, true, 21,
			mqttdevice.NewMqttValue(types.Throttle{Value: 0.87, Confidence: 1.}),
			mqttdevice.NewMqttValue(types.Steering{Value: -0.58, Confidence: 1.}),
			mqttdevice.NewMqttValue("pilot"),
			mqttdevice.NewMqttValue("ON"),
			mqttdevice.NewMqttValue("21")},
	}

	for _, c := range cases {
		a.mutex.Lock()
		a.throttle = c.throttle
		a.steering = c.steering
		a.driveMode = c.driveMode
		a.ctrlRecord = c.switchRecord
		a.distanceCm = c.distanceCm
		a.mutex.Unlock()

		time.Sleep(time.Second / time.Duration(int(pubFrequency)))
		time.Sleep(500 * time.Millisecond)

		if v := p.PublishedEvent("car/part/arduino/throttle"); string(v) != string(c.expectedThrottle) {
			t.Errorf("msg(car/part/arduino/throttle): %v, wants %v", v, c.expectedThrottle)
		}
		if v := p.PublishedEvent("car/part/arduino/steering"); string(v) != string(c.expectedSteering) {
			t.Errorf("msg(car/part/arduino/steering): %v, wants %v", v, c.expectedSteering)
		}
		if v := p.PublishedEvent("car/part/arduino/drive_mode"); string(v) != string(c.expectedDriveMode) {
			val, _ := v.StringValue()
			t.Errorf("msg(car/part/arduino/drive_mode): %v, wants %v", val, string(c.expectedDriveMode))
		}
		if v := p.PublishedEvent("car/part/arduino/switch_record"); string(v) != string(c.expectedSwitchRecord) {
			t.Errorf("msg(car/part/arduino/switch_record): %v, wants %v", v, c.expectedSwitchRecord)
		}
		if v := p.PublishedEvent("car/part/arduino/distance_cm"); string(v) != string(c.expectedDistance) {
			t.Errorf("msg(car/part/arduino/distance_cm): %v, wants %v", v, c.expectedThrottle)
		}
	}
}
