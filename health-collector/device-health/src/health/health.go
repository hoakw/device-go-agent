// The health package collects health information of the device and
// sends it to the SDT Cloud via MQTT messages. DeviceHealth connects to MQTT
// and publishes the device's health information every second.
package health

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	insp_net "net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	// ******

	human "github.com/dustin/go-humanize"
	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/leizongmin/fuser"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"

	sdtType "main/src/healthType"
)

// Global variables used in the health package.
//   - cli: MQTT Client type variable. Represents the connected MQTT server client.
//   - networkRecv: Network receive value.
//   - networkSent: Network sent value.
//   - delay: MQTT message publishing interval in seconds.
//   - KiB: Ki-byte unit.
//   - MiB: Mi-byte unit.
//   - GiB: Gi-byte unit.
//   - KB: K-byte unit.
//   - MB: M-byte unit.
//   - GB: G-byte unit.
//   - mqttUser: User ID used for MQTT connection.
//   - mqttPassword: Password used for MQTT connection.
//   - procLog: Struct that defines the format of the log.
var (
	cli          mqttCli.Client
	networkRecv  float64       = 0
	networkSent  float64       = 0
	delay        time.Duration = 2 //
	KiB          float64       = 1024
	MiB          float64       = 1024 * 1024
	GiB          float64       = 1024 * 1024 * 1024
	KB           float64       = 1000
	MB           float64       = 1000 * 1000
	GB           float64       = 1000 * 1000 * 1000
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
//   - *mqttCli.ClientOptions: Variable of type MQTT ClientOptions.
func createAwsClientOptions(mqttURL, rootCa, fullCertChain, clientKey, assetCode string) *mqttCli.ClientOptions {
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
	opts.SetClientID(fmt.Sprintf("blokworks-client-health-%s", cilentUUID))
	opts.SetConnectionLostHandler(func(client mqttCli.Client, err error) {
		procLog.Error.Printf("[MQTT] Connection lost: %v\n", err)
		os.Exit(1)
	})

	return opts
}

// This function defines options for connecting to the mosquitto MQTT broker.
// It specifies options such as TLS, MQTT URI, client name, and handlers.
//
// Input:
//   - config: Struct storing the config file saved on the device in JSON format.
//
// Output:
//   - mqttCli.Client: Variable of type MQTT client.
func connectToMqtt(
	config sdtType.ConfigInfo, // Information of config
) mqttCli.Client {
	procLog.Info.Printf("[MQTT] In connectToMqtt Function")
	opts := mqttCli.NewClientOptions()
	opts.AddBroker(config.MqttUrl)
	opts.SetPassword(mqttPassword)
	opts.SetUsername(mqttUser)
	opts.SetClientID(fmt.Sprintf("blokworks-client-health-%s", config.AssetCode))
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
	// topic := fmt.Sprintf("$aws/things/sdt-cloud-development/shadow/name/device-health/%s", config.AssetCode)
	topic := fmt.Sprintf("%s/%s/%s/bwc/health", config.ServiceCode, config.ProjectCode, config.AssetCode)

	resultBody, err := json.Marshal(payload)
	if err != nil {
		procLog.Error.Printf("[MQTT Unmarshal error: %v\n", err)
	}
	pub_token := cli.Publish(topic, 0, false, resultBody)

	if pub_token.Wait() && pub_token.Error() != nil {
		procLog.Error.Printf("[MQTT] Error: %v\n", pub_token.Error())
	}
}

// GetCpu function collects CPU information from the device. The collected information includes:
//   - CPU usage rate
//   - Number of CPU cores
//
// Output:
//   - NodeCpu = {"Cpu": usage rate, "Total": number of cores, "Time": collection time}
func GetCpu() (map[string]interface{}, []sdtType.NodeCpu) {
	cpuInfo, err := cpu.Percent(time.Second, false) // get data after 1sec.
	cpuTotal, err := cpu.Counts(true)
	if err != nil {
		procLog.Error.Printf("[HEALTH] CPU Error: %v\n", err)
	}
	newCpu := map[string]interface{}{
		"usage": fmt.Sprintf("%0.4f", cpuInfo[0]),
		"total": fmt.Sprintf("%d", cpuTotal*100),
	}

	nodeCpu_arr := make([]sdtType.NodeCpu, 0)
	insp_newCpu := sdtType.NodeCpu{
		Cpu:   cpuInfo[0],
		Total: cpuTotal,
		Time:  time.Now(),
	}

	nodeCpu_arr = append(nodeCpu_arr, insp_newCpu)

	return newCpu, nodeCpu_arr
}

// GetMem function collects memory information from the device. The collected information includes:
//   - Memory usage rate
//   - Total memory size
//
// Output:
//   - NodeMem = {"Mem": usage rate, "Total": total size, "Time": collection time}
func GetMem() (map[string]interface{}, []sdtType.NodeMem) {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		procLog.Error.Printf("[HEALTH] Memory Error: %v\n", err)
	}
	newMem := map[string]interface{}{
		"usage": fmt.Sprintf("%d", int64(float64(memInfo.Used)/KiB)),
		"total": fmt.Sprintf("%d", int64(float64(memInfo.Total)/KiB)),
	}

	nodeMem_arr := make([]sdtType.NodeMem, 0)
	insp_newMem := sdtType.NodeMem{
		Mem:   memInfo.UsedPercent,
		Total: fmt.Sprintf("%0.5f", float64(memInfo.Total)/MiB/KB),
		Time:  time.Now(),
	}

	nodeMem_arr = append(nodeMem_arr, insp_newMem)

	return newMem, nodeMem_arr
}

