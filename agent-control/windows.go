//go:build windows
// +build windows

// This package is the main package for Device-Control. Device-Control handles device management tasks..
// Device management includes controlling devices such as device rebooting and application management
// by receiving control messages from SDT Cloud and executing control commands. After processing,
// it sends the success or failure of the control to SDT Cloud.
//
// Device control functionalities include:
//   - Device terminal commands
//   - Device application deployment, deletion, execution, and termination
//   - Device configuration modification
//   - Device rebooting
//
// Supported server architectures for this package are Windows and Linux.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	winSvc "golang.org/x/sys/windows/svc"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	dockerCli "github.com/docker/docker/client"
	mqttCli "github.com/eclipse/paho.mqtt.golang"
	sdtConfig "main/src/config"
	sdtControl "main/src/control"
	sdtType "main/src/controlType"
	sdtDeploy "main/src/deploy"
	sdtDocker "main/src/docker"
	sdtMessage "main/src/message"
	sdtModel "main/src/model"
)

// Global variables used in the Device-Control package.
// - cli: MQTT Client type variable for the connected MQTT server client.
// - dockerClient: MQTT message publishing interval in seconds.
// - configData: BWC Config Struct.
// - svcInfo: Device control information Struct.
// - mqttUser: User ID used for MQTT connection.
// - mqttPassword: Password used for MQTT connection.
// - procLog: Struct defining the format of logs.
// - systemArch: Architecture of the device.
// - systemHome: Hostname of the device.
var (
	cli                      mqttCli.Client
	dockerClient             *dockerCli.Client
	configData               sdtType.ConfigInfo
	svcInfo                  sdtType.ControlService
	mqttUser                 = "sdt"
	mqttPassword             = "251327"
	procLog                  sdtType.Logger
	systemArch               = ""
	systemHome               = ""
	DEVICE_CONTROL_VERSION   = "4.0"
	DEVICE_HEALTH_VERSION    = "4.0"
	DEVICE_HEARTBEAT_VERSION = "4.0"
	PROCESS_CHECKER_VERSION  = "4.0"
	BWC_MANAGEMENT_VERSION   = "4.0"
)

type winControlService struct {
	ServiceName string
}

// SubMessage is a function executed when MQTT subscribes to a message.
// This function handles control command messages received from the cloud.
// Input:
//   - client: MQTT Client variable.
//   - msg: Message variable read from MQTT.
func SubMessage(client mqttCli.Client, msg mqttCli.Message) {
	var topic string
	m := sdtType.CmdControl{}
	procLog.Info.Printf("[MQTT] Get command message: %s\n", string(msg.Payload()))
	err := json.Unmarshal(msg.Payload(), &m)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal Error: %v\n", err)
		procLog.Error.Printf("[MQTT] Cannot send message. \n")
	} else {
		var result, configResult sdtType.ResultMsg

		// Ack 메시지를 보낸 후, 명령어 처리
		ackMsg := map[string]interface{}{
			"requestId": m.RequestId,
			"message":   "ok",
		}
		sdtMessage.SendAckMessage(ackMsg, cli, configData.ProjectCode, configData.AssetCode, configData.ServiceCode)
		procLog.Info.Printf("[MQTT] Send ACK message. \n")

		result, configResult = sdtControl.Control(svcInfo, configData, m, dockerClient, systemArch, systemHome, cli)

		// MQTT Pub
		topic = fmt.Sprintf("%s/%s/%s/bwc/control/response", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
		// DEBUG
		//topic = fmt.Sprintf("%s/%s/%s/debug", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)

		// 명령어 결과를 보냄
		if result.Result == nil {
			sdtMessage.SendDataEdgeMqtt(result, topic, cli)
		} else if result.Result.SubCommand != "reboot" {
			sdtMessage.SendDataEdgeMqtt(result, topic, cli)
		}

		// Config 정보를 보냄
		if configResult.AssetCode != "" {
			sdtMessage.SendDataEdgeMqtt(configResult, topic, cli)
		}

	}
}

