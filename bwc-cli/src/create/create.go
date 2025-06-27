// The create package handles functionalities for app execution testing and virtual
// environment creation. App execution testing is a test function that verifies
// the operation of the app in the device's SDT Cloud environment, and the app
// is created in '/etc/sdt/execute'. Virtual environments are created in the
// '/etc/sdt/venv' path.
package create

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	sdtType "main/src/cliType"
	sdtGet "main/src/get"
	sdtUtil "main/src/util"
)

// These are the global variables used in the create package.
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

// The DeployVenv function is responsible for creating a Python virtual environment on the device.
// Virtual environments are spaces used by Python apps to isolate package dependencies.
// They are utilized to manage dependencies for Python apps specifically.
//
// Input:
//   - bwcFramework: Struct containing information about the app's framework.
//   - dirPath: Path where the user executed BWC-CLI.
//   - configData: Struct containing configuration information for BWC.
//
// Output:
//   - error: Error message for the DeployVenv command.
func DeployVenv(bwcFramework sdtType.Framework, dirPath string, configData sdtType.ConfigInfo) error {
	// TODO
	//  만약 유저가 사용하고 싶은 python3 path를 정하고 싶다면...?
	//  base -> /usr/bin/python
	//  etc -> 다른 python 파일들!!
	procLog.Info.Printf("Venv create...\n")
	var createEnvCmd, envPath, pkgCmd, pkgFileName, homePath, runTimeVersion string

	// Set Variable
	if bwcFramework.Spec.Env.HomeName == "root" {
		homePath = "/root"
	} else {
		homePath = fmt.Sprintf("/home/%s", bwcFramework.Spec.Env.HomeName)
	}

	// Set Runtime Version
	runTimeVersion = strings.Replace(bwcFramework.Spec.Env.RunTime, "python", "", -1)

	// Check VenvHome
	envHome := "/etc/sdt/venv"
	if _, err := os.Stat(envHome); os.IsNotExist(err) {
		os.Mkdir(envHome, os.ModePerm)
	}

	envPath = fmt.Sprintf("%s/%s", envHome, bwcFramework.Spec.Env.VirtualEnv)
	pkgFileName = fmt.Sprintf("%s/%s", dirPath, bwcFramework.Spec.Env.Package)
	procLog.Info.Printf("Venv spec: device= %s, homePath=%s, envPath=%s, pkgFileName=%s\n", configData.DeviceType, homePath, envPath, pkgFileName)

	if configData.DeviceType == "nodeq" {
		fmt.Printf("Cannot create virtual env in nodeQ.\n")
		return errors.New("Cannot create virtual env in nodeQ.")
	} else if bwcFramework.Spec.Env.VirtualEnv == "base" {
		fmt.Printf("Already create 'base' venv.\n")
		os.Exit(1)
	} else {
		createEnvCmd = fmt.Sprintf("%s/miniconda3/bin/conda create -p %s python=%s -y", homePath, envPath, runTimeVersion)
		cmd_run := exec.Command("sh", "-c", createEnvCmd)
		result, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			//fmt.Printf("Create V-ENV Error: [%v] %s\n", cmd_err, result)
			procLog.Error.Printf("Venv creation failed: %v\n", cmd_err)
			procLog.Error.Printf("Venv creation's result: %s\n", result)
			return cmd_err
		}
		procLog.Info.Printf("Venv created: %s\n", bwcFramework.Spec.Env.VirtualEnv)
	}

	// install pkg
	// TODO: Error 내용 출력 수정(출력 내용 그대로 출력되도록)
	procLog.Info.Printf("Install package in venv.\n")
	pkgCmd = fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	cmd_run := exec.Command("sh", "-c", pkgCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Printf("Install package error: %s\n", stdout)
		return cmd_err
	}

	// Copy requirment file in env path.
	destPath := fmt.Sprintf("%s/requirements.txt", envPath)
	sdtUtil.CopyFile(pkgFileName, destPath)

	procLog.Info.Printf("Venv's package installed.\n")

	return nil
}

