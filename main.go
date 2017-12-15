package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// GCP settings passed by CLI parameters
var (
	project     = flag.String("project_id", "", "Project ID")
	registry    = flag.String("registry_id", "", "Registry ID")
	device      = flag.String("device_id", "", "Device ID")
	algorithm   = flag.String("algorithm", "", "Private Key Algorithm")
	private_key = flag.String("private_key_file", "", "Path to Private Key File")
)

// MQTT parameters
const (
	topicType = "events"   // or "state"
	qos = 1                // QoS 2 isn't supported in GCP
	retain = false
	username = "unused"    // always this value in GCP
	region = "us-central1" // GCP region
)

func main() {
	flag.Parse()

	// generate MQTT client
	client_id := fmt.Sprintf(
		"projects/%s/locations/%s/registries/%s/devices/%s",
		*project, region, *registry, *device)

	// load private key
	keyData, err := ioutil.ReadFile(*private_key)
	if err != nil { panic(err) }

	var key interface{}
	switch *algorithm {
	case "RS256":
		key, err = jwt.ParseRSAPrivateKeyFromPEM(keyData)
	case "ES256":
		key, err = jwt.ParseECPrivateKeyFromPEM(keyData)
	default:
		panic(fmt.Errorf("Unknown algorithm: %s", *algorithm))
	}
	if err != nil { panic(err) }

	// generate JWT as the MQTT password
	t := time.Now()
	token := jwt.NewWithClaims(jwt.GetSigningMethod(*algorithm), &jwt.StandardClaims{
		IssuedAt: t.Unix(),
		ExpiresAt: t.Add(time.Minute * 20).Unix(),
		Audience: *project,
	})
	pass, err := token.SignedString(key)
	if err != nil { panic(err) }

	// configure MQTT client
	opts := mqtt.NewClientOptions().
		AddBroker("ssl://mqtt.googleapis.com:8883").
		SetClientID(client_id).
		SetUsername(username).
		SetTLSConfig(&tls.Config{ MinVersion: tls.VersionTLS12 }).
		SetPassword(pass).
		SetProtocolVersion(4) // Use MQTT 3.1.1

	conn := mqtt.NewClient(opts)

	// connect to GCP Cloud IoT Core
	fmt.Println("Connecting...")
	tok := conn.Connect()
	if err := tok.Error(); err != nil { panic(err) }
	if !tok.WaitTimeout(time.Second * 5) { panic(fmt.Errorf("Connection Timeout")) }
	if err := tok.Error(); err != nil { panic(err) }

	// generate topic
	topic := fmt.Sprintf("/devices/%s/%s", device, topicType)

	// publish message 5 times
	for i := 0; i < 5; i++ {
		str := fmt.Sprintf("Message %d", i)
		fmt.Printf("Publishing: '%s'\n", str)
		conn.Publish(topic, qos, retain, str)
		time.Sleep(time.Millisecond * 500)
	}

	// need mqtt reconnect each 20 minutes for long use

	// disconnect
	fmt.Println("Disconnecting...")
	conn.Disconnect(1000)
}
