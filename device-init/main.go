// This package is the main package for device installation. Device installation
// handles the functionality of registering devices with the cloud.
// This package supports server architectures such as windows and linux.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	// "runtime"
	"github.com/shirou/gopsutil/v3/host"

	sdtAquaRack "main/src/aquarack"
)

// GetOS function retrieves the operating system (OS) information of the device.
//
// Output:
//   - map[string]interface{}: Structure containing the OS information.
func GetOS() map[string]interface{} {
	osInfo, _ := host.Info()
	osValue := map[string]interface{}{
		"osType":    osInfo.Platform,
		"osVersion": osInfo.PlatformVersion,
		"createdAt": int64(osInfo.BootTime) * 1000,
	}

	return osValue
}

// GetNetwork function retrieves the network IP information of the device,
// distinguishing between internal and external IPs.
//
// Input:
//   - systemArch: Architecture of the device.
//
// Output:
//   - string: Internal IP address.
//   - string: External IP address.
func GetNetwork(systemArch string) (string, string) {
	var ipIndex int
	if systemArch == "win" {
		ipIndex = 1
	} else {
		ipIndex = 0
	}

	netw, err := net.Interfaces()
	if err != nil {
		os.Stderr.WriteString("Oops: " + err.Error() + "\n")
		os.Exit(1)
	}

	var net_addrs string
	inNet := ""
	outNet := ""
	for _, inter := range netw {
		addrs, _ := inter.Addrs()
		if strings.Contains(inter.Name, "docker") {
			continue
		}
		if len(addrs) < 2 {
			continue
		} else if len(addrs) == 0 {
			continue
		} else if strings.Contains(addrs[ipIndex].String(), ":") {
			continue
		}

		net_addrs = addrs[ipIndex].String()
		if strings.Contains(strings.ToLower(inter.Name), "ham") || strings.Contains(strings.ToLower(inter.Name), "ztt") || strings.Contains(strings.ToLower(inter.Name), "zerotier") {
			outNet = outNet + fmt.Sprintf("/ %s: %s  ", inter.Name, net_addrs)
		} else {
			inNet = inNet + fmt.Sprintf("/ %s: %s  ", inter.Name, net_addrs)
		}

		if len(outNet) > 250 || len(inNet) > 250 {
			break
		}

	}
	inNet = strings.Trim(inNet, "/")
	outNet = strings.Trim(outNet, "/")

	return inNet, outNet

}

// GetGPU function
//
// Output:
//   - []map[string]interface{}: GPU information.
func GetGPU() []map[string]interface{} {
	var out bytes.Buffer
	var gpuInfo []map[string]interface{}

	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,utilization.gpu,memory.total,memory.used,temperature.gpu", "--format=csv,noheader,nounits")
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Printf("[INFO] This device not have gpu device or not found nvidia-smi cmd.\n%v\n", err)
		return nil
	}

	// Parsing for output
	output := out.String()
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Split(line, ", ")
		gpuData := map[string]interface{}{
			"index": fields[0],
			"name":  fields[1],
			//"util":     fields[2],
			//"totalMem": fields[3],
			//"UsedMem":  fields[4],
			//"temp":     fields[5],
		}
		gpuInfo = append(gpuInfo, gpuData)
	}

	return gpuInfo
}