func DeployVenv_Backup(bwcFramework sdtType.Framework, dirPath string) string {
	// TO DO
	//  만약 유저가 사용하고 싶은 python3 path를 정하고 싶다면...?
	//  base -> /usr/bin/python
	//  etc -> 다른 python 파일들!!
	var createEnvCmd, envPath, pkgCmd, pkgFileName string

	// Check bin file
	binFile := sdtGet.CheckExistBin(bwcFramework.Spec.Env.Bin, bwcFramework.Spec.Env.HomeName)

	// Check VenvHome
	envHome := "/etc/sdt/venv"
	if _, err := os.Stat(envHome); os.IsNotExist(err) {
		os.Mkdir(envHome, os.ModePerm)
	}

	envPath = fmt.Sprintf("%s/%s", envHome, bwcFramework.Spec.Env.VirtualEnv)
	pkgFileName = fmt.Sprintf("%s/%s", dirPath, bwcFramework.Spec.Env.Package)

	if bwcFramework.Spec.Env.VirtualEnv == "base" {
		if bwcFramework.Spec.Env.Bin == "python3" {
			pkgCmd = fmt.Sprintf("/usr/bin/pip3 install -r %s/%s", dirPath, bwcFramework.Spec.Env.Package)
		} else if bwcFramework.Spec.Env.Bin == "miniconda3" {
			if bwcFramework.Spec.Env.HomeName == "root" {
				pkgCmd = fmt.Sprintf("/root/miniconda3/bin/pip3 install -r %s/%s", dirPath, bwcFramework.Spec.Env.Package)
			} else {
				pkgCmd = fmt.Sprintf("/home/%s/miniconda3/bin/pip3 install -r %s/%s", bwcFramework.Spec.Env.HomeName, dirPath, bwcFramework.Spec.Env.Package)
			}
		}
	} else {
		if bwcFramework.Spec.Env.Bin == "python3" {
			createEnvCmd = fmt.Sprintf("/usr/bin/python3 -m venv %s", envPath)
		} else if bwcFramework.Spec.Env.Bin == "miniconda3" {
			if bwcFramework.Spec.Env.HomeName == "root" {
				createEnvCmd = fmt.Sprintf("/root/miniconda3/bin/python -m venv %s", envPath)
			} else {
				createEnvCmd = fmt.Sprintf("/home/%s/miniconda3/bin/python -m venv %s", bwcFramework.Spec.Env.HomeName, envPath)
			}
		} else {
			fmt.Printf("%s python bin file Not found\n", bwcFramework.Spec.Env.Bin)
		}
		cmd_run := exec.Command("sh", "-c", createEnvCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			fmt.Printf("Create V-ENV Error: %s\n", stdout)
			os.Exit(1)
		}
		fmt.Printf("Create V-ENV.\n")
		pkgCmd = fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	}

	// install pkg
	//pkgFileName = fmt.Sprintf("%s/%s", dirPath, bwcFramework.Spec.Env.Package)
	//pkgCmd = fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	//fmt.Println(pkgCmd)
	cmd_run := exec.Command("sh", "-c", pkgCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Printf("Install package Error: %s\n", stdout)
		os.Exit(1)
	}

	// Copy requirment file in env path.
	if bwcFramework.Spec.Env.VirtualEnv == "base" {
		return binFile
	}
	destPath := fmt.Sprintf("%s/requirement.txt", envPath)
	sdtUtil.CopyFile(pkgFileName, destPath)

	fmt.Printf("Installed V-ENV's package.\n")
	return binFile
}