// GetProc function collects process information from the device. The collected information includes:
//   - Process ID (PID)
//   - Process CPU usage rate
//   - Process Memory usage rate
//   - Process name
//
// Output:
//   - ProcInfo = {"Id": PID, "Cpu": usage rate, "Memory": usage rate, "Context": process name, "Time": collection time}
func GetProc(totalCpu int) []sdtType.ProcInfo {
	processes, _ := process.Processes()

	processes_cpu_mem := make([]sdtType.ProcInfo, 0)
	for _, p := range processes {
		p_cpu_percent, _ := p.CPUPercent()
		p_mem_percent, _ := p.MemoryPercent()
		p_name, _ := p.Name()
		p_id := fmt.Sprintf("%d", int(p.Pid))
		if p_cpu_percent < 0.3 {
			continue
		}
		newProc := sdtType.ProcInfo{
			Id:      p_id,
			Cpu:     float32(p_cpu_percent) / float32(totalCpu),
			Memory:  float32(p_mem_percent),
			Context: p_name,
			Time:    time.Now(),
		}

		processes_cpu_mem = append(processes_cpu_mem, newProc)
	}

	return processes_cpu_mem
}

// GetDisk function collects disk information from the device. The collected information includes:
//   - Disk name
//   - Disk total size
//   - Disk used size
//   - Disk usage rate
//   - Disk mount point
//
// Output:
//   - map[string]interface{} = {"total": total size of '/' path, "usage": usage of '/' path}
//   - DiskInfo = {"Name": name, "Totalsize": total size, "Used": used size, "UsedPercent": usage rate, "Mountpoint": mount point, "Time": collection time}
//   - float64 = overall disk usage rate
func GetDisk() (map[string]interface{}, []sdtType.DiskInfo, float64) {
	disks, _ := disk.Usage("/")
	newDisk := map[string]interface{}{
		"total": fmt.Sprintf("%d", int64(float64(disks.Total)/KB)),
		"usage": fmt.Sprintf("%d", int64(float64(disks.Used)/KB)),
	}

	diskList := make([]sdtType.DiskInfo, 0)
	parts, _ := disk.Partitions(true)
	totalDisk := 0.0
	usedDisk := 0.0
	for _, p := range parts {
		device := p.Mountpoint
		s, _ := disk.Usage(device)
		if s == nil || s.Total == 0 {
			continue
		} else if strings.Contains(p.Mountpoint, "var/lib") {
			continue
		}
		totalDisk = totalDisk + float64(s.Total)/GiB
		usedDisk = usedDisk + float64(s.Used)/GiB + 0.000001

		disk_percent := fmt.Sprintf("%0.2f", s.UsedPercent)
		insp_newDisk := sdtType.DiskInfo{
			Name:        p.Device,
			Totalsize:   human.Bytes(s.Total),
			Used:        float64(s.Used)/GiB + 0.000001,
			UsedPercent: disk_percent,
			Mountpoint:  p.Mountpoint,
			Time:        time.Now(),
		}

		diskList = append(diskList, insp_newDisk)
	}

	sort.SliceStable(diskList, func(i, j int) bool {
		return diskList[i].Used > diskList[j].Used
	})

	return newDisk, diskList, usedDisk / totalDisk * 100
}

