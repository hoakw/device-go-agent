// The process package publishes MQTT messages to the cloud about resource usage
// of apps deployed on the device. It connects to MQTT and publishes messages
// about the device status every 2 seconds.
package process

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	dockerCli "github.com/docker/docker/client"
	mqttCli "github.com/eclipse/paho.mqtt.golang"

	"github.com/google/uuid"
	"github.com/leizongmin/fuser"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/process"

	sdtType "main/src/processType"
)

// Global variables used in the process package.
// - cli: MQTT Client type variable representing the connected MQTT server's client.
// - delay: MQTT message publishing interval in seconds.
// - mqttUser: User ID used for MQTT connection.
// - mqttPassword: Password used for MQTT connection.
// - procLog: Struct defining the format of logs.
var (
	cli          mqttCli.Client
	mqttUser     = "sdt"
	mqttPassword = "251327"
	//configPath                 = "/etc/sdt/device.config/config.json"
	procLog      sdtType.Logger
	dockerClient *dockerCli.Client
)

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// CalculateCPUPercent function calculates the CPU usage percentage of container.
// The calculation is based on the difference in CPU usage between the current and previous points in time.
// The formula for CPU usage is as follows:
//   - CPU Percent = (cpuDelta/systemDelta) x 100(%)
//
// Input:
//   - stats: This is a JSON-type variable that stores the status information of a container.
//
// Output:
//   - float64: This is the CPU usage percentage of the container.
func CalculateCPUPercent(stats *types.StatsJSON) float64 {
	cpuDelta := float64(stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage)
	systemDelta := float64(stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage)

	if systemDelta > 0.0 && cpuDelta > 0.0 {
		return (cpuDelta / systemDelta) * 100.0 // cpu percent = (cpuDelta / systemDelta) * 100
	}
	return 0.0
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
	opts.SetClientID(fmt.Sprintf("blokworks-client-processchecker-%s", cilentUUID))
	opts.SetConnectionLostHandler(func(client mqttCli.Client, err error) {
		procLog.Error.Printf("[MQTT] Connection lost: %v\n", err)
		os.Exit(1)
	})

	return opts
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
	// topic := fmt.Sprintf("/test", configData.AssetCode)
	// topic := fmt.Sprintf("$aws/things/sdt-cloud-development/shadow/name/device-control/%s/app-health", configData.AssetCode)
	topic := fmt.Sprintf("%s/%s/%s/bwc/apps/health", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)

	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT] Unmarshal error: %v\n", err)
	}
	pub_token := cli.Publish(topic, 0, false, resultBody)

	if pub_token.Wait() && pub_token.Error() != nil {
		procLog.Error.Printf("[MQTT] Error: %v\n", pub_token.Error())
	}
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
	opts.SetClientID(fmt.Sprintf("blokworks-client-process-checker-%s", config.AssetCode))
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

// Contains function checks if a specific integer value exists in a list of integer.
//
// Input:
//   - elems: List of integer.
//   - v: Integer to check for.
//
// Output:
//   - bool: True if integer value exists in the list, false otherwise.
func contains(elems []int, v int) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

// GetProc function collects app process information deployed on the device. The collected information includes:
//   - App Process CPU usage
//   - App Process Memory usage
//
// Input:
//   - targetPid: Slice of integers representing app's PIDs to collect information from.
//
// Output:
//   - int: The usage of CPU(%).
//   - int: The usage of Memory(%).
func GetProc(targetPid []int) (int, int) {
	processes, _ := process.Processes()
	cpuTotal, _ := cpu.Counts(true)
	sumCpu := 0
	sumMem := 0
	for _, p := range processes {
		p_cpu_percent, _ := p.CPUPercent()
		p_mem_percent, _ := p.MemoryPercent()
		p_id := int(p.Pid)

		// if targetPid == p_id {
		if contains(targetPid, p_id) {
			// fmt.Printf("GET Process Resource: %s %f %f\n", p_id, p_cpu_percent, p_mem_percent)
			sumCpu = sumCpu + int(p_cpu_percent)
			sumMem = sumMem + int(p_mem_percent)
		}
	}
	return sumCpu / int(cpuTotal), sumMem
}

// WinGetProc function collects App Process IDs deployed on a windows device.
// The collected information includes:
//   - App Process ID
//
// Input:
//   - appName: Name of the application whose process IDs are to be collected.
//
// Output:
//   - []int: List of process IDs (PID).
func WinGetPid(appName string) []int {
	var pidArr []int
	cmd_run := exec.Command("wmic", "service", "get", "Name,", "ProcessId", "/format:csv")
	stdout1, err := cmd_run.Output()
	if err != nil {
		procLog.Error.Printf("[PROCESS-CHECKER] Get pid error: ", err)
		return pidArr
	}
	strOut := strings.Split(string(stdout1), "\n")
	for _, v := range strOut {
		if strings.Contains(v, appName) {
			pidStr := strings.Trim(strings.Split(v, ",")[2], "\r")
			pid, _ := strconv.Atoi(pidStr)
			pidArr = append(pidArr, pid)
			return pidArr
		}
	}

	procLog.Error.Printf("[PROCESS-CHECKER] Not found process")
	return pidArr

}