// SaveAppInfo function saves the app's metadata to BWC's app config file.
// The app config file records metadata of apps deployed on the device.
// The saved information includes the app's virtual environment and app name.
//
// Input:
//   - appName: Name of the app.
//   - appVenv: Virtual environment of the app.
//
// Output:
//   - error: Error message for the SaveAppInfo command.
func SaveAppInfo(appName string, appVenv string) error {
	procLog.Info.Printf("Record app's info in app.json\n")
	appInfoFile := "/etc/sdt/device.config/app.json"

	// check app's info file
	if _, err := os.Stat(appInfoFile); os.IsNotExist(err) {
		//fmt.Println("Not Found.")
		appInfo := []sdtType.AppInfo{
			{
				AppName: appName,
				AppVenv: appVenv,
			},
		}
		jsonData := sdtType.AppConfig{
			AppInfoList: appInfo,
		}
		saveJson, err := json.MarshalIndent(jsonData, "", "\t")
		if err != nil {
			procLog.Error.Printf("Failed save app's Marshal: %v\n", err)
			return err
		}

		err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("Failed save app's info: %v\n", err)
			return err
		}
	} else {
		jsonFile, err := ioutil.ReadFile(appInfoFile)
		if err != nil {
			procLog.Error.Printf("Failed load app's file: %v\n", err)
			return err
		}
		var jsonData sdtType.AppConfig
		err = json.Unmarshal(jsonFile, &jsonData)
		if err != nil {
			procLog.Error.Printf("Failed save app's Unmarshal: %v\n", err)
			return err
		}

		newApp := sdtType.AppInfo{
			AppName: appName,
			AppVenv: appVenv,
		}

		jsonData.AppInfoList = append(jsonData.AppInfoList, newApp)
		saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
		err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("Failed save app's file: %v\n", err)
			return err
		}
	}
	procLog.Info.Printf("Record app's info completed in app.json\n")
	return nil
}

// CreateService function creates a systemd service file (.service) to run the app.
//
// Input:
//   - appDir: Path of the app.
//   - bwcFramework: Struct containing information about the app's framework.
//
// Output:
//   - error: Error message for the CreateService command.
func CreateService(appDir string, bwcFramework sdtType.Framework) error {
	procLog.Info.Printf("Create app's svc file.\n")
	appName := bwcFramework.Spec.AppName
	appVenv := bwcFramework.Spec.Env.VirtualEnv
	runCmd := bwcFramework.Spec.RunFile
	// Specify the file name and path
	filePath := fmt.Sprintf("%s/%s.service", appDir, appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		procLog.Error.Printf("Error creating service file : %v\n", err)
		return err
	}
	defer file.Close()

	// Write content to the file
	content := fmt.Sprintf(`[Unit]
Description=%s

[Service]
WorkingDirectory=%s
Environment=PATH=/etc/sdt/venv/%s/bin:$PATH
ExecStart=/etc/sdt/venv/%s/bin/python %s
Restart=always
RestartSec=10
StandardOutput=file:/%s/app.log
StandardError=file:/%s/app-error.log

[Install]
WantedBy=multi-user.target
	`, appName, appDir, appVenv, appVenv, runCmd, appDir, appDir)
	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Printf("Error writing to the file: %v\n", err)
		return err
	}

	svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	err = sdtUtil.CopyFile(filePath, svcFile)

	if err != nil {
		procLog.Error.Printf("Error svc file copy to systemd: %v\n", err)
		return err
	}
	procLog.Info.Printf("App's svc file creation completed.\n")
	return nil
}

// CreateApp function is responsible for running the app. The app is managed and executed using Systemd.
//
// Input:
//   - bwcFramework: Struct containing information about the app's framework.
//
// Output:
//   - error: Error message for the CreateApp command.
func CreateApp(bwcFramework sdtType.Framework, appDir string, appType string) error {

	if appType == "inference" {
		err := sdtUtil.CreateInferenceDir(appDir)
		if err != nil {
			procLog.Error.Printf("Failed creating inference directory.\n")
			return err
		}
	}
	procLog.Info.Printf("[systemd] Start app servie.\n")
	startCmd := fmt.Sprintf("systemctl start %s", bwcFramework.Spec.AppName)
	cmd_run := exec.Command("sh", "-c", startCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("App deployment failed: %v\n", cmd_err)
		procLog.Error.Printf("App deployment failed - Result: %s\n", string(stdout))
		return cmd_err
	}
	procLog.Info.Printf("[systemd] Successfully start app servie.\n")
	return nil
}