// GetSerial function collects the serial information from the device. The collected information includes:
//   - Port information
//   - Tx value
//   - Rx value
//
// Output:
//   - SerialInfo = {"Index": index, "Uart": Uart value, "Port": port, "Irq": IRQ usage, "Tx": Tx usage, "Rx": Rx usage, "Time": collection time}
func GetSerial(archType string) []sdtType.SerialInfo {
	if archType == "win" {
		return nil
	}
	data, err := os.Open("/proc/tty/driver/serial")
	if err != nil {
		fmt.Println(err)
	}
	defer data.Close()

	scan := bufio.NewScanner(data)

	serial_info := make([]sdtType.SerialInfo, 0)
	var tx int
	var rx int

	scan.Scan()
	for scan.Scan() {
		data_slice := strings.Split(strings.Replace(scan.Text(), ":", " ", -1), " ")
		if len(data_slice) < 9 {
			tx = -1
			rx = -1
		} else {
			tx, _ = strconv.Atoi(data_slice[9])
			rx, _ = strconv.Atoi(data_slice[11])
		}
		// index, _ := strconv.Atoi(data_slice[0])
		irq, _ := strconv.Atoi(data_slice[7])
		newSerial := sdtType.SerialInfo{
			Index: data_slice[0],
			Uart:  data_slice[3],
			Port:  data_slice[5],
			Irq:   irq,
			Tx:    tx,
			Rx:    rx,
			Time:  time.Now(),
		}
		serial_info = append(serial_info, newSerial)
	}

	return serial_info
}

// GetNetwork function collects the network information from the device. The collected information includes:
//   - Network name
//   - Network address
//   - Network MTU
//   - Network hardware address
//
// Output:
//   - NetInfo = {"Index": index, "Name": name, "Address": IP, "Mtu": MTU, "HardwareAddr": hardware address, "Time": collection time}
func GetNetwork(archType string) (map[string]interface{}, []sdtType.NetInfo, string, string) {
	netw, err := net.IOCounters(false)
	if err != nil {
		procLog.Error.Printf("[HEALTH] Network Error: %v\n", err)
		os.Exit(1)
	}
	var uplink float64 = 0
	var downlink float64 = 0
	if networkRecv != 0 {
		downlink = (float64(netw[0].BytesRecv) - networkRecv) * 8 / float64(delay-1) / 1000 // GetCpu use 1sec.
		uplink = (float64(netw[0].BytesSent) - networkSent) * 8 / float64(delay-1) / 1000   // // GetCpu use 1sec.
	}
	networkRecv = float64(netw[0].BytesRecv)
	networkSent = float64(netw[0].BytesSent)
	// fmt.Println("NET: ", netw[0].BytesSent, "/", netw[0].BytesRecv)

	newNet := map[string]interface{}{
		"uplink":   fmt.Sprintf("%0.4f", uplink),
		"downlink": fmt.Sprintf("%0.4f", downlink),
	}

	insp_netw, err := insp_net.Interfaces()
	if err != nil {
		os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}

	net_info := make([]sdtType.NetInfo, 0)
	var net_addrs string
	inNet := ""
	outNet := ""

	var ipIndex int
	if archType == "win" {
		ipIndex = 1
	} else {
		ipIndex = 0
	}

	for _, inter := range insp_netw {
		addrs, _ := inter.Addrs()
		// fmt.Printf("[TEST] %s: %s\n", inter.Name, addrs)
		if strings.Contains(inter.Name, "docker") {
			continue
		}
		if len(addrs) < 2 {
			continue
		} else if len(addrs) == 0 {
			// net_addrs = "notFound"
			continue
		} else if strings.Contains(addrs[ipIndex].String(), ":") {
			continue
			// net_addrs = "notFound"
		}

		// -------------------------device network interface!!
		net_addrs = addrs[ipIndex].String()
		// fmt.Printf("[TEST] %s: %s\n", inter.Name, net_addrs)
		if strings.Contains(inter.Name, "ham") || strings.Contains(inter.Name, "ztt") || strings.Contains(strings.ToLower(inter.Name), "zerotier") {
			outNet = outNet + fmt.Sprintf("/ %s: %s  ", inter.Name, net_addrs)
		} else {
			inNet = inNet + fmt.Sprintf("/ %s: %s  ", inter.Name, net_addrs)
		}

		if len(outNet) > 250 || len(inNet) > 250 {
			break
		}
		// -------------------------

		insp_newNet := sdtType.NetInfo{
			Index:        inter.Index,
			Name:         inter.Name,
			HardwareAddr: inter.HardwareAddr,
			Mtu:          inter.MTU,
			Address:      net_addrs,
			Time:         time.Now(),
		}
		net_info = append(net_info, insp_newNet)
	}
	inNet = strings.Trim(inNet, "/")
	outNet = strings.Trim(outNet, "/")

	return newNet, net_info, inNet, outNet
}

