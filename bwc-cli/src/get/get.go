// The Get package collects information about apps, agents, etc., running on the device.
// The information collected includes device agents, apps, virtual environments, app
// templates, and code repository usernames.
package get

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	sdtType "main/src/cliType"
	sdtUtil "main/src/util"
)

// These are the global variables used in the get package.
// - procLog: This is the Struct that defines the format of the log.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// GetRequirement function reads the requirements of an app. It reads the requirement.txt file
// which defines information about python packages.
//
// Input:
//   - pkgFile: Path to the requirement file.
//
// Output:
//   - string: Contents of the requirement file (returned as a string).
func GetRequirement(pkgFile string) string {
	procLog.Info.Printf("Get list of package.\n")
	content, err := ioutil.ReadFile(pkgFile)
	if err != nil {
		procLog.Error.Printf("Read failed: %v\n", err)
		return ""
	}
	procLog.Info.Printf("Successfully get list of package.\n")
	return string(content)
}

// GetTemplateOwner function collects the owner username of an app template.
//
// Input:
//   - bwUrl: URL of SDT Cloud.
//   - targetTemplate: Name of the app template.
//   - configData: Config information struct of BWC.
//
// Output:
//   - string: Owner username of the app template.
//   - error: Error message for the GetTemplateOwner command.
func GetTemplateOwner(bwUrl string, targetTemplate string, configData sdtType.ConfigInfo) (string, error) {
	procLog.Info.Printf("Get owner of template: %s\n", targetTemplate)
	templateInfo, err := GetTemplate(bwUrl, configData)
	if err != nil {
		procLog.Error.Printf("Failed get owner: %v\n", err)
		return "", err
	}
	for _, template := range templateInfo.Content {
		if template.Name == targetTemplate {
			procLog.Info.Printf("Successfully get owner of template.\n")
			return template.Owner.Username, nil
		}
	}
	procLog.Error.Printf("Failed get owner: %s owner not found\n", targetTemplate)
	return "", errors.New("Owner not found.")
}

