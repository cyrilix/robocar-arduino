package mqttdevice

import (
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"testing"
	"warmup4ie2mqtt/testtools"
)

func TestIntegration(t *testing.T) {

	ctx, mqttC, mqttUri := testtools.MqttContainer(t)
	defer mqttC.Terminate(ctx)

	t.Run("ConnectAndClose", func(t *testing.T) {
		t.Logf("Mqtt connection %s ready", mqttUri)

		p := pahoMqttPubSub{Uri: mqttUri, ClientId: "TestMqtt", Username: "guest", Password: "guest"}
		p.Connect()
		p.Close()
	})
	t.Run("Publish", func(t *testing.T) {
		options := mqtt.NewClientOptions().AddBroker(mqttUri)
		options.SetUsername("guest")
		options.SetPassword("guest")

		client := mqtt.NewClient(options)
		token := client.Connect()
		defer client.Disconnect(100)
		token.Wait()
		if token.Error() != nil {
			t.Fatalf("unable to connect to mqtt broker: %v\n", token.Error())
		}

		c := make(chan string)
		defer close(c)
		client.Subscribe("test/publish", 0, func(client mqtt.Client, message mqtt.Message) {
			c <- string(message.Payload())
		}).Wait()

		p := pahoMqttPubSub{Uri: mqttUri, ClientId: "TestMqtt", Username: "guest", Password: "guest"}
		p.Connect()
		defer p.Close()

		p.Publish("test/publish", "Test1234")
		result := <-c
		if result != "Test1234" {
			t.Fatalf("bad message: %v\n", result)
		}

	})
}