// GetPort function collects the port information from the device. The collected information includes:
//   - Port status
//
// Output:
//   - map[string]interface{} = {"code": port number, "status": port status}
func GetPort() []map[string]interface{} {
	err := fuser.Update(nil)
	if err != nil {
		procLog.Error.Printf("[HEALTH] Port Error: %v\n", err)
		os.Exit(1)
	}

	portInfo := make([]map[string]interface{}, 0)

	// CH 1 : /dev/ttyMAX1
	// CH 2 : /dev/ttyMAX0
	// CH 3 : /dev/ttyMAX2
	// CH 4 : /dev/ttyMAX3
	index := []int{1, 0, 2, 3}

	for k, n := range index {
		portValue := make(map[string]interface{}, 0)
		serialNum := fmt.Sprintf("/dev/ttyMAX%d", n)
		portStatus := fuser.GetPath(serialNum)
		if len(portStatus) == 0 {
			portValue = map[string]interface{}{
				"code":   fmt.Sprintf("%d", k+1),
				"status": 0,
			}
		} else {
			portValue = map[string]interface{}{
				"code":   fmt.Sprintf("%d", k+1),
				"status": 1,
			}
		}
		portInfo = append(portInfo, portValue)
	}

	return portInfo
}

// GetGPU function
//
// Output:
//   - []map[string]interface{}: GPU information.
func GetGPU() ([]map[string]interface{}, []map[string]interface{}) {
	var out, gpuOut bytes.Buffer
	var gpuInfo, gpuMeta []map[string]interface{}

	// Get list of gpus
	//cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,utilization.gpu,memory.total,memory.used,temperature.gpu", "--format=csv,noheader,nounits")
	cmd := exec.Command("nvidia-smi", "-L")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		procLog.Warn.Printf("[INFO] This device not have gpu device or not found nvidia-smi cmd.\n%v\n", err)
		return nil, nil
	}

	// Parsing for output
	output := out.String()
	lines := strings.Split(output, "\n")
	lines = lines[:len(lines)-1] // 마지막 값은 필요 없는 요소이므로 삭제(\n 이후로 빈값이 있음)

	for n, _ := range lines {
		gpuOut.Reset() // 값 초기화
		gpuCmd := exec.Command("nvidia-smi", "-i", fmt.Sprintf("%d", n), "--query-gpu=index,name,utilization.gpu,memory.total,memory.used,temperature.gpu,fan.speed", "--format=csv,noheader,nounits")
		//procLog.Warn.Printf("[DEBUG] CMD: %s\n", gpuCmd)
		gpuCmd.Stdout = &gpuOut
		err = gpuCmd.Run()

		if err != nil {
			procLog.Warn.Printf("[WARN] Not found %d's GPU[%d].\n%v\n", n+1, n, err)
			continue
		}
		// Convert to string
		gpuOutput := gpuOut.String()

		// 공백 제거
		if strings.TrimSpace(gpuOutput) == "" {
			continue
		}
		// 뒤에 \n 제거
		gpuOutput = strings.Split(gpuOutput, "\n")[0]
		//procLog.Warn.Printf("[DEBUG] %d's DATA: %s\n", n, gpuOutput)

		fields := strings.Split(gpuOutput, ", ")
		// Get GPU Data
		gpuData := map[string]interface{}{
			"index":    fields[0],
			"name":     fields[1],
			"util":     fields[2],
			"totalMem": fields[3],
			"usedMem":  fields[4],
			"temp":     fields[5],
			"fanSpeed": fields[6],
		}
		gpuInfo = append(gpuInfo, gpuData)

		// Get GPU Metadata
		gpuData = map[string]interface{}{
			"index": fields[0],
			"name":  fields[1],
		}
		gpuMeta = append(gpuMeta, gpuData)
	}

	return gpuInfo, gpuMeta
}