// GetRepoOwner function retrieves the username of the code repository's owner.
//
// Input:
//   - bwUrl: URL of SDT Cloud.
//   - accessToken: Token value for calling SDT Cloud API.
//   - sdtcloudId: User ID in SDT Cloud.
//
// Output:
//   - sdtType.GiteaUser: Struct containing information about the code repository user.
func GetRepoOwner(bwUrl string, accessToken string, sdtcloudId string) sdtType.GiteaUser {
	procLog.Info.Printf("Get onwer of repository.\n")
	apiUrl := fmt.Sprintf("%s/stackbase/v1/gitea-manager/users/me", bwUrl)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		procLog.Error.Printf("Http not connected. : %v\n", err)
		fmt.Printf("Http not connected. : %v\n", err)
		os.Exit(1)
	}

	// req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("accept", "application/json")
	req.Header.Set("email", sdtcloudId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("Failed call api: %v\n", err)
		fmt.Printf("Failed call api: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	respBody, _ := ioutil.ReadAll(resp.Body)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])

	if statusValue >= 400 {
		// TO DO
		// - 에러 코드에 따른 에러 출력 문자 정리 필요
		procLog.Error.Printf("API call Error %d: %v\n", statusValue, err)
		fmt.Printf("API call Error %d: %v\n", statusValue, err)
		os.Exit(1)
	}

	// Get Info's template
	var ownerInfo sdtType.GiteaUser
	if err := json.Unmarshal(respBody, &ownerInfo); err != nil {
		procLog.Error.Printf("unmarshal Error: %v\n", err)
		fmt.Printf("unmarshal Error: %v\n", err)
		os.Exit(1)
	}
	procLog.Info.Printf("Successfully get onwer of repository: %s\n", ownerInfo)
	return ownerInfo
}

// GetTemplate function collects the list of app templates.
//
// Input:
//   - bwUrl: URL of SDT Cloud.
//   - configData: Struct containing configuration information for BWC.
//
// Output:
//   - sdtType.TemplateInfo: Struct containing information about the app templates.
//   - error: Error message for the GetTemplate command.
func GetTemplate(bwUrl string, configData sdtType.ConfigInfo) (sdtType.TemplateInfo, error) {
	procLog.Info.Printf("Get template.\n")
	var templateInfo sdtType.TemplateInfo
	apiUrl := fmt.Sprintf("%s/stackbase/v1/gitea-manager/repos/all", bwUrl)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		fmt.Printf("Http not connected. : %v\n", err)
		procLog.Error.Printf("Http not connected. : %v\n", err)
		os.Exit(1)
	}

	// req.Header.Add("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+configData.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("Failed call api: %v\n", err)
		fmt.Printf("Failed call api: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	respBody, err := ioutil.ReadAll(resp.Body)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])

	if statusValue >= 400 {
		fmt.Printf("[%d] Error in GetTemplate: %s\n", statusValue, respBody)
		fmt.Printf("Please check your account. Your account is not authorized.\n")
		fmt.Printf("Please login your device. The token may have expired.\n")
		procLog.Error.Printf("Error in GetTemplate[%d]: %s\n, ", statusValue, respBody)
		procLog.Warn.Printf("Please check your account. Your account is not authorized.\n")
		procLog.Warn.Printf("Please login your device. The token may have expired.\n")
		os.Exit(1)
	}

	// Get Info's template
	//procLog.Info.Printf("Request Body: %s\n", respBody)
	if err := json.Unmarshal(respBody, &templateInfo); err != nil {
		procLog.Error.Printf("unmarshal Error: %v\n", err)
		fmt.Printf("unmarshal Error: %v\n", err)
		os.Exit(1)
	}
	procLog.Info.Printf("Successfully get template.\n")
	//procLog.Info.Printf("Template: %s\n", templateInfo)
	return templateInfo, nil
}

// GetStatus function prints the SDT Cloud connection status of the device.
//
// Input:
//   - assetCode: Serial number of the device.
//   - organizationId: ID of the organization to which the device belongs.
//   - bwUrl: URL of SDT Cloud.
func GetStatus(assetCode string, organizationId string, bwURL string) {
	procLog.Info.Printf("Get status of device.\n")
	apiUrl := fmt.Sprintf("%s/assets/%s/status", bwURL, assetCode)

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		procLog.Error.Printf("Http not connected. : %v\n", err)
		fmt.Printf("Http not connected. : %v\n", err)
		os.Exit(1)
	}

	// req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OrganizationId", organizationId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		procLog.Error.Printf("Failed call api: %v\n", err)
		fmt.Printf("Failed call api: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	//request에 대한 응답
	respBody, err := ioutil.ReadAll(resp.Body)
	statusArr := strings.Split(resp.Status, " ")
	statusValue, _ := strconv.Atoi(statusArr[0])
	if statusValue >= 400 {
		procLog.Error.Printf("API call Error %d: %v\n", statusValue, err)
		fmt.Printf("API call Error %d: %v\n", statusValue, err)
		os.Exit(1)

	}
	result := map[string]interface{}{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		procLog.Error.Printf("unmarshal Error: %v\n", err)
		fmt.Printf("unmarshal Error: %v\n", err)
		os.Exit(1)
	}
	t := time.Unix(int64(result["stateUpdatedAt"].(float64)/1000), 0)
	resultTime := t.Format("2006-01-02 15:04:05")

	fmt.Printf("[%s] State:%s, Status:%s \n", resultTime, result["state"], result["status"])
	procLog.Info.Printf("Successfully get status of device.\n")
}

// GetInfoDevice function prints the information of the device.
//
// Input:
//   - deviceData: Struct containing information about the device.
func GetInfoDevice(deviceData sdtType.ConfigInfo) {
	// TODO
	// 	- Get modelname
	// fmt.Printf("Model Name: %s\n", deviceData.ModelName)
	fmt.Printf("Organzation Code: %s\n", deviceData.Organzation)
	fmt.Printf("Serial Code(Asset Code): %s\n", deviceData.AssetCode)
	fmt.Printf("Project Code: %s\n", deviceData.ProjectCode)
	fmt.Printf("Device Type: %s\n", deviceData.DeviceType)
	fmt.Printf("SDTCloud User: %s\n", deviceData.SdtcloudId)
}

// GetAppList function collects the list of deployed apps on the device.
//
// Output:
//   - sdtType.AppStatus: Struct containing information about the app status.
func GetAppList() []sdtType.AppStatus {
	procLog.Info.Printf("Get list of app.\n")
	var appStatus []sdtType.AppStatus
	appInfoFile := "/etc/sdt/device.config/app.json"

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("Failed load app's file: %v\n", err)
		return appStatus
	}
	var jsonData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Failed delete app's Unmarshal: %v\n", err)
		return appStatus
	}

	procLog.Info.Printf("Convert file to json.\n")
	for _, val := range jsonData.AppInfoList {
		status := "Running"
		if appPid, _ := sdtUtil.GetPid(val.AppName); appPid == -1 {
			status = "Not running"
		}
		app := sdtType.AppStatus{
			AppName: val.AppName,
			Status:  status,
			AppId:   val.AppId,
			AppVenv: val.AppVenv,
		}
		appStatus = append(appStatus, app)
	}
	procLog.Info.Printf("Successfully get list of app.\n")
	return appStatus
}

