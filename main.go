package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"io/ioutil"
	"strconv"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/phayes/freeport"

	"github.com/VolantMQ/volantmq"
	"github.com/VolantMQ/volantmq/auth"
	"github.com/VolantMQ/volantmq/transport"
	"github.com/VolantMQ/volantmq/packet"
)

// GCP settings passed by CLI parameters
var (
	project     = flag.String("project_id", "", "Project ID")
	registry    = flag.String("registry_id", "", "Registry ID")
	device      = flag.String("device_id", "", "Device ID")
	algorithm   = flag.String("algorithm", "", "Private Key Algorithm")
	region      = flag.String("region", "us-central1", "GCP Region")
	private_key = flag.String("private_key_file", "", "Path to Private Key File")
	public_key  = flag.String("public_key_file", "", "Path to Public Key File")
	run_test    = flag.Bool("run_test", false, "Run GCP simulation test using VolantMQ")
	server      = flag.String("server", "ssl://mqtt.googleapis.com:8883", "MQTT Server")
)

// MQTT parameters
const (
	topicType = "events"   // or "state"
	qos = 1                // QoS 2 isn't supported in GCP
	retain = false
	username = "unused"    // always this value in GCP
)

type testGcpAuth struct {}

func (a testGcpAuth) Password(user, pass string) auth.Status {
	if user != "unused" { return auth.StatusDeny }

	token, err := jwt.Parse(pass, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		keyData, err := ioutil.ReadFile(*public_key)
		if err != nil { return nil, err }
		key, err := jwt.ParseRSAPublicKeyFromPEM(keyData)
		if err != nil { return nil, err }
		return key, nil
	})
	if err != nil {
		fmt.Println(err)
		return auth.StatusDeny
	}

	// fmt.Println(token.Claims)
	claims := token.Claims.(jwt.MapClaims)
	if !claims.VerifyAudience(*project, true) {
		fmt.Println("Invalid audience")
		return auth.StatusDeny
	}

	t := time.Now()
	if !claims.VerifyExpiresAt(t.Unix(), true) {
		fmt.Println("Invalid expires at")
		return auth.StatusDeny
	}
	if !claims.VerifyIssuedAt(t.Unix(), true) {
		fmt.Println("Invalid issued at")
		return auth.StatusDeny
	}

	// fmt.Println("JWT verification success!")
	return auth.StatusAllow
}

func (a testGcpAuth) ACL(clientId, user, topic string, access auth.AccessType) auth.Status {
	expClientId := fmt.Sprintf(
		"projects/%s/locations/%s/registries/%s/devices/%s",
		*project, *region, *registry, *device)

	fmt.Println("Checking ACL:", clientId, user, topic, access)
	if  user == "unused" &&
		topic == fmt.Sprintf("/devices/%s/%s", *device, topicType) &&
		clientId == expClientId {
		return auth.StatusAllow
	}

	panic("ACL failed")
	// return auth.StatusDeny
}

func main() {
	flag.Parse()

	// start test server with VolantMQ
	if *run_test {
		port, err := freeport.GetFreePort()
		if err != nil { panic(err) }
		*server = fmt.Sprintf("tcp://:%d", port)
		listenerStatus := func(id string, status string) {
			// fmt.Println("Listener status:", id, status)
		}

		if err := auth.Register("internal", &testGcpAuth{}); err != nil { panic(err) }
		authMng, err := auth.NewManager("internal")
		if err != nil { panic(err) }

		srvCfg := volantmq.NewServerConfig()
		srvCfg.Authenticators = "internal"
		srvCfg.AllowedVersions = map[packet.ProtocolVersion]bool{
			packet.ProtocolV31: false,
			packet.ProtocolV311: true,
			packet.ProtocolV50: false,
		}
		srvCfg.TransportStatus = listenerStatus
		srv, err := volantmq.NewServer(srvCfg)
		if err != nil { panic(err) }

		err = srv.ListenAndServe(transport.NewConfigTCP(&transport.Config{
			Port: strconv.Itoa(port), AuthManager: authMng}))
		if err != nil { panic(err) }
		defer func() {
			time.Sleep(time.Millisecond * 100) // workaround for negative WaitGroup
			srv.Close()
		}()
	}

	// generate MQTT client
	client_id := fmt.Sprintf(
		"projects/%s/locations/%s/registries/%s/devices/%s",
		*project, *region, *registry, *device)

	fmt.Println("Client ID:", client_id)

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
		AddBroker(*server).
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
	topic := fmt.Sprintf("/devices/%s/%s", *device, topicType)

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