// RegisterDevice registers the device with the cloud. It calls the cloud's
// device registration API to perform the registration. This step registers
// the device with the cloud but does not make it operational or usable by the cloud.
//
// Input:
//   - serialNumber: Serial number of the device.
//   - OrganizationId: Organization ID to register the device under.
//   - BwURL: BW API URL of the cloud.
//   - BwPort: BW API Port of the cloud.
//
// Output:
//   - string: Cloud access key.
//   - string: Cloud secret key.
func RegisterDevice(serialNumber string, organizationId string, bwURL string, bwPort int) (string, string) {
	input := map[string]interface{}{
		"code": serialNumber,
	}
	pbytes, _ := json.Marshal(input)
	buff := bytes.NewBuffer(pbytes)

	apiUrl := fmt.Sprintf("http://%s:%d/init/assets", bwURL, bwPort)
	req, err := http.NewRequest("POST", apiUrl, buff)
	if err != nil {
		fmt.Printf("[ERROR] Http not connected. : %v\n", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OrganizationId", organizationId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Connection to SDTCloud failed.\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	respBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println("[INFO]: 1-1. Register equirement: ", resp.Status)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])

	// TODO
	//  - 에러 코드에 대한 에러메시지 반환 - BlokWorks와 맞춰야 함.
	if statusValue == 404 {
		fmt.Printf("[ERROR] %s not found in SDTCloud.\n", serialNumber)
		fmt.Printf("[ERROR] Please check your serialNumber.\n")
		os.Exit(1)
	} else if statusValue == 409 {
		fmt.Printf("[ERROR] %s already been registered in SDTCloud.\n", serialNumber)
		fmt.Printf("[ERROR] Please check your serialNumber.\n")
		os.Exit(1)
	} else if statusValue < 400 {
		fmt.Println("[INFO]: Success register.")
	}

	result := map[string]interface{}{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		fmt.Printf("[ERROR] This is an incorrect secret key.\b")
		fmt.Printf("[ERROR] Please contact SDT inc.\n")
		os.Exit(1)
	}
	// fmt.Println(result["model"]["code"])
	accessKeyId := fmt.Sprintf("%s", result["accessKeyId"])
	secretAccessKey := fmt.Sprintf("%s", result["secretAccessKey"])

	return accessKeyId, secretAccessKey
}

// ConnectDevice connects the device to the cloud. This step is part of registering
// the device with the cloud, not making it operational or usable by the cloud.
//
// Input:
//   - accessKeyId: Cloud access key.
//   - secretAccessKey: Cloud secret key.
//   - assetCode: Serial number of the device.
//   - OrganizationId: Organization ID to register the device under.
//   - BwURL: BW API URL of the cloud.
//   - BwPort: BW API Port of the cloud.
func ConnectDevice(accessKeyId string, secretAccessKey string, assetCode string, organizationId string, bwURL string, bwPort int) {
	input := map[string]interface{}{
		"accessKeyId":     accessKeyId,
		"secretAccessKey": secretAccessKey,
	}
	pbytes, _ := json.Marshal(input)
	buff := bytes.NewBuffer(pbytes)

	apiUrl := fmt.Sprintf("http://%s:%d/init/assets/%s/connection", bwURL, bwPort, assetCode)
	req, err := http.NewRequest("POST", apiUrl, buff)
	if err != nil {
		fmt.Printf("[ERROR] Http not connected. : %v\n", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OrganizationId", organizationId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Connection to SDTCloud failed.\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	// respBody, err := ioutil.ReadAll(resp.Body)
	fmt.Println("[INFO]: 1-2. Connection equirement: ", resp.Status)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])
	if statusValue < 400 {
		fmt.Println("[INFO]: Success connection.")
	} else {
		fmt.Printf("[ERROR] The device is not connected.\n")
		fmt.Printf("[ERROR] Please contact SDT inc.\n")
		os.Exit(1)
	}
}

// fileDownload downloads a file from the given URI to the specified directory with the target filename.
//
// Input:
//   - dir: BWC Root Path.
//   - targetFile: Name of the file to save.
//   - fullURLFile: URI value of the file to download.
func ProvisioningDevice(assetCode string, organizationId string, dir string, bwURL string, bwPort int, serviceType string) {
	apiUrl := fmt.Sprintf("http://%s:%d/init/assets/%s/provisions", bwURL, bwPort, assetCode)
	req, err := http.NewRequest("POST", apiUrl, nil)
	if err != nil {
		fmt.Printf("[ERROR] Http not connected. : %v\n", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OrganizationId", organizationId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Connection to SDTCloud failed.\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	fmt.Println("[INFO]: 1-3. Provisioning: ", resp.Status)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])
	if statusValue < 400 {
		fmt.Println("[INFO]: Success provisioning.")
	} else {
		fmt.Printf("[ERROR] The device provisioning failed.\n")
		fmt.Printf("[ERROR] Please contact SDT inc.\n")
		os.Exit(1)
	}

	// response data
	respBody, err := ioutil.ReadAll(resp.Body)
	result := map[string]interface{}{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		panic(err)
	}

	// Download cert file.
	var rootcaFile string
	if serviceType == "onprem" {
		rootcaFile = "rootCa.pem"
	} else {
		rootcaFile = "AmazonRootCA1.pem"
	}
	priFile := "no_project-private.pem"
	certFile := "no_project-certificate.pem"

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
	}

	fileDownload(dir, rootcaFile, result["rootCa"].(string))
	fileDownload(dir, priFile, result["privateKey"].(string))
	fileDownload(dir, certFile, result["certificate"].(string))
	//if serviceType != "onprem" {
	//	fileDownload(dir, rootcaFile, result["rootCa"].(string))
	//	fileDownload(dir, priFile, result["privateKey"].(string))
	//	fileDownload(dir, certFile, result["certificate"].(string))
	//}
}

// fileDownload downloads a file from the specified URI to the given directory with the target filename.
//
// Input:
//   - dir: The root path of BWC.
//   - targetFile: The name of the file to save.
//   - fullURLFile: The URI value of the file to download.
func fileDownload(dir string, targetFile string, fullURLFile string) {

	fileName := fmt.Sprintf("%s%s", dir, targetFile)
	file, err := os.Create(fileName)
	if err != nil {
		fmt.Println("[ERROR] fileDownload file creation error: ", err)
		os.Exit(1)
	}

	// Put content on file
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	resp, err := client.Get(fullURLFile)
	if err != nil {
		fmt.Println("[ERROR] fileDownload URL get file error: ", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.Status[:3] != "200" {
		fmt.Printf("[ERROR] Download error: %s\n", resp.Status)
		os.Exit(1)
	}
	io.Copy(file, resp.Body)
	//fmt.Printf("[INFO]: Downloaded a file %s with size %d\n", fileName, size)
	fmt.Printf("[INFO] Successfully download cert files.\n")

	defer file.Close()
}

// SendHwInfo sends hardware information of the device to the cloud. The collected
// information includes OS and Network details. After this step, the device can be used in the cloud.
//
// Input:
//   - assetCode: The serial number of the device.
//   - OrganizationId: The organization ID for registration.
//   - osInfo: OS information of the device.
//   - inNet: Internal network IP address.
//   - OutNet: External network IP address.
//   - BwURL: BW API URL of the cloud.
//   - BwPort: BW API Port of the cloud.
func SendHwInfo(assetCode string, organizationId string, osInfo map[string]interface{}, gpuInfo []map[string]interface{}, inNet string, outNet string, bwURL string, bwPort int) {
	input := map[string]interface{}{
		"os": osInfo,
		"network": map[string]interface{}{
			"privateIP": inNet,
			"publicIP":  outNet,
		},
		"gpu": gpuInfo,
	}

	pbytes, _ := json.Marshal(input)
	buff := bytes.NewBuffer(pbytes)

	apiUrl := fmt.Sprintf("http://%s:%d/init/assets/%s/hardware", bwURL, bwPort, assetCode)
	req, err := http.NewRequest("POST", apiUrl, buff)
	if err != nil {
		fmt.Printf("[ERROR] Http not connected. : %v\n", err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OrganizationId", organizationId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[ERROR] Connection to SDTCloud failed.\n")
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	_, err = ioutil.ReadAll(resp.Body)
	fmt.Println("[INFO]: 1-4. Send H/W information: ", resp.Status)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])
	if statusValue < 400 {
		fmt.Println("[INFO] Success send H/W data.")
	} else {
		fmt.Printf("[WARNING] Data transmission failed..\n")
	}

}

// SetAssetCode saves the device's serial number to the BWC config. BWC config is
// stored in "/etc/sdt/device.config" on the device.
//
// Input:
//   - archType: The architecture of the device.
//   - asset: The serial number of the device.
//   - Organization: The organization ID for registration.
func SetAssetCode(archType string, asset string, organzation string) {
	var targetFile string
	if archType == "win" {
		targetFile = "C:/sdt/device.config/config.json"
	} else {
		targetFile = "/etc/sdt/device.config/config.json"
	}

	jsonFile, err := ioutil.ReadFile(targetFile)
	if err != nil {
		fmt.Printf("[ERROR](setAssetCode) Not found file Error: %v\n", err)
		os.Exit(1)
	}
	var jsonData map[string]interface{}
	jsonRecode := json.NewDecoder(strings.NewReader(string(jsonFile)))
	jsonRecode.UseNumber()
	err = jsonRecode.Decode(&jsonData)
	if err != nil {
		fmt.Printf("[ERROR](setAssetCode) Unmarshal Error: %v\n", err)
		os.Exit(1)
	}

	jsonData["assetcode"] = asset
	jsonData["organzation"] = organzation

	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
	err = ioutil.WriteFile(targetFile, saveJson, 0644)
	if err != nil {
		fmt.Printf("[ERROR](setAssetCode) Not found file Error: %v\n", err)
		os.Exit(1)
	}
}

// SetConfig saves cloud service information to the BWC config. BWC config is
// stored in /etc/sdt/device.config on the device.
//
// Input:
//   - mqttURL: MQTT URL of the cloud.
//   - serviceCode: Service code of the cloud.
//   - serviceType: Service type of the cloud server (EKS, DEV, OnPrem).
func SetConfig(archType string, mqttURL string, serviceCode string, serviceType string, serverip string) (string, error) {
	var targetFile string
	if archType == "win" {
		targetFile = "C:/sdt/device.config/config.json"
	} else {
		targetFile = "/etc/sdt/device.config/config.json"
	}

	jsonFile, err := ioutil.ReadFile(targetFile)

	if err != nil {
		fmt.Printf("[ERROR](setConfig) Not found file Error: %v\n", err)
		return "", err
	}

	var jsonData map[string]interface{}
	jsonRecode := json.NewDecoder(strings.NewReader(string(jsonFile)))
	jsonRecode.UseNumber()
	err = jsonRecode.Decode(&jsonData)
	if err != nil {
		fmt.Printf("[ERROR](setConfig) Unmarshal Error: %v\n", err)
		return "", err
	}

	jsonData["mqtturl"] = mqttURL
	jsonData["servicecode"] = serviceCode
	jsonData["servicetype"] = serviceType
	jsonData["serverip"] = serverip

	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
	err = ioutil.WriteFile(targetFile, saveJson, 0644)
	if err != nil {
		return "", err
	}
	return jsonData["devicetype"].(string), err
}

// This function takes the server's architecture information as input and configures
// the environment accordingly, then executes core functions.
//
// Input:
//   - organizationId: ID of the organization to register.
//   - assetCode: Serial number of the device.
//   - archType: Architecture of the device.
//   - organization: Organization to register.
//   - serviceType: Type of cloud server.
//   - bwIP: Cloud BW IP address.
func main() {
	organizationId := flag.String("oid", "0", "0")
	assetCode := flag.String("acode", "0", "0")
	archType := flag.String("arch", "linux", "linux")
	serviceType := flag.String("type", "", "")
	bwIP := flag.String("ip", "", "")
	flag.Parse()

	// Set Service Type
	//var bwURL, mqttURL, serviceCode, codeRepoIp, codeRepoPort, fileUrl string
	//var bwPort int
	var bwURL, mqttURL, serviceCode, codeRepoIp, fileUrl string
	var bwPort, codeRepoPort int
	if *serviceType == "aws-dev" {
		codeRepoIp = "43.200.53.170"
		codeRepoPort = 32421
		bwURL = "43.200.53.170"
		mqttURL = "ssl://avk03ee629rck-ats.iot.ap-northeast-2.amazonaws.com:8883"
		bwPort = 31731
		serviceCode = "sdtcloud"
		fileUrl = "https://sdt-cloud-s3.s3.ap-northeast-2.amazonaws.com/blokworks-client/aquarack/aquarack-sensor-collector.zip"
	} else if *serviceType == "dev" {
		codeRepoIp = "192.168.1.162"
		codeRepoPort = 32421
		bwURL = "192.168.1.162"
		mqttURL = "ssl://avk03ee629rck-ats.iot.ap-northeast-2.amazonaws.com:8883"
		bwPort = 31731
		serviceCode = "sdtcloud"
		fileUrl = "https://sdt-cloud-s3.s3.ap-northeast-2.amazonaws.com/blokworks-client/aquarack/aquarack-sensor-collector.zip"
	} else if *serviceType == "onprem" {
		bwURL = *bwIP
		codeRepoIp = bwURL
		codeRepoPort = 32421
		mqttURL = fmt.Sprintf("tcp://%s:32259", bwURL)
		bwPort = 31731
		serviceCode = "sdtcloud"
		fileUrl = "https://sdt-cloud-s3.s3.ap-northeast-2.amazonaws.com/blokworks-client/aquarack/aquarack-sensor-collector.zip"
	} else if *serviceType == "eks" {
		codeRepoIp = "cloud-repo.sdt.services"
		bwURL = "cloud-api-router.sdt.services"
		mqttURL = "ssl://avk03ee629rck-ats.iot.ap-northeast-2.amazonaws.com:8883"
		bwPort = 80
		serviceCode = "sdtcloud-live"
		fileUrl = "https://sdt-cloud-s3.s3.ap-northeast-2.amazonaws.com/blokworks-client/aquarack/aquarack-sensor-collector.zip"

	}

	deviceType, _ := SetConfig(*archType, mqttURL, serviceCode, *serviceType, *bwIP)

	if *organizationId == "0" || *assetCode == "0" {
		fmt.Printf("Please fill in parameter!!\b")
		os.Exit(0)
	}

	var dir string
	if *archType == "win" {
		dir = "C:/sdt/cert/"
		//SetAssetCode(*archType, *assetCode, *organization)
	} else {
		dir = "/etc/sdt/cert/"
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		os.Mkdir(dir, os.ModePerm)
	}

	accessKeyId, secretAccessKey := RegisterDevice(*assetCode, *organizationId, bwURL, bwPort)
	ConnectDevice(accessKeyId, secretAccessKey, *assetCode, *organizationId, bwURL, bwPort)
	ProvisioningDevice(*assetCode, *organizationId, dir, bwURL, bwPort, *serviceType)
	osInfo := GetOS()
	inNet, outNet := GetNetwork(*archType)
	gpuInfo := GetGPU()

	SendHwInfo(*assetCode, *organizationId, osInfo, gpuInfo, inNet, outNet, bwURL, bwPort)

	// 버전 선택
	fmt.Println(mqttURL, serviceCode, bwPort)
	//version := "v1.0.2"
	//fileUrl = fmt.Sprintf("%s/%s.zip", fileUrl, version)
	// Aquarack 인지 체크 후, 추가 에이전트 설치
	fmt.Printf("This device is %s\n", deviceType)
	if deviceType == "aquarack" {
		//fmt.Println(fileUrl)
		fmt.Println("Install aquarack agent...")
		sdtAquaRack.Deploy(fileUrl, codeRepoIp, codeRepoPort, deviceType)
	}
}