// CheckNetwork function checks the network information of the device. If the network
// information has changed, it returns that the network information has been updated.
//
// Input:
//   - curNet: Currently stored network information
//   - targetNet: Network information currently retrieved from the device
//
// Output:
//   - bool: true (changed) or false (unchanged)
func CheckNetwork(curNet map[string]interface{}, targetNet map[string]interface{}) bool {
	if curNet["privateIP"] != targetNet["privateIP"] {
		return true
	} else if curNet["publicIP"] != targetNet["publicIP"] {
		return true
	}
	return false
}

// SendNetInfo function sends updated network information of the device to the cloud
// when network information has changed.
//
// Input:
//   - assetCode: Serial number (AssetCode) of the device.
//   - input: Current network information of the device.
//   - bwUrl: API URL of the cloud.
func SendNetInfo(assetCode string, input map[string]interface{}, bwUrl string) {
	pbytes, _ := json.Marshal(input)
	buff := bytes.NewBuffer(pbytes)

	apiUrl := fmt.Sprintf("%s/assets/%s/hardware", bwUrl, assetCode)
	procLog.Info.Printf("[HTTP] Check... change IP's info\n")
	req, err := http.NewRequest("PUT", apiUrl, buff)
	if err != nil {
		procLog.Error.Printf("[HTTP] Http not connected. : %v(%s)\n", err, apiUrl)
	}

	req.Header.Add("Content-Type", "application/json")
	// req.Header.Add("X-OrganizationId", organizationId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("[HTTP] Http API Call Failed: %v(%s)\n", err, apiUrl)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	_, err = ioutil.ReadAll(resp.Body)
	// fmt.Println("[INFO]: 1-4. Send H/W information: ", resp.Status)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])
	if statusValue >= 400 {
		procLog.Error.Printf("[HTTP] API Call Error: %v(%s)\n", errors.New("Fail api call."), apiUrl)
		procLog.Error.Printf("[HTTP]Error: [%d] %v \n", statusValue, err)
	}

}

// Main function of the health package. Depending on the SDT Cloud service type of the device,
// this function selects an MQTT broker and publishes messages.
// The message is defined as follows:
//
//	Payload = {"timestamp": 1858182312, "data": {"cpu": ~~, "memory": ~~, "disk": ~, "network": ~~, "port": ~~}}
//
// Input:
//   - mqttType: SDTCloud service type of the device.
//   - archType: Architecture of the device.
//   - rootPath: Root path of SDTCloud stored on the device.
func RunBody(mqttType string, archType string, rootPath string) {
	var configData sdtType.ConfigInfo
	curNetInter := map[string]interface{}{
		"privateIP": "",
		"publicIP":  "",
	}

	jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
	procLog.Info.Printf("[HEALTH] read config file: %s \n", jsonFilePath)
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		procLog.Error.Printf("[HEALTH] Not found file Error: %v\n", err)
	}

	err = json.Unmarshal(jsonFile, &configData)
	if err != nil {
		procLog.Error.Printf("[HEALTH] Unmarshal Error: %v\n", err)
	}

	// Set apiurl
	var bwUrl string
	if configData.ServiceType == "eks" {
		bwUrl = "http://cloud-api-router.sdt.services"
	} else if configData.ServiceType == "aws-dev" {
		bwUrl = "http://43.200.53.170:31731"
	} else if configData.ServiceType == "dev" {
		bwUrl = "http://192.168.1.162:31731"
	} else if configData.ServiceType == "onprem" {
		bwUrl = fmt.Sprintf("http://%s:31731", configData.ServerIp)
	} else {
		procLog.Error.Printf("%s is not supported.\n", configData.ServiceType)
		os.Exit(1)
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
		opts := createAwsClientOptions(configData.MqttUrl, rootCa, fullCertChain, private, configData.AssetCode)
		cli = mqttCli.NewClient(opts)
	} else {
		err = errors.New("Please input mqtt variable.")
		procLog.Error.Printf("[MAIN] mqtt command Error: %v\n", err)
		panic(err)
	}

	if token := cli.Connect(); token.Wait() && token.Error() != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", token.Error())
	}

	defer cli.Disconnect(250)

	// Setting 5 sec
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

		//node CPU
		nodecpu_info, inspectorCpu := GetCpu()
		//node Memory
		nodemem_info, inspectorMem := GetMem()
		//processor
		inspectorProc := GetProc(inspectorCpu[0].Total)
		//disk
		disk_info, inspectorDisk, totalUsedDisk := GetDisk()
		//serial info
		inspectorSerial := GetSerial(archType)
		//network info
		net_info, inspectorNet, inNet, outNet := GetNetwork(archType)
		//port info
		port_info := GetPort()
		//gpu info
		gpuInfo, gpuMeta := GetGPU()

		healthData := map[string]interface{}{
			"cpu":     nodecpu_info,
			"memory":  nodemem_info,
			"disk":    disk_info,
			"network": net_info,
			"port":    port_info,
			"gpu":     gpuInfo,
		}

		netInter := map[string]interface{}{
			"privateIP": inNet,
			"publicIP":  outNet,
		}
		// fmt.Println("GET NETWORK: ", netInter)
		if CheckNetwork(curNetInter, netInter) {
			// For gpu, gpu info send to bw when reboot or restart(process).
			hwMsg := map[string]interface{}{
				"network": netInter,
				"gpu":     gpuMeta,
			}

			SendNetInfo(configData.AssetCode, hwMsg, bwUrl)
			curNetInter["privateIP"] = inNet
			curNetInter["publicIP"] = outNet

		}

		msg := map[string]interface{}{
			//"timestamp": int64(time.Now().UTC().Unix() * 1000),
			"timestamp": int64(curTime.UTC().Unix() * 1000),
			"data":      healthData,
		}
		fmt.Printf("Time: %v / %d\n", curTime, int64(curTime.UTC().Unix()*1000))
		// fmt.Println(msg)
		sendDataEdgeMqtt(msg, configData)

		// Save Inspector File
		all_data := map[string]interface{}{
			"time":          time.Now().Unix(),
			"process":       inspectorProc,
			"disk":          inspectorDisk,
			"totalUsedDisk": totalUsedDisk,
			"serial":        inspectorSerial,
			"network":       inspectorNet,
			"cpu":           inspectorCpu,
			"memory":        inspectorMem,
		}

		data_string, err := json.Marshal(all_data)
		if err != nil {
			procLog.Error.Printf("[HEALTH] Marshal Error: %v \n", err)
			panic(err)
		}

		var dir string
		if archType == "win" {
			dir = fmt.Sprintf("C:/sdt/inspector")
		} else {
			dir = fmt.Sprintf("/etc/sdt/inspector")
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, os.ModePerm)
		}

		inspectorFile := fmt.Sprintf("%s/data.json", dir)
		err = ioutil.WriteFile(inspectorFile, data_string, os.FileMode(0644))
		if err != nil {
			procLog.Error.Printf("[HEALTH] Wriet Error: %v\n", err)
			panic(err)
		}

		//procLog.Info.Printf("Send device's health.\n")
		//time.Sleep((delay - 1) * time.Second) // GetCpu use 1sec.
	}
}