// GetPid function collects app process IDs deployed on a linux device.
// The collected information includes:
//   - App Process ID
//
// Input:
//   - appName: Name of the application whose process IDs are to be collected.
//
// Output:
//   - []int: List of process IDs (PID).
func GetPid(appName string) []int {
	getPid := fmt.Sprintf("systemctl show --property MainPID %s.service", appName)
	cmd_run := exec.Command("sh", "-c", getPid)
	stdout1, err := cmd_run.Output()
	var pidArr []int
	if err != nil {
		procLog.Error.Printf("[PROCESS-CHECKER] Get pid error: ", err)
		return pidArr
	}
	strOut := string(stdout1)
	pidStr := strings.Split(strOut[:len(strOut)-1], "=")
	if pidStr[len(pidStr)-1] == "" || pidStr[len(pidStr)-1] == "0" {
		procLog.Error.Printf("[PROCESS-CHECKER] Not found process: ")
		return pidArr
	}
	pid, err := strconv.Atoi(pidStr[len(pidStr)-1])
	if err != nil {
		procLog.Error.Printf("[PROCESS-CHECKER] Convert pid (string -> int) error: ", err)
		return pidArr
	} else if pid == 0 {
		procLog.Error.Printf("[PROCESS-CHECKER] Not found process")
		return pidArr
	}

	// Get pid Arr
	getPid = fmt.Sprintf("ps -eo ppid,pid | grep %d | awk '{print $2}'", pid)
	cmd_run = exec.Command("sh", "-c", getPid)
	stdout2, err := cmd_run.Output()
	if err != nil {
		procLog.Error.Printf("[PROCESS-CHECKER] Get pid error: ", err)
		return pidArr
	}
	pidResult := strings.Split(string(stdout2), "\n")
	for _, vals := range pidResult {
		pid, _ = strconv.Atoi(vals)
		pidArr = append(pidArr, pid)
	}

	return pidArr
}

// GetPort function collects the port information used by an app deployed on Linux ARM32 devices.
// The collected information includes:
//   - Port number
//
// Input:
//   - pid: PID of the app.
//
// Output:
//   - string: Port number.
func GetPort(pid int) string {
	var portName string = ""
	err := fuser.Update(nil)
	if err != nil {
		procLog.Error.Printf("[HEALTH] Port Error: %v\n", err)
		os.Exit(1)
	}

	// CH 1 : /dev/ttyMAX1
	// CH 2 : /dev/ttyMAX0
	// CH 3 : /dev/ttyMAX2
	// CH 4 : /dev/ttyMAX3
	index := []int{1, 0, 2, 3}

	for keys, n := range index {
		serialNum := fmt.Sprintf("/dev/ttyMAX%d", n)
		portStatus := fuser.GetPath(serialNum)
		if len(portStatus) != 0 && contains(portStatus, pid) {
			portName = fmt.Sprintf("%d", keys+1)
		}
	}

	return portName
}

// GetProcDockerd function collects resource usage statistics for containers managed by Dockerd.
//
// Input:
//   - appName: The name of the application.
//   - appId: The ID of the application.
//
// Output:
//   - int: The usage of CPU(%).
//   - int: The usage of Memory(%).
func GetProcDockerd(appName string, appId string) (int, int) {
	targetContainer := fmt.Sprintf("%s-%s", appName, appId)
	ctx := context.Background()
	containers, err := dockerClient.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		procLog.Error.Printf("Dockerclient error: %v\n", err)
		return -1, -1
	}

	for _, container := range containers {
		//procLog.Info.Printf("%s %s\n", container.ID[:10], container.Image)
		containerName := container.Names[0][1:]
		if targetContainer == containerName {

			// Check container state(Running? or Exited?)
			if container.State == "exited" {
				return -1, -1
			}

			stats, err := dockerClient.ContainerStats(ctx, container.ID, false)
			if err != nil {
				procLog.Error.Printf("Error getting container stats: %v\n", err)
				return -1, -1
			}
			defer stats.Body.Close()

			var containerStats types.StatsJSON
			err = json.NewDecoder(stats.Body).Decode(&containerStats)
			if err != nil {
				procLog.Error.Printf("Error decoding stats: %v\n", err)
				return -1, -1
			}

			cpuUsage := CalculateCPUPercent(&containerStats)
			memoryUsage := float64(containerStats.MemoryStats.Usage) / (1024 * 1024) // MB
			memoryLimit := float64(containerStats.MemoryStats.Limit) / (1024 * 1024) // MB
			memoryPercent := (memoryUsage / memoryLimit) * 100.0

			return int(cpuUsage), int(memoryPercent)
		}
	}

	procLog.Error.Printf("Not found container[%s]\n", targetContainer)
	return -1, -1
}

