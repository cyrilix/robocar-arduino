package mqttdevice

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"io"
	"log"
)

type Publisher interface {
	Publish(topic string, payload interface{})
}

type Subscriber interface {
	Subscribe(topic string, mh MQTT.MessageHandler)
}

type MQTTPubSub interface {
	Publisher
	Subscriber
	io.Closer
}

type pahoMqttPubSub struct {
	Uri      string
	Username string
	Password string
	ClientId string
	Qos      int
	Retain   bool
	client   MQTT.Client
}

func NewPahoMqttPubSub(uri string, username string, password string, clientId string, qos int, retain bool) MQTTPubSub {
	p := pahoMqttPubSub{Uri: uri, Username: username, Password: password, ClientId: clientId, Qos: qos, Retain: retain}
	p.Connect()
	return &p
}

// Publish message to broker
func (p *pahoMqttPubSub) Publish(topic string, payload interface{}) {
	tokenResp := p.client.Publish(topic, byte(p.Qos), p.Retain, payload)
	if tokenResp.Error() != nil {
		log.Fatalf("%+v\n", tokenResp.Error())
	}
}

// Register func to execute on message
func (p *pahoMqttPubSub) Subscribe(topic string, callback MQTT.MessageHandler) {
	tokenResp := p.client.Subscribe(topic, byte(p.Qos), callback)
	if tokenResp.Error() != nil {
		log.Fatalf("%+v\n", tokenResp.Error())
	}
}

// Close connection to broker
func (p *pahoMqttPubSub) Close() error {
	p.client.Disconnect(500)
	return nil
}

func (p *pahoMqttPubSub) Connect() {
	if p.client != nil && p.client.IsConnected() {
		return
	}
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(p.Uri)
	opts.SetUsername(p.Username)
	opts.SetPassword(p.Password)
	opts.SetClientID(p.ClientId)
	opts.SetAutoReconnect(true)
	opts.SetDefaultPublishHandler(
		//define a function for the default message handler
		func(client MQTT.Client, msg MQTT.Message) {
			fmt.Printf("TOPIC: %s\n", msg.Topic())
			fmt.Printf("MSG: %s\n", msg.Payload())
		})

	//create and start a client using the above ClientOptions
	p.client = MQTT.NewClient(opts)
	if token := p.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}
