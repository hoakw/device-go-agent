// The heartbeat package updates the device's heartbeat status to the cloud via MQTT messages.
// Heartbeat establishes an MQTT connection and publishes device status messages at a 10-second interval.
package heartbeat

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	sdtType "main/src/heartbeatType"
)

// Global variables used in the Heartbeat package.
// - cli: MQTT Client type variable representing the connected MQTT server's client.
// - delay: MQTT message publishing interval in seconds.
// - mqttUser: User ID used for MQTT connection.
// - mqttPassword: Password used for MQTT connection.
// - procLog: Struct defining the format of logs.
var (
	cli          mqttCli.Client
	delay        time.Duration = 10
	mqttUser                   = "sdt"
	mqttPassword               = "251327"
	procLog      sdtType.Logger
)

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// This function defines options for connecting to the AWS IoT Core MQTT Broker.
// It specifies options such as TLS, MQTT URI, client name, and handlers.
//
// Input:
//   - mqttURL: The endpoint (URI) of the IoT Core MQTT Broker.
//   - rootCa: Path to the rootCa PEM file.
//   - fullCertChain: Path to the fullCertChain PEM file.
//   - clientKey: Path to the clientKey PEM file.
//
// Output:
//   - *mqttCli.ClientOptions: Variable of type MQTT clientOptions.
func createAwsClientOptions(mqttURL, rootCa, fullCertChain, clientKey string) *mqttCli.ClientOptions {
	// Load CA certificate
	caCert, err := ioutil.ReadFile(rootCa)
	if err != nil {
		log.Fatalf("Error reading CA certificate file: %v", err)
	}

	fullcert, err := tls.LoadX509KeyPair(fullCertChain, clientKey)
	if err != nil {
		log.Fatalf("Error reading client key file: %v", err)
	}

	fullcert.Leaf, err = x509.ParseCertificate(fullcert.Certificate[0])
	if err != nil {
		log.Fatalf("Error Parse Certificate: %v", err)
	}

	// Create certificate pool with CA certificate
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(caCert)

	// Create TLS configuration
	tlsConfig := &tls.Config{
		RootCAs:            roots,
		Certificates:       []tls.Certificate{fullcert},
		InsecureSkipVerify: true,
	}

	// set uuid
	newUUID := uuid.New()
	cilentUUID := newUUID.String()

	opts := mqttCli.NewClientOptions()
	opts.AddBroker(mqttURL)
	opts.SetTLSConfig(tlsConfig)
	opts.SetClientID(fmt.Sprintf("blokworks-client-heartbeat-%s", cilentUUID))
	opts.SetConnectionLostHandler(func(client mqttCli.Client, err error) {
		procLog.Error.Printf("[MQTT] Connection lost: %v\n", err)
		os.Exit(1)
	})

	return opts
}

// This function defines options for connecting to the Mosquitto MQTT Broker.
// It specifies options such as TLS, MQTT URI, client name, and handlers.
//
// Input:
//   - config: Struct storing the config file saved on the device in JSON format.
//
// Output:
//   - mqttCli.Client: Variable of type MQTT Client.
func connectToMqtt(
	config sdtType.ConfigInfo, // Information of config
) mqttCli.Client {
	procLog.Info.Printf("[HEARTBEAT] In connectToMqtt Function \n")
	opts := mqttCli.NewClientOptions()
	opts.AddBroker(config.MqttUrl)
	opts.SetPassword(mqttPassword)
	opts.SetUsername(mqttUser)
	opts.SetClientID(fmt.Sprintf("blokworks-client-heartbeat-%s", config.AssetCode))
	opts.SetConnectionLostHandler(func(client mqttCli.Client, err error) {
		procLog.Error.Printf("[MQTT] Connection lost: %v\n", err)
		os.Exit(1)
	})

	cli = mqttCli.NewClient(opts)

	token := cli.Connect()
	if token.Wait() && token.Error() != nil {
		procLog.Error.Printf("[MQTT] Error: %v\n", token.Error())
		os.Exit(1)
	}

	return cli
}

// This function publishes a message to the MQTT Broker.
//
// Input:
//   - payload: Message content to publish, of type interface{} which is a map variable.
//   - config: Struct storing the config file saved on the device in JSON format.
func sendDataEdgeMqtt(
	payload map[string]interface{}, // Result of command
	config sdtType.ConfigInfo, // Information of config
) {
	// topic := fmt.Sprintf("$aws/things/sdt-cloud-development/shadow/name/device-heartbeat/%s", config.AssetCode)
	topic := fmt.Sprintf("%s/%s/%s/bwc/heartbeat", config.ServiceCode, config.ProjectCode, config.AssetCode)

	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal error: %v\n", err)
	}
	pub_token := cli.Publish(topic, 0, false, resultBody)

	if pub_token.Wait() && pub_token.Error() != nil {
		procLog.Error.Printf("[MQTT] Error: %v\n", pub_token.Error())
	}
}

// This function is the main operation function of Heartbeat. It selects the MQTT
// broker based on the SDT Cloud service type of the device and publishes messages.
// The message is defined as follows:
//
//	Payload = {"timestamp": 1858182312, "data": {"heartbeat": "OK"}}
//
// Input:
//
//   - mqttType: SDTCloud service type that the device uses.
//   - archType: Architecture of the device.
//   - rootPath: Root path of SDTCloud stored on the device.
func RunBody(mqttType string, archType string, rootPath string) {
	var configData sdtType.ConfigInfo

	jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		procLog.Error.Printf("[HEARTBEAT] Not found file Error: %v\n", err)
	}

	err = json.Unmarshal(jsonFile, &configData)
	if err != nil {
		procLog.Error.Printf("[HEARTBEAT] Unmarshal Error: %v\n", err)
	}

	// mqtt Setting key
	var rootCa string
	if configData.ServerIp == "onprem" {
		rootCa = fmt.Sprintf("%s/cert/rootCa.pem", rootPath)
	} else {
		rootCa = fmt.Sprintf("%s/cert/AmazonRootCA1.pem", rootPath)
	}
	private := fmt.Sprintf("%s/cert/%s-private.pem", rootPath, configData.ProjectCode)
	fullCertChain := fmt.Sprintf("%s/cert/%s-certificate.pem", rootPath, configData.ProjectCode)

	// Set Mqtt
	if mqttType == "onprem" {
		// Set mqtt client - EC2
		cli = connectToMqtt(configData)
	} else if mqttType == "aws-dev" || mqttType == "eks" || mqttType == "dev" {
		// Set MQTT - AWS IoT Core
		opts := createAwsClientOptions(configData.MqttUrl, rootCa, fullCertChain, private)
		cli = mqttCli.NewClient(opts)
	} else {
		err = errors.New("Please input mqtt variable.")
		procLog.Error.Printf("[MAIN] Docker connection Error: %v\n", err)
		panic(err)
	}

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	defer cli.Disconnect(250)

	for true {
		heartBeat := map[string]interface{}{
			"heartbeat": "OK",
		}

		msg := map[string]interface{}{
			"timestamp": int64(time.Now().UTC().Unix() * 1000),
			"data":      heartBeat,
		}

		sendDataEdgeMqtt(msg, configData)

		time.Sleep(delay * time.Second)
	}
}