// Main function of the process package. Selects MQTT broker based on the device's SDTCloud
// service type and publishes messages. The payload is defined as follows:
//
// Payload = {"assetCode": SerialNumber, "data": {"appName": ~~, "appId": ~~, "pid": ~, "cpu": ~~, "memory": ~~}, "venvName": ~~}
//
// Input:
//   - mqttType: SDTCloud service type used by the device.
//   - archType: Architecture of the device.
//   - rootPath: Root path of SDTCloud stored on the device.
func RunBody(mqttType string, archType string, rootPath string, appPath string) {
	jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	// yamlFile, err := ioutil.ReadFile("./config.yaml")
	if err != nil {
		procLog.Error.Printf("[PROCESS-CHECKER] Not found file Error: %v\n", err)
	}

	var configData sdtType.ConfigInfo
	err = json.Unmarshal(jsonFile, &configData)
	if err != nil {
		procLog.Error.Printf("[PROCESS-CHECKER] Unmarshal Error: %v\n", err)
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

	// Set Docker Client
	dockerClient, err = dockerCli.NewClientWithOpts(dockerCli.FromEnv)
	if err != nil {
		procLog.Error.Printf("[MAIN] Docker connection Error: %v\n", err)
		panic(err)
	}
	defer dockerClient.Close()

	// Send app health
	var pid []int
	var mainPid, cpu, mem int
	var envList []string
	var result map[string]interface{}
	var appHealth []map[string]interface{}

	// Setting 5 sec [Delay time]
	for {
		curSec := time.Now().Second()
		if curSec%5 == 0 {
			//time.Sleep(1 * time.Second)
			break
		}
	}

	// when execute 5sec...
	delayTime := time.NewTicker(5 * time.Second)
	defer delayTime.Stop()

	for true {
		curTime := <-delayTime.C
		procLog.Info.Printf("%v: Get data.\n", curTime)

		// get app info - change dir -> json
		//dirs, err := ioutil.ReadDir(appPath)
		appInfoFile := fmt.Sprintf("%s/device.config/app.json", rootPath)
		jsonFile, err = ioutil.ReadFile(appInfoFile)
		if err != nil {
			procLog.Warn.Printf("Failed load app's info: %v\n", err)
			procLog.Warn.Printf("The app has never been deployed in device.\n")
			continue
		}
		var jsonData sdtType.AppConfig
		err = json.Unmarshal(jsonFile, &jsonData)
		if err != nil {
			procLog.Error.Printf("Failed delete app's Unmarshal: %v\n", err)
			return
		}

		// Init variable.
		appHealth = make([]map[string]interface{}, 0)
		envList = make([]string, 0)

		for _, appInfo := range jsonData.AppInfoList {
			// check -> systemd and dockerd
			cpu = -1
			mem = -1
			mainPid = 0

			if appInfo.Managed == "dockerd" {
				cpu, mem = GetProcDockerd(appInfo.AppName, appInfo.AppId)
			} else { // systemd
				if archType == "win" {
					pid = WinGetPid(appInfo.AppName)
				} else {
					pid = GetPid(appInfo.AppName)
				}
				cpu, mem = GetProc(pid)
				if len(pid) == 0 {
					mainPid = -1
				} else {
					mainPid = pid[0]
				}
			}

			if mainPid == 0 {
				mainPid = -1
			}

			// for nodeq!!!
			// portName := getPort(mainPid)
			healthData := map[string]interface{}{
				"appName": appInfo.AppName,
				"appId":   appInfo.AppId,
				"pid":     mainPid,
				"cpu":     cpu,
				"memory":  mem,
				// "portName": portName,
			}
			appHealth = append(appHealth, healthData)
		}

		// Get env list
		envDir, _ := ioutil.ReadDir(fmt.Sprintf("%s/venv", rootPath))

		for _, f := range envDir {
			if f.IsDir() {
				envList = append(envList, f.Name())
			}
		}

		result = map[string]interface{}{
			"assetCode": configData.AssetCode,
			"data":      appHealth,
			"venvName":  envList,
		}

		sendDataEdgeMqtt(result, configData)

		//time.Sleep(delay * time.Second)
	}
}
