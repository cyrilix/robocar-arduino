package arduino

import (
	"bufio"
	"fmt"
	"net"
	"robocar/mode"
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

	a := ArduinoPart{serial: conn}
	go a.Run()

	channel1, channel2, channel3, channel4, channel5, channel6, distanceCm := 678, 910, 1112, 1678, 1910, 112, 128
	cases := []struct {
		name, content                      string
		expectedThrottle, expectedSteering float32
		expectedDriveMode                  mode.DriveMode
		expectedSwitchRecord               bool
		expectedDistanceCm                 int
	}{
		{"Good value",
			fmt.Sprintf("12345,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"Unparsable line",
			"12350,invalid line\n",
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"Switch record on",
			fmt.Sprintf("12355,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 998, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, true, distanceCm},

		{"Switch record off",
			fmt.Sprintf("12360,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1987, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"Switch record off",
			fmt.Sprintf("12365,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1850, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"Switch record on",
			fmt.Sprintf("12370,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, 1003, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, true, distanceCm},


		{"DriveMode: user",
			fmt.Sprintf("12375,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 998, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"DriveMode: pilot",
			fmt.Sprintf("12380,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1987, distanceCm),
			-1., -1., mode.DriveModePilot, false, distanceCm},
		{"DriveMode: pilot",
			fmt.Sprintf("12385,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1850, distanceCm),
			-1., -1., mode.DriveModePilot, false, distanceCm},

		// DriveMode: user
		{"DriveMode: user",
			fmt.Sprintf("12390,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, 1003, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},


		{"Sterring: over left",
			fmt.Sprintf("12395,%d,%d,%d,%d,%d,%d,50,%d\n", 99, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"Sterring: left",
			fmt.Sprintf("12400,%d,%d,%d,%d,%d,%d,50,%d\n", 998, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -0.93, mode.DriveModeUser, false, distanceCm},
		{"Sterring: middle",
			fmt.Sprintf("12405,%d,%d,%d,%d,%d,%d,50,%d\n", 1450, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., -0.04, mode.DriveModeUser, false, distanceCm},
		{"Sterring: right",
			fmt.Sprintf("12410,%d,%d,%d,%d,%d,%d,50,%d\n", 1958, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., 0.96, mode.DriveModeUser, false, distanceCm},
		{"Sterring: over right",
			fmt.Sprintf("12415,%d,%d,%d,%d,%d,%d,50,%d\n", 2998, channel2, channel3, channel4, channel5, channel6, distanceCm),
			-1., 1., mode.DriveModeUser, false, distanceCm},


		{"Throttle: over down",
			fmt.Sprintf("12420,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 99, channel3, channel4, channel5, channel6, distanceCm),
			-1., -1., mode.DriveModeUser, false, distanceCm},
		{"Throttle: down",
			fmt.Sprintf("12425,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 998, channel3, channel4, channel5, channel6, distanceCm),
			-0.95, -1., mode.DriveModeUser, false, distanceCm},
		{"Throttle: stop",
			fmt.Sprintf("12430,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 1450, channel3, channel4, channel5, channel6, distanceCm),
			-0.03, -1., mode.DriveModeUser, false, distanceCm},
		{"Throttle: up",
			fmt.Sprintf("12435,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 1948, channel3, channel4, channel5, channel6, distanceCm),
			0.99, -1., mode.DriveModeUser, false, distanceCm},
		{"Throttle: over up",
			fmt.Sprintf("12440,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, 2998, channel3, channel4, channel5, channel6, distanceCm),
			1., -1., mode.DriveModeUser, false, distanceCm},

		{"Distance cm",
			fmt.Sprintf("12445,%d,%d,%d,%d,%d,%d,50,%d\n", channel1, channel2, channel3, channel4, channel5, channel6, 43),
			-1., -1., mode.DriveModeUser, false, 43},
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

		time.Sleep(1* time.Millisecond)
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
			t.Errorf("%s: bad distanceCm, expected: %v" +
				", actual:%v", c.name, c.expectedDistanceCm, a.distanceCm)
		}
		a.mutex.Unlock()
	}
}
