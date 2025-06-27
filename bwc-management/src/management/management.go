// BWC Management manages the BWC agents on the device. It receives project
// change commands from the cloud to reload agents with the new project configurations.
// BWC Management is responsible solely for the functionality of reloading agents
// in response to project changes.
package management

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"

	sdtType "main/src/managementType"
)

// Global variables used in the BWC Management package:
// - pjCode: Project code of the device.
// - assetCode: Serial number of the device.
// - cli: MQTT Client type variable, representing the connected MQTT server's client.
// - mqttUser: User ID used for MQTT connection.
// - mqttPassword: Password used for MQTT connection.
// - procLog: Struct defining the format of logs.
// - systemArch: Architecture of the device.
// - rootPath: Root path of BWC.
// - mqType: Type of MQTT service used by BWC.
var (
	pjCode       string
	assetCode    string
	serviceCode  string
	serviceType  string
	deviceType   string
	cli          mqttCli.Client
	mqttUser     = "sdt"
	mqttPassword = "251327"
	procLog      sdtType.Logger
	systemArch   string
	rootPath     string
	mqType       string
)

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// 사용안함
func rollback(err error) {
	procLog.Warn.Printf("[Rollback] Errors: %v\n", err)
	// projectChange("")
	// processRestart()

	// if token := cli.Unsubscribe(newTopic); token.Wait() && token.Error() != nil {
	// 	procLog.Error.Println("[Project] Error unsubscribing from the current topic:", token.Error())
	// }
	// changeSubscription(m.ProjectCode, assetCode)

	procLog.Warn.Printf("[Rollback] Config was rollback.\n")
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
	opts.SetClientID(fmt.Sprintf("blokworks-client-bwcmanagement-%s", cilentUUID))
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
//   - config: Struct storing the Config file saved on the device in JSON format.
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
//   - pjCode: The project code to which the device belongs.
//   - config: Struct storing the Config file saved on the device in JSON format.
func sendDataEdgeMqtt(
	payload map[string]interface{}, // Result of command
	pjCode string, // Information of config
	assetCode string,
) {
	topic := fmt.Sprintf("%s/%s/%s/bwc/register-project/response", serviceCode, pjCode, assetCode)

	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT Unmarshal error: %v\n", err)
	}
	pub_token := cli.Publish(topic, 0, false, resultBody)

	if pub_token.Wait() && pub_token.Error() != nil {
		procLog.Error.Printf("[MQTT] Error: %v\n", pub_token.Error())
	}
}

// ProjectChange function changes the ProjectCode value in the BWC Config file
// to the specified project code.
//
// Input:
//   - projectCode: The new project code to set.
//   - dir: The BWC Root Path.
//
// Output:
//   - error: An error message string.
func ProjectChange(projectCode string, dir string) error {
	// yaml read
	targetFile := fmt.Sprintf("%s/device.config/config.json", dir)
	jsonFile, err := ioutil.ReadFile(targetFile)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Not found file Error: %v\n", err)
		return err
	}

	var jsonData map[string]interface{}
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Unmarshal Error: %v\n", err)
		return err
	}

	jsonData["projectcode"] = projectCode

	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
	err = ioutil.WriteFile(targetFile, saveJson, 0644)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Save yaml file Error: %v\n", err)
		return err
	}

	return err
}