// InstallDefaultPkg function installs default packages provided by SDT Cloud into the virtual environment.
// Default packages provided by SDT Cloud include MQTT, S3, and MQTTforNodeQ.
// These packages enable communication with respective services.
//
// Input:
//   - venvName: Path of the virtual environment.
//   - giteaIP: IP address of the code repository.
//   - giteaUrl: Port value of the code repository.
//   - deviceType: Type of the device. (ECN, NodeQ)
func InstallDefaultPkg(venvName string, giteaIP string, giteaUrl string, deviceType string, serviceType string) {
	procLog.Info.Printf("Install SDT's package in %s.\n", venvName)
	procLog.Info.Printf("Device is %s.\n", deviceType)
	var pkgCmd, pkgLink string
	var pkgList []string

	pkgLink = fmt.Sprintf("/etc/sdt/venv/%s/bin/pip3 install --trusted-host %s --index-url %s/api/packages/app.manager/pypi/simple/", venvName, giteaIP, giteaUrl)

	if deviceType == "nodeq" {
		pkgList = []string{"sdtcloudnodeqmqtt", "sdtcloud"}
	} else if deviceType == "ecn" {
		pkgList = []string{"sdtcloudpubsub", "sdtclouds3", "sdtcloud"}
	} else {
		procLog.Warn.Printf("%s type not found.\n", deviceType)
		fmt.Printf("[Warnning] %s type not found.\n", deviceType)
		return
	}

	if serviceType == "onprem" {
		procLog.Info.Printf("This service type is %s, so download onprem's pkg..\n", serviceType)
		pkgList = []string{"sdtcloudonprem"}
		// local cmd
		// /etc/sdt/venv/mqtt-test/bin/pip install --trusted-host 192.168.1.232 --index-url http://192.168.1.232:8081/repository/pypi-group/simple sdtcloudonprem
	}

	procLog.Info.Printf("Execution install cmd.\n")
	for _, pkgName := range pkgList {
		procLog.Info.Printf("Default pkg install... [%s]\n", pkgName)
		pkgCmd = fmt.Sprintf("%s %s", pkgLink, pkgName)
		cmd_run := exec.Command("sh", "-c", pkgCmd)
		sdtout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Warn.Printf("SDT's package install failed: %v\n", cmd_err)
			procLog.Warn.Printf("SDT's package install failed - Result: %s\n", sdtout)
			fmt.Printf("[Warnning] SDT's package install failed: [%v] %s \n", cmd_err, sdtout)
			return
		}
	}
	procLog.Info.Printf("Successfully install SDT's package in %s.\n", venvName)
}

// RestartApp function restarts the app.
//
// Input:
//   - appName: Name of the app.
//   - dirOption: Directory path of the app to deploy.
func RestartApp(appName string, dirOption string) {
	// stop service
	procLog.Info.Printf("%s app's stop\n", appName)
	stopCmd := fmt.Sprintf("systemctl stop %s", appName)
	cmd_run := exec.Command("sh", "-c", stopCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("%s app's stop failed: %s\n", appName, stdout)
	}

	// remove app in execution
	procLog.Warn.Printf("%s app delete in execution.\n", appName)
	appRemoveCmd := fmt.Sprintf("rm -rf /etc/sdt/execute/%s", appName)
	cmd_run = exec.Command("sh", "-c", appRemoveCmd)
	stdout, cmd_err = cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Printf("%s app's file remove failed: %s\n", appName, stdout)
	}

	// copy app in execution
	procLog.Info.Printf("%s app copy to execution.\n", appName)
	appDir := fmt.Sprintf("/etc/sdt/execute/%s", appName)
	sdtUtil.CopyDir(dirOption, appDir)
}