// GetAppId function retrieves the ID of an app.
//
// Input:
//   - appName: Name of the app.
//
// Output:
//   - String: ID of the app.
func GetAppId(appName string) string {
	procLog.Info.Printf("Get appID.\n")
	appInfoFile := "/etc/sdt/device.config/app.json"

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("Failed load app's file: %v\n", err)
		return ""
	}
	var jsonData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Failed delete app's Unmarshal: %v\n", err)
		return ""
	}

	for _, val := range jsonData.AppInfoList {
		if val.AppName == appName {
			procLog.Info.Printf("Successfully get appID.\n")
			return val.AppId
		}
	}
	procLog.Info.Printf("App not found.\n")
	return ""
}

// GetBWCList function collects the status information of agents on the device.
//
// Output:
//   - sdtType.AppStatus: Struct containing agent status information.
func GetBWCList() []sdtType.AppStatus {
	procLog.Info.Printf("Get BWC's process.\n")
	var bwcStatus []sdtType.AppStatus
	bwcList := []string{"device-control", "device-health", "device-heartbeat", "process-checker", "bwc-management"}

	for _, val := range bwcList {
		status := "Running"
		if appPid, _ := sdtUtil.GetPid(val); appPid == -1 {
			status = "Not running"
		}
		bwcInfo := sdtType.AppStatus{
			AppName: val,
			Status:  status,
		}
		bwcStatus = append(bwcStatus, bwcInfo)
	}
	procLog.Info.Printf("Successfully get BWC's process.\n")
	return bwcStatus
}

// GetVenvList function collects the list of virtual environments installed on the device.
//
// Output:
//   - []string: List of virtual environments.
func GetVenvList() []string {
	// TODO
	//   - 어떤 Bin 파일로 생성 됐는지 출력
	procLog.Info.Printf("Get list of venv.\n")
	var envList []string
	envDir, _ := ioutil.ReadDir("/etc/sdt/venv")

	for _, f := range envDir {
		if f.IsDir() {
			envList = append(envList, f.Name())
		}
	}
	procLog.Info.Printf("Successfully get list of venv.\n")
	return envList
}

// CheckExistVenv function checks whether a specific virtual environment exists.
//
// Input:
//   - targetVenv: Name of the virtual environment.
//
// Output:
//   - bool: True if the virtual environment exists, False otherwise.
func CheckExistVenv(targetVenv string) bool {
	procLog.Info.Printf("Check venv.\n")
	envDir, _ := ioutil.ReadDir("/etc/sdt/venv")
	for _, f := range envDir {
		if f.IsDir() {
			if f.Name() == targetVenv {
				procLog.Info.Printf("%s venv found.\n", targetVenv)
				return true
			}
		}
	}
	procLog.Info.Printf("%s venv not found.\n", targetVenv)
	return false
}