// ProcessRestart function restarts the BWC Agents. When the project value changes,
// the BWC Agents need to be restarted to operate with the updated project value.
//
// Output:
//   - error: An error message string.
func ProcessRestart() error {
	var svcList []string
	var cmd_err error
	if systemArch == "win" {
		svcList = []string{"SDTCloud DeviceControl", "SDTCloud DeviceHealth", "SDTCloud DeviceHeartbeat", "SDTCloud ProcessChecker"}
	} else if deviceType == "aquarack" {
		svcList = []string{"device-control", "device-health", "device-heartbeat", "process-checker", "aquarack-data-collector"}
	} else {
		svcList = []string{"device-control", "device-health", "device-heartbeat", "process-checker"}
	}
	// aquarack-data-collector

	for _, svc := range svcList {
		if systemArch == "win" {
			cmd_run := exec.Command("sc", "stop", svc)
			_, cmd_err = cmd_run.Output()
			cmd_run = exec.Command("sc", "start", svc)
			_, cmd_err = cmd_run.Output()
		} else {
			cmd := fmt.Sprintf("systemctl restart %s", svc)
			cmd_run := exec.Command("sh", "-c", cmd)
			_, cmd_err = cmd_run.Output()
		}
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] %s Restart Error: %v\n", svc, cmd_err)
			return cmd_err
		} else {
			procLog.Info.Printf("[DEPLOY] %s checker Restart Success\n", svc)
		}
	}
	return nil
}

// ProjectCert function downloads a new Cert file when changing the project.
//
// Input:
//   - projectInfo: Struct containing project code and Cert download information.
//   - cntProject: Previous project code value.
//   - dir: BWC Root Path.
//
// Output:
//   - error: String of type Error.
func ProjectCert(projectInfo sdtType.ProjectControl, cntProject string, dir string) error {
	// Onprem pass.
	//if serviceType == "onprem" {
	//	return nil
	//}
	if projectInfo.ProjectCode == "no_project" {
		DeleteCert(cntProject, dir)
	} else {
		priFile := fmt.Sprintf("%s-private.pem", projectInfo.ProjectCode)
		certFile := fmt.Sprintf("%s-certificate.pem", projectInfo.ProjectCode)
		procLog.Info.Printf("[Project] Download: %s\n", priFile)
		fileDownload(dir, priFile, projectInfo.PrivateKey)
		procLog.Info.Printf("[Project] Download: %s\n", certFile)
		fileDownload(dir, certFile, projectInfo.Cert)
	}
	return nil
}

// DeleteCert function deletes the previous Cert file when changing the project. It does not delete the Cert file for the No_Project.
//
// Input:
//   - cntProject: Previous project code value.
//   - dir: BWC Root Path.
//
// Output:
//   - error: String of type Error.
func DeleteCert(cntProject string, dir string) {
	priFile := fmt.Sprintf("%s/cert/%s-private.pem", dir, cntProject)
	certFile := fmt.Sprintf("%s/cert/%s-certificate.pem", dir, cntProject)

	os.Remove(priFile)
	os.Remove(certFile)
}

// fileDownload function downloads a file from the provided Cert URI.
//
// Input:
//   - dir: BWC Root Path.
//   - targetFile: Name of the file to save.
//   - fullURLFile: URI value of the file to download.
//
// Output:
//   - error: String of type Error.
func fileDownload(dir string, targetFile string, fullURLFile string) error {

	fileName := fmt.Sprintf("%s/cert/%s", dir, targetFile)
	file, err := os.Create(fileName)
	if err != nil {
		procLog.Info.Printf("[DEPLOY] fileDownload file creation error: ", err)
		return err
	}

	// Put content on file
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	procLog.Info.Printf("[DEBUG] ", fullURLFile)
	resp, err := client.Get(fullURLFile)
	if err != nil {
		procLog.Info.Printf("[DEPLOY] fileDownload URL get file error: ", err)
		return err
	}
	defer resp.Body.Close()

	if resp.Status[:3] != "200" {
		procLog.Info.Printf("Download error: %s\n", resp.Status)
		return err
	}
	size, err := io.Copy(file, resp.Body)
	procLog.Info.Printf("[INFO]: Downloaded a file %s with size %d\n", fileName, size)

	defer file.Close()
	return err
}

