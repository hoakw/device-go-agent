// The message package handles creating and sending messages to SDT Cloud.
package message

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"io/ioutil"
	sdtType "main/src/controlType"
	"os"
	"time"
)

// Global variables used in the message package:
//   - cli: MQTT client variable of type Client, representing the client connected to the MQTT server.
//   - procLog: Struct defining the format of logs.
//   - mqttUser: User ID used for MQTT connection.
//   - mqttPassword: Password used for MQTT connection.
var (
	cli          mqttCli.Client
	procLog      sdtType.Logger
	mqttUser     = "sdt"
	mqttPassword = "251327"
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
//   - *mqttCli.ClientOptions: Variable of type MQTT ClientOptions.
func createAwsClientOptions(mqttURL, rootCa, fullCertChain, clientKey string) *mqttCli.ClientOptions {
	// Load CA certificate
	caCert, err := ioutil.ReadFile(rootCa)
	if err != nil {
		procLog.Error.Printf("Error reading CA certificate file: %v", err)
	}

	fullcert, err := tls.LoadX509KeyPair(fullCertChain, clientKey)
	if err != nil {
		procLog.Error.Printf("Error reading client key file: %v", err)
	}

	fullcert.Leaf, err = x509.ParseCertificate(fullcert.Certificate[0])
	if err != nil {
		procLog.Error.Printf("Error Parse Certificate: %v", err)
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
	opts.SetClientID(fmt.Sprintf("blokworks-client-live-control-%s", cilentUUID))
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
	config sdtType.ConfigInfo, // The variable of config
) mqttCli.Client {
	procLog.Info.Println("[MQTT] In connectToMqtt Function")
	opts := mqttCli.NewClientOptions()
	opts.AddBroker(config.MqttUrl)
	opts.SetPassword(mqttPassword)
	opts.SetUsername(mqttUser)
	opts.SetClientID(fmt.Sprintf("blokworks-client-control-%s", config.AssetCode))
	opts.SetConnectionLostHandler(func(client mqttCli.Client, err error) {
		procLog.Error.Printf("[MQTT] Connection lost: %v\n", err)
		os.Exit(1)
	})

	cli = mqttCli.NewClient(opts)

	token := cli.Connect()
	if token.Wait() && token.Error() != nil {
		procLog.Error.Printf("[MAIN] Error %v \n", token.Error())
		os.Exit(1)
	}

	return cli
}

// This function publishes a message to the MQTT Broker.
//
// Input:
//   - payload: Message content to publish, of type interface{} which is a map variable.
//   - config: Struct storing the config file saved on the device in JSON format.
func SendDataEdgeMqtt(
	payload sdtType.ResultMsg, // The variable of command
	topic string,
	cli mqttCli.Client,
) {
	// topic := fmt.Sprintf("sdtcloud/%s/%s/bwc/control/response", projectCode, cmd_info["assetCode"])
	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal error: %v\n", err)
	} else {
		pub_token := cli.Publish(topic, 0, false, resultBody)
		procLog.Info.Printf("[MQTT] Send message: Topic: %s\nMessage:%s\n", topic, resultBody)
		pub_token.WaitTimeout(time.Second * 5)
		//if pub_token.Wait() && pub_token.Error() != nil {
		if pub_token.Error() != nil {
			procLog.Error.Printf("[MAIN] Error %v \n", pub_token.Error())
			os.Exit(1)
		}
	}
}

// This function publishes a message to the MQTT Broker.
//
// Input:
//   - payload: Message content to publish, of type interface{} which is a map variable.
//   - config: Struct storing the config file saved on the device in JSON format.
func SendDataEdgeMqttInterface(
	payload map[string]interface{}, // The variable of command
	topic string,
	cli mqttCli.Client,
) {
	// topic := fmt.Sprintf("sdtcloud/%s/%s/bwc/control/response", projectCode, cmd_info["assetCode"])
	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal error: %v\n", err)
	} else {
		pub_token := cli.Publish(topic, 0, false, resultBody)
		procLog.Info.Printf("[MQTT] Send message: Topic: %s\nMessage:%s\n", topic, resultBody)
		if pub_token.Wait() && pub_token.Error() != nil {
			procLog.Error.Printf("[MAIN] Error %v \n", pub_token.Error())
			os.Exit(1)
		}
	}
}

// SetMqttClient selects an MQTT broker based on the SDTCloud service type of the device and publishes messages accordingly.
//
// Input:
//   - mqttType: Type of SDTCloud service used by the device.
//   - configData: Device's BWC Config Struct.
//   - rootCa: Path to the rootCa file.
//   - fullCertChain: Path to the fullCertChain file.
//   - private: Path to the private file.
func SetMqttClient(mqttType string, configData sdtType.ConfigInfo, rootCa string, fullCertChain string, private string) mqttCli.Client {
	if mqttType == "onprem" {
		// Set mqtt client - EC2
		cli = connectToMqtt(configData)
	} else if mqttType == "aws-dev" || mqttType == "eks" || mqttType == "dev" {
		// Set MQTT - AWS IoT Core
		opts := createAwsClientOptions(configData.MqttUrl, rootCa, fullCertChain, private)
		cli = mqttCli.NewClient(opts)
	} else {
		err := errors.New("Please input mqtt variable.")
		procLog.Error.Printf("[MQTT] MQTT connection Error: %v\n", err)
		os.Exit(1)
	}

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		procLog.Error.Printf("Failed to connect to MQTT broker: %v\n", token.Error())
		os.Exit(1)
	}

	return cli
}

func SendAckMessage(
	payload map[string]interface{}, // The variable of command
	cli mqttCli.Client,
	projectCode string,
	assetCode string,
	serviceCode string,
) {
	topic := fmt.Sprintf("%s/%s/%s/bwc/control/request/ack", serviceCode, projectCode, assetCode)
	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal error: %v\n", err)
	} else {
		pub_token := cli.Publish(topic, 0, false, resultBody)
		procLog.Info.Printf("[MQTT] Send message: Topic: %s\nMessage:%s\n", topic, resultBody)
		if pub_token.Wait() && pub_token.Error() != nil {
			procLog.Error.Printf("[MAIN] Error %v \n", pub_token.Error())
			os.Exit(1)
		}
	}
}