// initError defines and initializes the log format. The log formats are defined as Info,
// Warn, and Error, and the output is done using Printf. If you call the function to define and
// initialize the log formats, you can use it to output logs as follows:
// The way to log the text "Hello World" as an Info log type is shown below.
// procLog.Info.Printf("Hello World\n")
// Output: [INFO] Hello World
func initError(logFile io.Writer) {
	procLog.Info = log.New(logFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Warn = log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Error = log.New(logFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// This function is the main operation function of device control. It selects the MQTT broker
// based on the SDTCloud service type of the device and publishes messages. The default behavior
// of RunBody includes handling reboot control and installing SDT Cloud base packages.
// Reboot control involves receiving reboot commands from SDT Cloud, rebooting the device, and
// sending a message to SDT Cloud once the boot is complete. Installing SDT Cloud base packages
// installs python packages for MQTT and S3 when using apps that utilize python.
func RunBody() {
	// Set parameter
	mqttType := svcInfo.MqttType
	rootPath := svcInfo.RootPath
	minicondaPath := svcInfo.MinicondaPath
	commonPythonPath := svcInfo.CommonPythonPath
	//appPath := svcInfo.AppPath

	// var configData sdtType.ConfigInfo
	jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		procLog.Error.Printf("[MAIN] %d Not found file Error: %v\n", err, time.Now())
		panic(err)
	}

	err = json.Unmarshal(jsonFile, &configData)
	if err != nil {
		procLog.Error.Printf("[MAIN] Unmarshal Error: %v\n", err)
		panic(err)
	}

	// Set Variable
	if configData.ServiceType == "aws-dev" {
		svcInfo.SdtcloudIP = "43.200.53.170"
		svcInfo.GiteaPort = 32421
		svcInfo.MinioURL = "43.200.53.170:31191"
	} else if configData.ServiceType == "dev" {
		svcInfo.SdtcloudIP = "192.168.1.162"
		svcInfo.GiteaPort = 32421
		svcInfo.MinioURL = "192.168.1.162:31191"
	} else if configData.ServiceType == "eks" {
		// svcInfo.SdtcloudIP = "af95e343669454d61b2763f9e1d1159c-184121212.ap-northeast-2.elb.amazonaws.com"
		svcInfo.SdtcloudIP = "cloud-repo.sdt.services"
		svcInfo.GiteaPort = 80
		svcInfo.MinioURL = "s3"
	} else if configData.ServiceType == "onprem" {
		svcInfo.SdtcloudIP = configData.ServerIp
		svcInfo.GiteaPort = 32421
		svcInfo.MinioURL = fmt.Sprintf("%s:31191", configData.ServerIp)
	} else {
		procLog.Error.Printf("%s not supported. Please check your service.\n", configData.ServiceType)
	}

	// Set username
	if systemHome == "" {
		procLog.Error.Printf("[MAIN] Please input home variable.\n")
		os.Exit(1)
	}
	procLog.Info.Printf("[MAIN] Your username is %s.\n", systemHome)

	// mqtt Setting key
	var rootCa string
	if configData.ServerIp == "onprem" {
		rootCa = fmt.Sprintf("%s/cert/rootCa.pem", rootPath)
	} else {
		rootCa = fmt.Sprintf("%s/cert/AmazonRootCA1.pem", rootPath)
	}
	private := fmt.Sprintf("%s/cert/%s-private.pem", rootPath, configData.ProjectCode)
	fullCertChain := fmt.Sprintf("%s/cert/%s-certificate.pem", rootPath, configData.ProjectCode)

	// Set system exit
	stopchan := make(chan os.Signal)
	signal.Notify(stopchan, syscall.SIGINT, syscall.SIGKILL)
	defer close(stopchan)

	// Set cocker client
	dockerClient, err = dockerCli.NewClientWithOpts(dockerCli.FromEnv)
	if err != nil {
		procLog.Error.Printf("[MAIN] Docker connection Error: %v\n", err)
		panic(err)
	}
	defer dockerClient.Close()

	// Set Mqtt
	cli = sdtMessage.SetMqttClient(mqttType, configData, rootCa, fullCertChain, private)
	defer cli.Disconnect(250)

	//reboot check!!
	if configData.Reboot == "rebooting" {
		procLog.Info.Printf("[REBOOT] Completed reboot. \n")

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult("bash", "reboot", "reboot completed.")

		cmdStatus := sdtType.NewCmdStatus(http.StatusOK)

		cmdStatus.ErrMsg = ""
		cmdStatus.Succeed = 1

		rebootMsg := sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: configData.RequestId,
		}

		rebootTopic := fmt.Sprintf("%s/%s/%s/bwc/control/response", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
		sdtMessage.SendDataEdgeMqtt(rebootMsg, rebootTopic, cli)

		// 메시지 보낸 후, 상태값 변경
		sdtConfig.Rebooting("completed", "")
	}

	// Send python env info
	// get miniconda
	var runtimeList []string
	var runtimeInfo, runtimeCmd string
	if svcInfo.ArchType == "win" {
		runtimeCmd = fmt.Sprintf("%s/../python", minicondaPath)
	} else {
		runtimeCmd = fmt.Sprintf("%s/python", minicondaPath)
	}
	cmd := exec.Command(runtimeCmd, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		procLog.Warn.Printf("[MAIN] Not found: Miniconda3 python3(%v)\n", err)
	} else {
		runtimeInfo = string(output)
		runtimeInfo = strings.ReplaceAll(runtimeInfo, " ", "-")
		runtimeInfo = strings.ReplaceAll(runtimeInfo, "\n", "")
		runtimeInfo = fmt.Sprintf("Miniconda3-%s", runtimeInfo)
		runtimeList = append(runtimeList, runtimeInfo)
	}

	// get base python
	runtimeCmd = fmt.Sprintf("%s/python", commonPythonPath)
	cmd = exec.Command(runtimeCmd, "--version")
	output, err = cmd.CombinedOutput()
	if err != nil {
		procLog.Warn.Printf("[MAIN] Not found: Base python3(%v)\n", err)
		procLog.Warn.Printf("[MAIN] Since python3 is not found, some function will not work.\n")
		// TODO
		//  - Python3이 없으면 에이전트가 실행되지 않도록 수정해야 함
	} else {
		runtimeInfo = string(output)
		runtimeInfo = strings.ReplaceAll(runtimeInfo, " ", "-")
		runtimeInfo = strings.ReplaceAll(runtimeInfo, "\n", "")
		runtimeList = append(runtimeList, runtimeInfo)
	}

	// Set result about runtime
	runtimeResult := map[string]interface{}{
		"python": runtimeList,
		"bwcVersion": map[string]interface{}{
			"device-control":  DEVICE_CONTROL_VERSION,
			"device-health":   DEVICE_HEALTH_VERSION,
			"process-checker": DEVICE_HEARTBEAT_VERSION,
			"bwc-management":  PROCESS_CHECKER_VERSION,
			"heartbeat":       BWC_MANAGEMENT_VERSION, // 삭제 예정.
		},
	}

	procLog.Info.Printf("[MAIN] Send runtime info: %s\n", runtimeList)
	runtimeTopic := fmt.Sprintf("%s/%s/%s/bwc/control/runtime", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
	sdtMessage.SendDataEdgeMqttInterface(runtimeResult, runtimeTopic, cli)

	// Create Base Venv
	sdtDeploy.CreateBaseVenv(systemHome, svcInfo)
	sdtDeploy.InstallDefaultPkg("base", configData.DeviceType, configData.ServiceType, svcInfo)

	// Set subscribe mqtt
	topic := fmt.Sprintf("%s/%s/%s/bwc/control/request", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
	token := cli.Subscribe(topic, 0, SubMessage)

	if token.Wait() && token.Error() != nil {
		procLog.Error.Printf("[MAIN] %v \n", token.Error())
		os.Exit(1)
	}

	select {
	case <-stopchan:
		procLog.Error.Println("[MAIN] Interrupt, exit.")
		break
	}
}

// This function receives server architecture information, configures the environment
// accordingly, and executes core functions.
//
// Input:
//   - mqtt: Type of MQTT service, which can be aws (AWS IoT Core) or mosq (Mosquitto).
//     -- aws: AWS IoT Core
//     -- mosq: Mosquitto
//   - - exmq: EXMQ
//   - arch: Architecture of the device.
//   - home: Hostname of the device.
func main() {
	// Set parameter
	var mqttType, archType, rootPath, minicondaPath, commonPythonPath, appPath, venvPath, home string
	var baseCmd [2]string
	flag.StringVar(&mqttType, "mqtt", "", "Please input mqtt type(mosq? or aws? or exmq?)")
	flag.StringVar(&archType, "arch", "", "Please input architecture type(amd? or arm? or win?)")
	flag.StringVar(&home, "home", "", "Please input home's name.")
	flag.Parse()
	systemArch = archType
	systemHome = home

	// Set Config PATH
	if archType == "win" {
		rootPath = "C:/sdt"
		minicondaPath = fmt.Sprintf("C:/Users/%s/miniconda3/Scripts", systemHome) //C:\sdt\venv\test\Scripts
		commonPythonPath = fmt.Sprintf("C:/Users/%s/AppData/Local/Programs/Python/Python39", systemHome)
		appPath = "C:/sdt/app"
		venvPath = "C:/sdt/venv"
		baseCmd[0] = "cmd"
		baseCmd[1] = "/c"
	} else {
		rootPath = "/etc/sdt"
		minicondaPath = fmt.Sprintf("/home/%s/miniconda3/bin", systemHome)
		commonPythonPath = "/usr/bin"
		appPath = "/usr/local/sdt/app"
		venvPath = "/etc/sdt/venv"
		baseCmd[0] = "sh"
		baseCmd[1] = "-c"
	}

	// Set Service Variable
	svcInfo.MqttType = mqttType
	svcInfo.ArchType = archType
	svcInfo.RootPath = rootPath
	svcInfo.MinicondaPath = minicondaPath
	svcInfo.CommonPythonPath = commonPythonPath
	svcInfo.AppPath = appPath
	svcInfo.VenvPath = venvPath
	svcInfo.BaseCmd = baseCmd

	// Set logger
	logFilePath := fmt.Sprintf("%s/device.logs/device-control.log", rootPath)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	initError(logFile)

	sdtDeploy.Getlog(procLog)
	sdtConfig.Getlog(procLog)
	sdtControl.Getlog(procLog)
	sdtMessage.Getlog(procLog)
	sdtDocker.Getlog(procLog)
	sdtModel.Getlog(procLog)

	// Run main

	// Set Service Variable
	winSvcInfo := winControlService{
		ServiceName: "DeviceControlService",
	}
	err = winSvc.Run(winSvcInfo.ServiceName, &winSvcInfo)
	//RunBody()

}

// svc.Handler 인터페이스 구현
func (srv *winControlService) Execute(args []string, req <-chan winSvc.ChangeRequest, stat chan<- winSvc.Status) (svcSpecificEC bool, exitCode uint32) {
	stat <- winSvc.Status{State: winSvc.StartPending}

	// 실제 서비스 내용
	procLog.Info.Printf("[SVC] Service Content!!!\n")
	stopChan := make(chan bool, 1)
	go RunBody()

	stat <- winSvc.Status{State: winSvc.Running, Accepts: winSvc.AcceptStop | winSvc.AcceptShutdown}

LOOP:
	for {
		// 서비스 변경 요청에 대해 핸들링
		switch r := <-req; r.Cmd {
		case winSvc.Stop, winSvc.Shutdown:
			stopChan <- true
			procLog.Warn.Printf("[SVC] Service Stop!!!\n")
			break LOOP

		case winSvc.Interrogate:
			procLog.Error.Printf("[SVC] Service Interrogate!!!\n")
			stat <- r.CurrentStatus
			time.Sleep(100 * time.Millisecond)
			stat <- r.CurrentStatus

			//case svc.Pause:
			//case svc.Continue:
		}
	}

	stat <- winSvc.Status{State: winSvc.StopPending}
	return
}