// SubMessage function is executed when MQTT subscribes to a message. This function handles
// the control to modify the BWC Config project code and restart the BWC Agent upon receiving
// a project change message.
//
// Input:
//   - client: MQTT Client variable.
//   - msg: Message variable read from MQTT.
func SubMessage(client mqttCli.Client, msg mqttCli.Message) {
	var dir string
	if systemArch == "win" {
		dir = "C:/sdt"
	} else {
		dir = "/etc/sdt"
	}

	cntTopic := fmt.Sprintf("%s/%s/%s/bwc/register-project/request", serviceCode, pjCode, assetCode)
	procLog.Info.Printf("[Topic] Before Changing, subscription topic to: %s\n", cntTopic)

	var m sdtType.ProjectControl
	procLog.Info.Printf("[MQTT] Get command message: %s\n", string(msg.Payload()))
	err := json.Unmarshal(msg.Payload(), &m)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal Error: %v\n", err)
	} else {
		// checker project 변경
		pjerr := ProjectChange(m.ProjectCode, dir)
		if pjerr != nil {
			procLog.Error.Printf("[Project] Can't change projectcode Error: %v\n", pjerr)
			// rollback(pjerr, assetCode)
		}
		pjerr = ProjectCert(m, pjCode, dir)
		if pjerr != nil {
			procLog.Error.Printf("[Project] Can't change projectcode Error: %v\n", pjerr)
			// rollback(pjerr, assetCode)
		}
		pjerr = ProcessRestart()
		if pjerr != nil {
			procLog.Error.Printf("[Project] Can't change projectcode Error: %v\n", pjerr)
			// rollback(pjerr, assetCode)
		}
		// return result
		resultMsg := checkResult(assetCode, pjerr, pjCode, m.ProjectCode)
		sendDataEdgeMqtt(resultMsg, pjCode, assetCode)

		// change topic
		// Get project Code
		var configData sdtType.ConfigInfo
		jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
		jsonFile, err := ioutil.ReadFile(jsonFilePath)
		if err != nil {
			procLog.Error.Printf("[MAIN] Not found file Error: %v\n", err)
			panic(err)
		}

		err = json.Unmarshal(jsonFile, &configData)
		if err != nil {
			procLog.Error.Printf("[MAIN] Unmarshal Error: %v\n", err)
			panic(err)
		}
		//pjCode = m.ProjectCode
		pjCode = configData.ProjectCode

		// disconnect.
		cli.Disconnect(0)

		// pem setting
		var rootCa string
		if configData.ServerIp == "onprem" {
			rootCa = fmt.Sprintf("%s/cert/rootCa.pem", rootPath)
		} else {
			rootCa = fmt.Sprintf("%s/cert/AmazonRootCA1.pem", rootPath)
		}
		private := fmt.Sprintf("%s/cert/%s-private.pem", rootPath, pjCode)
		fullCertChain := fmt.Sprintf("%s/cert/%s-certificate.pem", rootPath, pjCode)

		// Set aws - Mqtt
		SetMqttClient(mqType, configData, rootCa, fullCertChain, private)
		procLog.Info.Printf("[MQTT] Reconnecting aws mqtt\n")

		ChangeSubscription()
	}
}

// ChangeSubscription function changes the Subscribe Topic value due to project changes in BWC Management.
// When this function is executed, the Topic is initialized with the changed project code, and a new Subscribe
// connection attempt is made.
//
// Input:
//   - client: MQTT Client variable.
//   - msg: Message variable read from MQTT.
func ChangeSubscription() {
	// TODO
	//  - 변경된 Cert 파일로 mqtt client 갱신하도록 수정
	newTopic := fmt.Sprintf("%s/%s/%s/bwc/register-project/request", serviceCode, pjCode, assetCode)
	procLog.Info.Printf("[Topic] Changing subscription new topic to: %s\n", newTopic)

	token := cli.Subscribe(newTopic, 0, SubMessage)

	if token.Wait() && token.Error() != nil {
		fmt.Printf("[MAIN] %v \n", token.Error())
		os.Exit(1)
	}

}

