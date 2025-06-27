// The Delete package handles functionalities to delete apps and virtual environments.
package delete

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	sdtType "main/src/cliType"
)

// These are the global variables used in the Delete package.
// - procLog: This is the Struct that defines the format of the Log.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// DeleteVenv function deletes the virtual environment installed on the device.
// All packages installed in the virtual environment are also deleted.
//
// Input:
//   - venvName: Name of the virtual environment.
func DeleteVenv(venvName string) {
	procLog.Warn.Printf("Delete %s venv.\n", venvName)
	targetEnv := fmt.Sprintf("/etc/sdt/venv/%s", venvName)
	removeErr := os.RemoveAll(targetEnv)
	if removeErr != nil {
		procLog.Error.Printf("%s venv deletion failed: %v\n", venvName, removeErr)
		fmt.Printf("%s venv deletion failed: %v\n", venvName, removeErr)
	}
}

// DeleteApp function deletes the app installed on the device.
// All data associated with the app will be deleted.
//
// Input:
//   - appName: Name of the app.
//   - appId: ID of the app.
func DeleteApp(appName string, appId string) {
	procLog.Warn.Printf("Delete %s app.\n", appName)
	// disable service
	disableCmd := fmt.Sprintf("systemctl disable %s", appName)
	cmd_run := exec.Command("sh", "-c", disableCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {

		procLog.Error.Printf("%s app's disable failed: %s\n", appName, stdout)
	}

	// stop service
	stopCmd := fmt.Sprintf("systemctl stop %s", appName)
	cmd_run = exec.Command("sh", "-c", stopCmd)
	stdout, cmd_err = cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("%s app's stop failed: %s\n", appName, stdout)
	}

	// remove service file
	removeCmd := fmt.Sprintf("rm /etc/systemd/system/%s.service", appName)
	cmd_run = exec.Command("sh", "-c", removeCmd)
	stdout, cmd_err = cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("%s app's svc file remove failed: %s\n", appName, stdout)
	}

	// remove appID in app.json
	var appRemoveCmd string
	if appId == "" {
		appRemoveCmd = fmt.Sprintf("rm -rf /etc/sdt/execute/%s", appName)
	} else {
		appRemoveCmd = fmt.Sprintf("rm -rf /usr/local/sdt/app/%s_%s", appName, appId)
	}
	cmd_run = exec.Command("sh", "-c", appRemoveCmd)
	stdout, cmd_err = cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Printf("%s app's file remove failed: %s\n", appName, stdout)
	}
}

// DeleteAppInfo function deletes information about the app to be deleted from the app config.
//
// Input:
//   - appName: Name of the app.
//
// Output:
//   - string: ID of the app.
func DeleteAppInfo(appName string) string {
	procLog.Warn.Printf("Delete %s app's info in app.json.\n", appName)
	var appId string = ""
	appInfoFile := "/etc/sdt/device.config/app.json"

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("Load app's file failed: %v\n", err)
		return appId
	}
	var jsonData, saveData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Unmarshal: %v\n", err)
		return appId
	}

	for _, val := range jsonData.AppInfoList {
		if val.AppName == appName {
			appId = val.AppId
			continue
		}
		saveData.AppInfoList = append(saveData.AppInfoList, val)
	}

	saveJson, _ := json.MarshalIndent(&saveData, "", "\t")
	err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
	if err != nil {
		procLog.Error.Printf("Delete app's file failed: %v\n", err)
		return appId
	}
	return appId
}