// CheckVenvUsed function collects information about the usage status of a virtual environment.
//
// Input:
//   - targetVenv: Name of the virtual environment.
//
// Output:
//   - string: Name of the app that is using the virtual environment.
//   - bool: True if the virtual environment is in use, False otherwise.
func CheckVenvUsed(targetVenv string) (string, bool) {
	procLog.Info.Printf("Check venv's used.\n")
	appInfoFile := "/etc/sdt/device.config/app.json"

	if _, err := os.Stat(appInfoFile); os.IsNotExist(err) {
		return "", false
	}

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("Failed load app's file: %v\n", err)
		fmt.Printf("Failed load app's file.\n")
		os.Exit(1)
	}

	var jsonData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Failed delete app's Unmarshal: %v\n", err)
		fmt.Printf("Failed delete app's Unmarshal.\n")
		os.Exit(1)
	}

	for _, val := range jsonData.AppInfoList {
		if targetVenv == val.AppVenv {
			procLog.Info.Printf("Check venv's used: %s used.\n", targetVenv)
			return val.AppName, true
		}
	}
	procLog.Info.Printf("Check venv's used: %s not used.\n", targetVenv)
	return "", false
}

// CheckExistApp checks if a specific app exists on the device.
//
// Input:
//   - targetApp: The name of the app to check.
//
// Output:
//   - bool: Indicates whether the app exists on the device.
func CheckExistApp(targetApp string) bool {
	procLog.Info.Printf("Check app exist.\n")
	appInfoFile := "/etc/sdt/device.config/app.json"

	if _, err := os.Stat(appInfoFile); os.IsNotExist(err) {
		return false
	}

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("Failed load app's file: %v\n", err)
		fmt.Printf("Failed load app's file.\n")
		os.Exit(1)
	}

	var jsonData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Failed delete app's Unmarshal: %v\n", err)
		fmt.Printf("Failed delete app's Unmarshal.\n")
		os.Exit(1)
	}

	for _, val := range jsonData.AppInfoList {
		if targetApp == val.AppName {
			procLog.Info.Printf("%s app found.\n", targetApp)
			return true
		}
	}
	procLog.Info.Printf("%s app not found.\n", targetApp)
	return false
}

// CheckExistBin checks if a specific bin (runtime) exists on the device.
//
// Input:
//   - targetBin: The name of the bin file to check.
//   - systemHome: The hostname of the device.
//
// Output:
//   - string: The name of the runtime on the device.
func CheckExistBin(targetBin string, systemHome string) string {
	procLog.Info.Printf("Check bin file.\n")
	var runtimeInfo string
	if targetBin == "miniconda3" {
		runtimeCmd := fmt.Sprintf("/home/%s/miniconda3/bin/python", systemHome)
		cmd := exec.Command(runtimeCmd, "--version")
		output, err := cmd.CombinedOutput()

		if err != nil {
			procLog.Error.Printf("Not found: Miniconda3 python3(%s)\n", runtimeCmd)
			fmt.Printf("Not found: Miniconda3 python3(%s)\n", runtimeCmd)
			os.Exit(1)
		}

		runtimeInfo = string(output)
		runtimeInfo = strings.ReplaceAll(runtimeInfo, " ", "-")
		runtimeInfo = strings.ReplaceAll(runtimeInfo, "\n", "")
		runtimeInfo = fmt.Sprintf("Miniconda3-%s", runtimeInfo)
	} else if targetBin == "python3" {
		runtimeCmd := "/usr/bin/python3"
		cmd := exec.Command(runtimeCmd, "--version")
		output, err := cmd.CombinedOutput()

		if err != nil {
			procLog.Error.Printf("Not found: Base python3(%s)\n", runtimeCmd)
			fmt.Printf("Not found: Base python3(%s)\n", runtimeCmd)
			os.Exit(1)
		}

		runtimeInfo = string(output)
		runtimeInfo = strings.ReplaceAll(runtimeInfo, " ", "-")
		runtimeInfo = strings.ReplaceAll(runtimeInfo, "\n", "")
	} else {
		fmt.Printf("Please check framework.yaml file.\n")
		fmt.Printf("You can use miniconda3 or python3.")
		os.Exit(1)
	}
	procLog.Info.Printf("%s bin file found.\n", targetBin)
	return runtimeInfo
}