// CheckResult function generates a completion message to be sent to the cloud by BWC Management
// after project change and BWC Agent reload. The message format is as follows:
//
// msg = {"assetCode": "SerialNumber", "status": {"succeed": 0 or 1, "errMsg": string}, "result": {"message": string, "releasedAt": ~~, "updatedAt": ~~}}
//
// Input:
//   - assetCode: Serial number of the device.
//   - errData: Error message for control failure.
//   - priProject: Previous project code before the change.
//   - curProject: Current project code after the change.
//
// Output:
//   - map[string]interface{}: Message to be sent to the cloud.
func checkResult(
	assetcode string, // Edge's name
	// returnMessage string, // Command result
	errData error, // Command error message
	// statusCode int, // Status Code
	priProject string, // Prior Project Code
	curProject string, // Current Project Code

) map[string]interface{} {
	var result map[string]interface{}
	var cmdMsg map[string]interface{}
	var statusBody map[string]interface{}
	var succeed int
	var errMessage string
	var returnMessage string // Command result

	if errData == nil {
		errMessage = ""
		succeed = 1
		returnMessage = fmt.Sprintf("Change %s successed.", curProject)
	} else {
		errMessage = fmt.Sprintf("%v", errData)
		returnMessage = ""
		succeed = 0
	}

	result = map[string]interface{}{
		"message":    returnMessage,
		"releasedAt": int64(time.Now().UTC().Unix() * 1000),
		"updatedAt":  int64(time.Now().UTC().Unix() * 1000),
	}

	statusBody = map[string]interface{}{
		"succeed": succeed,
		// "statusCode":	statusCode,
		"errMsg": errMessage,
		// "errMsg":		http.StatusText(statusCode),
	}

	cmdMsg = map[string]interface{}{
		"assetCode": assetcode,
		"result":    result,
		"status":    statusBody,
		// "requestId":  requestId,
	}

	return cmdMsg
}

// mainOperation is the main function of BWC Management. It selects MQTT broker based on the SDTCloud service type of the device
// and publishes messages accordingly.
//
// Input:
//   - mqttType: Type of SDTCloud service used by the device.
//   - archType: Architecture of the device.
//   - rootPath: Root path for SDTCloud stored on the device.
func RunBody(mqttType string, archType string, sdtPath string) {
	// Set golbal parameter
	rootPath = sdtPath
	systemArch = archType
	var configData sdtType.ConfigInfo
	jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		procLog.Error.Printf("[MAIN] Not found file Error: %v\n", err)
		panic(err)
	}

	err = json.Unmarshal(jsonFile, &configData)
	if err != nil {
		procLog.Error.Printf("[MAIN] Unmarshal Error: %v\n", err)
		panic(err)
	}

	// Set deviceType
	deviceType = configData.DeviceType

	// mqtt Setting key
	var rootCa string
	if configData.ServerIp == "onprem" {
		rootCa = fmt.Sprintf("%s/cert/rootCa.pem", rootPath)
	} else {
		rootCa = fmt.Sprintf("%s/cert/AmazonRootCA1.pem", rootPath)
	}
	private := fmt.Sprintf("%s/cert/%s-private.pem", rootPath, configData.ProjectCode)
	fullCertChain := fmt.Sprintf("%s/cert/%s-certificate.pem", rootPath, configData.ProjectCode)

	// Set topic
	// topic := fmt.Sprintf("sdtcloud/%s/%s/bwc/register-project",configData.ProjectCode, configData.AssetCode)

	// Set system exit
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	// Set Mqtt
	mqType = mqttType
	serviceType = mqttType
	SetMqttClient(mqttType, configData, rootCa, fullCertChain, private)

	defer cli.Disconnect(250)

	// Set subscribe mqtt
	pjCode = configData.ProjectCode
	assetCode = configData.AssetCode
	serviceCode = configData.ServiceCode
	ChangeSubscription()

	select {
	case <-stopchan:
		procLog.Error.Println("[MAIN] Interrupt, exit.")
		break
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
func SetMqttClient(mqttType string, configData sdtType.ConfigInfo, rootCa string, fullCertChain string, private string) {
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
		panic(err)
	}

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}
}
