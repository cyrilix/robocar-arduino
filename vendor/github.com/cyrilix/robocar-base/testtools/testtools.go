package testtools

import (
	"context"
	"fmt"
	"github.com/cyrilix/robocar-base/mqttdevice"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"sync"
	"testing"
)

func MqttContainer(t *testing.T) (context.Context, testcontainers.Container, string) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "eclipse-mosquitto",
		ExposedPorts: []string{"1883/tcp"},
		WaitingFor:   wait.ForLog("listen socket on port 1883."),
	}
	mqttC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}

	ip, err := mqttC.Host(ctx)
	if err != nil {
		t.Error(err)
	}
	port, err := mqttC.MappedPort(ctx, "1883/tcp")
	if err != nil {
		t.Error(err)
	}

	mqttUri := fmt.Sprintf("tcp://%s:%d", ip, port.Int())
	return ctx, mqttC, mqttUri
}

func NewFakePublisher() *FakePublisher {
	return &FakePublisher{msg: make(map[string]mqttdevice.MqttValue)}
}

type FakePublisher struct {
	muMsg sync.Mutex
	msg   map[string]mqttdevice.MqttValue
}

func (f *FakePublisher) Publish(topic string, payload mqttdevice.MqttValue) {
	f.muMsg.Lock()
	defer f.muMsg.Unlock()
	f.msg[topic] = payload
}

func (f *FakePublisher) PublishedEvent(topic string) mqttdevice.MqttValue {
	f.muMsg.Lock()
	defer f.muMsg.Unlock()
	return f.msg[topic]
}

type fakeMessage struct {
	qos     byte
	topic   string
	payload []byte
	acked   bool
}

func (f *fakeMessage) Duplicate() bool {
	return false
}

func (f *fakeMessage) Qos() byte {
	return f.qos
}

func (f *fakeMessage) Retained() bool {
	return false
}

func (f *fakeMessage) Topic() string {
	return f.topic
}

func (f *fakeMessage) MessageID() uint16 {
	return 1234
}

func (f *fakeMessage) Payload() []byte {
	return f.payload
}

func (f *fakeMessage) Ack() {
	f.acked = true
}

func NewFakeMessage(topic string, payload []byte) mqtt.Message {
	return &fakeMessage{
		qos:     0,
		topic:   topic,
		payload: payload,
		acked:   false,
	}
}