// RunBodyForInspector function creates a Data.json file for use by the Inspector.
// Data.json stores information about the device's resources, network, etc.
//
// Input:
//   - archType: Architecture of the device.
func RunBodyForInspector(archType string) {
	for true {
		//node CPU
		_, inspectorCpu := GetCpu()
		//node Memory
		_, inspectorMem := GetMem()
		//processor
		inspectorProc := GetProc(inspectorCpu[0].Total)
		//disk
		_, inspectorDisk, totalUsedDisk := GetDisk()
		//serial info
		inspectorSerial := GetSerial(archType)
		//network info
		_, inspectorNet, _, _ := GetNetwork(archType)

		// Save Inspector File
		all_data := map[string]interface{}{
			"time":          time.Now().Unix(),
			"process":       inspectorProc,
			"disk":          inspectorDisk,
			"totalUsedDisk": totalUsedDisk,
			"serial":        inspectorSerial,
			"network":       inspectorNet,
			"cpu":           inspectorCpu,
			"memory":        inspectorMem,
		}

		data_string, err := json.Marshal(all_data)
		if err != nil {
			procLog.Error.Printf("[HEALTH] Marshal Error: %v \n", err)
			panic(err)
		}

		var dir string
		if archType == "win" {
			dir = fmt.Sprintf("C:/sdt/inspector")
		} else {
			dir = fmt.Sprintf("/etc/sdt/inspector")
		}
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			os.Mkdir(dir, os.ModePerm)
		}

		inspectorFile := fmt.Sprintf("%s/data.json", dir)
		err = ioutil.WriteFile(inspectorFile, data_string, os.FileMode(0644))
		if err != nil {
			procLog.Error.Printf("[HEALTH] Wriet Error: %v\n", err)
			panic(err)
		}

		time.Sleep((delay - 1) * time.Second) // GetCpu use 1sec.
	}
}
