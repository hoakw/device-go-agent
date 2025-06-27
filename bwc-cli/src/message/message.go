// The message package handles creating and sending messages to SDT Cloud
// regarding commands processed by BWC-CLI.
package message

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	sdtType "main/src/cliType"
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
		procLog.Error.Printf("Error reading CA certificate file: %v\n", err)
	}

	fullcert, err := tls.LoadX509KeyPair(fullCertChain, clientKey)
	if err != nil {
		procLog.Error.Printf("Error reading client key file: %v\n", err)
	}

	fullcert.Leaf, err = x509.ParseCertificate(fullcert.Certificate[0])
	if err != nil {
		procLog.Error.Printf("Error Parse Certificate: %v\n", err)
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
	opts.SetClientID(fmt.Sprintf("blokworks-client-bwc-cli-%s", cilentUUID))

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
	procLog.Info.Printf("[MQTT] In connectToMqtt Function")
	opts := mqttCli.NewClientOptions()
	opts.AddBroker(config.MqttUrl)
	opts.SetPassword(mqttPassword)
	opts.SetUsername(mqttUser)
	opts.SetClientID(fmt.Sprintf("blokworks-client-bwc-management-%s", config.AssetCode))
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
//   - config: Struct storing the Config file saved on the device in JSON format.
func sendDataEdgeMqtt(
	payload map[string]interface{}, // Result of command
	configData sdtType.ConfigInfo,
) {
	topic := fmt.Sprintf("%s/%s/%s/bwc/control/self-deploy", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)

	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("Unmarshal error: %v\n", err)
	}
	pub_token := cli.Publish(topic, 0, false, resultBody)

	if pub_token.Wait() && pub_token.Error() != nil {
		procLog.Error.Printf("Send to mqtt error: %v\n", pub_token.Error())
	}
}

// CheckResult function generates a result message after executing a control command to be sent to the cloud.
// Below is the format of the message:
//
//	msg = {
//		"assetCode": "SerialNumber",
//		"status": {
//			"succeed": 0 or 1,
//			"statusCode": int
//			"errMsg": string
//		},
//		"result": {
//			"command": string,
//			"subCommand": string,
//			"pid": int,
//			"size": int,
//			"binFile":     binFile,
//			"requirement": requirementStr,
//			"appName": string,
//			"venvName": string,
//			"appId": string,
//			"message": string,
//			"releasedAt": int64,
//			"updatedAt": int64,
//			"appRepoPath": string
//		},
//		"requestId": string
//	}
//
// Input:
//   - rootPath: The root path of BWC.
//   - configData: BWC Config information struct.
//   - appName: Name of the app.
//   - returnMessage: Result message of the control command.
//   - errData: Error message of the control command.
//   - statusCode: Status code of the control command.
//   - subCommand: Sub-command of the control command.
//   - command: Main command of the control command.
//   - requestId: ID of the control command.
//   - pid: PID of the app.
//   - fileSize: Size of the app.
//   - jsonData: Config values of the app.
//   - appRepoPath: Path of the app in the code repository.
//   - appId: ID of the app.
//   - binFile: Bin(Runtime) file of the app.
//   - requirementStr: Installed package information of the virtual environment.
//   - venvName: Name of the virtual environment for the app (relevant only for python3 apps).
func SendResult(
	rootPath string,
	configData sdtType.ConfigInfo,
	appName string, // Local app Name
	returnMessage string, // Command result
	errData error, // Command error message
	statusCode int, // Status Code
	subCommand string, // Value of subcommand
	command string, // Value of command
	requestId string, // Requtest ID
	pid int, // Process PID
	fileSize int64, // Size of file
	jsonData map[string]interface{}, // Data of config(json, yaml, etc..)
	appRepoPath string, // app gitea repo path
	appId string,
	binFile string,
	requirementStr string,
	venvName string,
) {
	var result map[string]interface{}
	var cmdMsg map[string]interface{}
	var statusBody map[string]interface{}
	var succeed int
	var errMessage string

	// Set mqtt
	// mqtt Setting key
	var rootCa string
	if configData.ServiceType == "onprem" {
		rootCa = fmt.Sprintf("%s/cert/rootCa.pem", rootPath)
	} else {
		rootCa = fmt.Sprintf("%s/cert/AmazonRootCA1.pem", rootPath)
	}
	private := fmt.Sprintf("%s/cert/%s-private.pem", rootPath, configData.ProjectCode)
	fullCertChain := fmt.Sprintf("%s/cert/%s-certificate.pem", rootPath, configData.ProjectCode)

	if configData.ServiceType == "onprem" {
		cli = connectToMqtt(configData)
	} else if configData.ServiceType == "aws-dev" || configData.ServiceType == "eks" || configData.ServiceType == "dev" {
		opts := createAwsClientOptions(configData.MqttUrl, rootCa, fullCertChain, private)
		cli = mqttCli.NewClient(opts)
	} else {
		fmt.Printf("Please check 'servicetype' variable in BWC config file. \n")
		os.Exit(1)
	}

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		procLog.Error.Printf("Failed to connect to MQTT broker: %v\n", token.Error())
	}

	defer cli.Disconnect(250)

	if errData == nil {
		errMessage = ""
		succeed = 1
		// returnMessage 	= fmt.Sprintf("%s/%s successed.", command, subCommand)
	} else {
		errMessage = fmt.Sprintf("%v", errData)
		// returnMessage 	= fmt.Sprintf("%s/%s failed", command, subCommand)
		succeed = 0
	}

	if jsonData == nil {
		result = map[string]interface{}{
			"command":     command,
			"subCommand":  subCommand,
			"pid":         pid,
			"size":        fileSize,
			"binFile":     binFile,
			"requirement": requirementStr,
			"venvName":    venvName,
			"appName":     appName,
			"appId":       appId,
			"message":     returnMessage,
			"releasedAt":  int64(time.Now().UTC().Unix() * 1000),
			"updatedAt":   int64(time.Now().UTC().Unix() * 1000),
			"appRepoPath": appRepoPath,
		}
	} else {
		result = map[string]interface{}{
			"command":     command,
			"subCommand":  subCommand,
			"pid":         pid,
			"size":        fileSize,
			"binFile":     binFile,
			"requirement": requirementStr,
			"venvName":    venvName,
			"appName":     appName,
			"appId":       appId,
			"message":     jsonData,
			"releasedAt":  int64(time.Now().UTC().Unix() * 1000),
			"updatedAt":   int64(time.Now().UTC().Unix() * 1000),
			"appRepoPath": appRepoPath,
		}
	}

	statusBody = map[string]interface{}{
		"succeed":    succeed,
		"statusCode": statusCode,
		"errMsg":     errMessage,
		// "errMsg":		http.StatusText(statusCode),
	}

	cmdMsg = map[string]interface{}{
		"assetCode": configData.AssetCode,
		"result":    result,
		"status":    statusBody,
		"requestId": requestId,
	}

	sendDataEdgeMqtt(cmdMsg, configData)
}
