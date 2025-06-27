// The cli package processes user-requested commands. The functionalities handled by BWC-CLI include:
//   - create-app: Test app (execute test)
//   - create-venv: Create virtual environment
//   - deploy-app: Deploy app
//   - delete-venv: Delete virtual environment
//   - delete-app: Delete app
//   - get-app: Get app list
//   - get-venv: Get virtual environment list
//   - get-bwc: Get BWC agent list
//   - get-template: Get app template list
//   - init-app: Download app template
//   - info: Get device information
//   - logs-bwc: Check agent logs
//   - logs-app: Check app logs
//   - login: Login function
//   - status: Check device status
//   - update-venv: Update virtual environment
package cli

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh/terminal"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	sdtType "main/src/cliType"
	sdtCreate "main/src/create"
	sdtDelete "main/src/delete"
	sdtDeploy "main/src/deploy"
	sdtGet "main/src/get"
	sdtInit "main/src/init"
	sdtLogin "main/src/login"
	sdtLogs "main/src/logs"
	sdtMessage "main/src/message"
	sdtUpdate "main/src/update"
	sdtUtil "main/src/util"
)

// These are the global variables used in the cli package.
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

// RunBody is the main function of BWC-CLI. It processes combinations of commands received.
// App management functionalities send processing results as messages to SDT Cloud after handling operations.
// Commands are received in combinations of main functions and subtypes.
// Commands are processed in RunBody by combining the main function and subtype as "main-function-subtype".
// For example, if it's processed with the deploy main function and app subtype,
// it will be handled in RunBody as the "deploy-app" command.
//
// Input:
//   - cmd: Command type.
//   - archType: Device architecture.
//   - rootPath: Root path of BWC.
//   - appPath: Path of the app.
//   - bwcFramework: Framework Struct of the app.
//   - cliInfo: Command information Struct.
//   - giteaURL: URL of the code repository.
//   - bwURL: URL of SDT Cloud.
//   - giteaIP: IP address of the code repository.
func RunBody(cmd string,
	archType string,
	rootPath string,
	appPath string,
	bwcFramework sdtType.Framework,
	cliInfo sdtType.CliCmd,
	svcInfo sdtType.ControlService,
	// giteaURL string,
	// bwURL string,
	// giteaIP string,
	// minioURL string,
) {
	// os.Exit(1)
	// Set parameter
	var cliResult map[string]interface{}
	var configData sdtType.ConfigInfo
	var cliMessage string

	// Get Config's Info
	jsonFilePath := fmt.Sprintf("%s/device.config/config.json", rootPath)
	jsonFile, err := ioutil.ReadFile(jsonFilePath)
	if err != nil {
		fmt.Printf("Device's config file not found file Error: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(jsonFile, &configData)
	if err != nil {
		fmt.Printf("Device's config file Unmarshal Error: %v\n", err)
		os.Exit(1)
	}

	// Create RequestId
	newUUID := uuid.New()
	requestId := newUUID.String()

	switch cmd {
	case "login":
		sdtLogin.SaveLoginInfo(svcInfo.BwURL)
	case "init-app":
		// Get Ownername about target-template.
		ownerName, _ := sdtGet.GetTemplateOwner(svcInfo.BwURL, cliInfo.TemplateOption, configData)
		if ownerName == "" {
			fmt.Printf("%s's template not found \n", cliInfo.TemplateOption)
			os.Exit(1)
		}

		// Create app.
		sdtInit.CreateApp(cliInfo.NameOption, cliInfo.TemplateOption, svcInfo.GiteaURL, ownerName, bwcFramework.Spec.Env.HomeName)
		fmt.Printf("Create %s app in device.\n", cliInfo.NameOption)
	case "get-app":
		appList := sdtGet.GetAppList()
		fmt.Printf(" %-15s %-30s %-15s %-30s\n", "Status", "Name", "Venv", "AppID")
		// fmt.Printf("-----------------------------------------------\n")
		for _, val := range appList {
			fmt.Printf(" %-15s %-30s %-15s %-30s\n", val.Status, val.AppName, val.AppVenv, val.AppId)
		}
	case "get-venv":
		// Get env list
		envList := sdtGet.GetVenvList()
		fmt.Printf(" %-20s %-7s %-20s\n", "Name", "Used", "App")
		// fmt.Printf("-------------------------------\n")
		for _, val := range envList {
			appName, used := sdtGet.CheckVenvUsed(val)
			if used {
				fmt.Printf(" %-20s %-7s %-20s\n", val, "used", appName)
			} else {
				fmt.Printf(" %-20s %-7s %-20s\n", val, "no", appName)
			}
		}
	case "get-bwc":
		appList := sdtGet.GetBWCList()
		fmt.Printf(" %-15s %-30s\n", "Status", "Name")
		// fmt.Printf("-----------------------------------------------\n")
		for _, val := range appList {
			fmt.Printf(" %-15s %-30s\n", val.Status, val.AppName)
		}
	case "get-template":
		var templateType, ownerName string

		templateList, _ := sdtGet.GetTemplate(svcInfo.BwURL, configData)
		fmt.Printf(" %-30s %-20s %-30s\n", "Name", "Owner", "Type")
		// fmt.Printf("-----------------------------------------------\n")
		for _, val := range templateList.Content {
			ownerList := strings.Split(val.Owner.Username, ".")

			if len(ownerList) < 2 {
				ownerName = val.Owner.Username
			} else {
				//fmt.Println(val.Owner.Username)
				ownerName = ownerList[1]
			}
			if strings.Contains(val.Name, "template") {
				templateType = "base-template"
			} else {
				templateType = "app"
			}
			fmt.Printf(" %-30s %-20s %-30s\n", val.Name, ownerName, templateType)
		}
	case "deploy-app":
		// var username, password, giteaURL, localRepoPath, releaseTitle, repoUser string
		// Check exist about app
		if sdtGet.CheckExistApp(bwcFramework.Spec.AppName) {
			fmt.Printf("%s's app already exist.\n", bwcFramework.Spec.AppName)
			os.Exit(1)
		}

		// Check exist about venv
		//fmt.Printf("Checking if %s venv exists.\n", bwcFramework.Spec.Env.VirtualEnv)
		envList := sdtGet.GetVenvList()
		if !sdtUtil.Contains(envList, bwcFramework.Spec.Env.VirtualEnv) && bwcFramework.Spec.Env.VirtualEnv != "" {
			fmt.Printf("Venv not found: %s\n", bwcFramework.Spec.Env.VirtualEnv)
			fmt.Printf("Create %s venv in device\n", bwcFramework.Spec.Env.VirtualEnv)

			// Create venv.
			err := sdtCreate.DeployVenv(bwcFramework, cliInfo.DirOption, configData)
			if err != nil {
				fmt.Printf("Venv creation failed: %v\n", err)
				sdtDelete.DeleteVenv(bwcFramework.Spec.Env.VirtualEnv)
				os.Exit(1)
			}

			sdtCreate.InstallDefaultPkg(bwcFramework.Spec.Env.VirtualEnv, svcInfo.GiteaIP, svcInfo.GiteaURL, configData.DeviceType, configData.ServiceType)
			requirementStr := sdtGet.GetRequirement(bwcFramework.Spec.Env.Package)

			cliResult = map[string]interface{}{
				"name":        bwcFramework.Spec.Env.VirtualEnv,
				"requirement": requirementStr,
				"binFile":     bwcFramework.Spec.Env.RunTime,
			}
			cliMessage = fmt.Sprintf("%s's %s successed.", bwcFramework.Spec.Env.VirtualEnv, "create-venv")

			sdtMessage.SendResult("/etc/sdt", configData, "", cliMessage, nil,
				http.StatusOK, "venvCreate", "virtualEnv", requestId,
				-1, -1, nil, "", "", cliResult["binFile"].(string), cliResult["requirement"].(string), bwcFramework.Spec.Env.VirtualEnv,
			)

			fmt.Printf("Venv cration completed: %s \n", bwcFramework.Spec.Env.VirtualEnv)

			//os.Exit(1)
		}

		// Check upload in stackbase
		if cliInfo.UploadOption {
			// Check app is inference?
			//fmt.Println(bwcFramework)
			//if bwcFramework.Inference.WeightFile != "" {
			//	fmt.Printf("Upload inference app in code repository.\n")
			//	fmt.Printf("Warning: weight file not upload in code repository. Weight file upload in object-storage.\n")
			//}

			fmt.Printf("Upload app in code repository.\n")
			// Get Repo(=templateName)'s OwnerName.
			repoOwnerName, _ := sdtGet.GetTemplateOwner(svcInfo.BwURL, bwcFramework.Stackbase.RepoName, configData)
			// Get Device's ownerName by using your device's sdtcloudID in device's config.json).
			deviceOwnerName := sdtGet.GetRepoOwner(svcInfo.BwURL, configData.AccessToken, configData.SdtcloudId).Username

			if repoOwnerName == "" {
				repoOwnerName = deviceOwnerName
				procLog.Info.Printf("Frist push app in code repository: %s\n")
			} else if repoOwnerName != deviceOwnerName { // repo's ownerName differ device's ownerName.
				// Update password about repoOwnerName.
				fmt.Printf("Device's stackbase user differ app's repo user. Please input password.\n")
				fmt.Printf("Password for %s: ", strings.Split(repoOwnerName, ".")[1])
				pw, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
				configData.SdtcloudId = repoOwnerName
				configData.SdtcloudPw = string(pw)
			}
			sdtDeploy.UploadStackbase(bwcFramework, cliInfo.DirOption, svcInfo.GiteaURL, configData, repoOwnerName)
			fmt.Printf("Successfully upload.")
		}

		// Create AppId
		newUUID := uuid.New()
		appId := newUUID.String()

		// Get file's size
		appSize, err := sdtUtil.GetDirectorySize(cliInfo.DirOption)

		// Move App's file
		appDir := fmt.Sprintf("%s/%s_%s", appPath, bwcFramework.Spec.AppName, appId)
		sdtUtil.CopyDir(cliInfo.DirOption, appDir)

		// Save App's info in json
		sdtDeploy.SaveAppInfo(bwcFramework.Spec.AppName, appId, bwcFramework.Spec.Env.VirtualEnv, "systemd")

		// Create deamon service file and Move svc file in systemd directory
		// TODO env.bin과 bin.runtime 을 정리해야 함
		// TODO 다른 언어에 대한 앱 실행 로직 추가해야 함
		//if bwcFramework.Spec.Env.Bin == "python3" {
		//	sdtDeploy.CreatePythonService(appDir, bwcFramework)
		//} else if bwcFramework.Spec.Env.RunTime == "go" {
		//	sdtDeploy.CreateGoService(appDir, bwcFramework)
		//}
		//procLog.Info.Printf("App's lang is %s.\n", bwcFramework.Spec.Env.RunTime)

		// Start APP
		var appRepoPath string
		deployErr := sdtDeploy.DeployApp(bwcFramework, appDir, svcInfo.MinioURL)
		if deployErr != nil {
			sdtDelete.DeleteAppInfo(bwcFramework.Spec.AppName)
			//sdtDelete.DeleteApp(bwcFramework.Spec.AppName, appId)
			fmt.Printf("App deployment failed: %v\n", deployErr)
			os.Exit(1)
		}

		// Set App Repo
		appOwner, ownerErr := sdtGet.GetTemplateOwner(svcInfo.BwURL, bwcFramework.Stackbase.RepoName, configData)
		appRepoPath = fmt.Sprintf("%s/%s/%s:%s\n", svcInfo.GiteaURL,
			appOwner,
			bwcFramework.Stackbase.RepoName,
			bwcFramework.Stackbase.TagName,
		)
		if ownerErr != nil {
			sdtDelete.DeleteAppInfo(bwcFramework.Spec.AppName)
			sdtDelete.DeleteApp(bwcFramework.Spec.AppName, appId)
			fmt.Printf("App deployment failed: %v\n", ownerErr)
			os.Exit(1)
		}

		// Get PID
		pid, err := sdtUtil.GetPid(bwcFramework.Spec.AppName)
		if err != nil {
			sdtLogs.GetLogsApp(bwcFramework.Spec.AppName, appId)
			sdtDelete.DeleteAppInfo(bwcFramework.Spec.AppName)
			sdtDelete.DeleteApp(bwcFramework.Spec.AppName, appId)
			fmt.Printf("App deployment failed: %v\n", err)
			os.Exit(1)
		}

		cliResult = map[string]interface{}{
			"name": bwcFramework.Spec.AppName,
			"pid":  pid,
			"size": appSize,
		}
		cliMessage = fmt.Sprintf("%s's %s successed.", bwcFramework.Spec.AppName, cmd)

		sdtMessage.SendResult("/etc/sdt", configData, bwcFramework.Spec.AppName, cliMessage, nil,
			http.StatusOK, "appDeploy", "deploy", requestId,
			cliResult["pid"].(int), cliResult["size"].(int64), nil, appRepoPath, appId, "", "", bwcFramework.Spec.Env.VirtualEnv,
		)
		fmt.Printf("App deployment completed: %s\n", bwcFramework.Spec.AppName)

		// Send Message about app's config.
		// Get config
		procLog.Info.Printf("Get app's confing.\n")
		jsonResult := sdtDeploy.GetAppConfig(appId, bwcFramework.Spec.AppName, archType)
		cliMessage = fmt.Sprintf("Successfully get %s's config.", bwcFramework.Spec.AppName)

		sdtMessage.SendResult("/etc/sdt", configData, bwcFramework.Spec.AppName, cliMessage, nil,
			http.StatusOK, "get", "config", requestId,
			-1, -1, jsonResult, "", appId, "", "", "",
		)

	case "create-app":
		// Check exist about app
		if sdtGet.CheckExistApp(bwcFramework.Spec.AppName) && cliInfo.AppOption == "" {
			fmt.Printf("%s's app already exist.\n", bwcFramework.Spec.AppName)
			os.Exit(1)
		}

		// Check exist about venv
		//fmt.Printf("Checking if %s venv exists.\n", bwcFramework.Spec.Env.VirtualEnv)
		envList := sdtGet.GetVenvList()
		if !sdtUtil.Contains(envList, bwcFramework.Spec.Env.VirtualEnv) {
			fmt.Printf("Not found venv: %s\n", bwcFramework.Spec.Env.VirtualEnv)
			fmt.Printf("Create %s venv in device\n", bwcFramework.Spec.Env.VirtualEnv)

			// Create venv.
			sdtCreate.DeployVenv(bwcFramework, cliInfo.DirOption, configData)
			sdtCreate.InstallDefaultPkg(bwcFramework.Spec.Env.VirtualEnv, svcInfo.GiteaIP, svcInfo.GiteaURL, configData.DeviceType, configData.ServiceType)
			requirementStr := sdtGet.GetRequirement(bwcFramework.Spec.Env.Package)

			cliResult = map[string]interface{}{
				"name":        bwcFramework.Spec.Env.VirtualEnv,
				"requirement": requirementStr,
				"binFile":     bwcFramework.Spec.Env.RunTime,
			}
			cliMessage = fmt.Sprintf("%s's %s successed.", bwcFramework.Spec.Env.VirtualEnv, "create-venv")

			sdtMessage.SendResult("/etc/sdt", configData, "", cliMessage, nil,
				http.StatusOK, "venvCreate", "virtualEnv", requestId,
				-1, -1, nil, "", "", cliResult["binFile"].(string), cliResult["requirement"].(string), bwcFramework.Spec.Env.VirtualEnv,
			)

			fmt.Printf("Venv creation completed: %s\n", bwcFramework.Spec.Env.VirtualEnv)
			//os.Exit(1)
		}

		if cliInfo.AppOption == "restart" { // AppOption == restart
			// App setting
			sdtCreate.RestartApp(bwcFramework.Spec.AppName, cliInfo.DirOption)
			// App start
			sdtCreate.CreateApp(bwcFramework, "", "")
			fmt.Printf("App restart completed: %s\n", bwcFramework.Spec.AppName)
		} else { // default create function

			// Move App's file
			appDir := fmt.Sprintf("/etc/sdt/execute/%s", bwcFramework.Spec.AppName)
			sdtUtil.CopyDir(cliInfo.DirOption, appDir)

			// Save App's info in json
			sdtCreate.SaveAppInfo(bwcFramework.Spec.AppName, bwcFramework.Spec.Env.VirtualEnv)

			// Create deamon service file and Move svc file in systemd directory
			err = sdtCreate.CreateService(appDir, bwcFramework)

			// Start APP
			sdtCreate.CreateApp(bwcFramework, appDir, bwcFramework.Spec.AppType)

			fmt.Printf("App execution completed: %s\n", bwcFramework.Spec.AppName)
		}

	case "create-venv":
		// Check exit venv
		if sdtGet.CheckExistVenv(bwcFramework.Spec.Env.VirtualEnv) {
			fmt.Printf("%s's venv already exist.\n", bwcFramework.Spec.Env.VirtualEnv)
			os.Exit(1)
		}

		// Create Venv
		err := sdtCreate.DeployVenv(bwcFramework, cliInfo.DirOption, configData)
		if err != nil {
			fmt.Printf("Venv creation failed: %v\n", err)
			sdtDelete.DeleteVenv(bwcFramework.Spec.Env.VirtualEnv)
			os.Exit(1)
		}
		// Install Default Package - sdtcloudpubsub, sdtclouds3
		sdtCreate.InstallDefaultPkg(bwcFramework.Spec.Env.VirtualEnv, svcInfo.GiteaIP, svcInfo.GiteaURL, configData.DeviceType, configData.ServiceType)
		// Get Requirement
		requirementStr := sdtGet.GetRequirement(bwcFramework.Spec.Env.Package)

		cliResult = map[string]interface{}{
			"name":        bwcFramework.Spec.Env.VirtualEnv,
			"requirement": requirementStr,
			"binFile":     bwcFramework.Spec.Env.RunTime,
		}
		cliMessage = fmt.Sprintf("%s's %s successed.", bwcFramework.Spec.Env.VirtualEnv, cmd)

		sdtMessage.SendResult("/etc/sdt", configData, "", cliMessage, nil,
			http.StatusOK, "venvCreate", "virtualEnv", requestId,
			-1, -1, nil, "", "", cliResult["binFile"].(string), cliResult["requirement"].(string), bwcFramework.Spec.Env.VirtualEnv,
		)
		fmt.Printf("Venv creation completed: %s\n", bwcFramework.Spec.Env.VirtualEnv)

	case "update-venv":
		// Check exist venv
		if !sdtGet.CheckExistVenv(bwcFramework.Spec.Env.VirtualEnv) {
			fmt.Printf("Venv not found: %s\n", bwcFramework.Spec.Env.VirtualEnv)
			os.Exit(1)
		}

		// Install Default Package - sdtcloudpubsub, sdtclouds3
		sdtCreate.InstallDefaultPkg(bwcFramework.Spec.Env.VirtualEnv, svcInfo.GiteaIP, svcInfo.GiteaURL, configData.DeviceType, configData.ServiceType)

		binFile := sdtGet.CheckExistBin(bwcFramework.Spec.Env.Bin, bwcFramework.Spec.Env.HomeName)

		envHome := "/etc/sdt/venv"
		sdtUpdate.UpdateVenv(bwcFramework, envHome, cliInfo.DirOption)

		// Get Requirement
		var requirementStr string
		pkgFile := bwcFramework.Spec.Env.Package

		content, err := ioutil.ReadFile(pkgFile)
		if err != nil {
			fmt.Printf("Venv update failed: %v\n", err)
			os.Exit(1)
		}
		requirementStr = string(content)

		cliResult = map[string]interface{}{
			"name":        bwcFramework.Spec.Env.VirtualEnv,
			"requirement": requirementStr,
			"binFile":     binFile,
		}
		cliMessage = fmt.Sprintf("%s's %s successed.", bwcFramework.Spec.Env.VirtualEnv, cmd)

		sdtMessage.SendResult("/etc/sdt", configData, "", cliMessage, nil,
			http.StatusOK, "venvUpdate", "virtualEnv", requestId,
			-1, -1, nil, "", "", cliResult["binFile"].(string), cliResult["requirement"].(string), bwcFramework.Spec.Env.VirtualEnv,
		)
		fmt.Printf("Venv update completed: %s\n", bwcFramework.Spec.Env.VirtualEnv)

	case "delete-venv":
		// Check exist venv
		if !sdtGet.CheckExistVenv(cliInfo.NameOption) {
			fmt.Printf("Venv not found: %s\n", cliInfo.NameOption)
			os.Exit(1)
		}

		// Check venv used
		appName, used := sdtGet.CheckVenvUsed(cliInfo.NameOption)
		if used {
			fmt.Printf("%s venv used by %s app.\n", cliInfo.NameOption, appName)
			fmt.Printf("If you want to delete venv, delete %s app.\n", appName)
			os.Exit(1)
		}

		sdtDelete.DeleteVenv(cliInfo.NameOption)
		statusCode := 200
		cliMessage = fmt.Sprintf("%s's %s successed.", cliInfo.NameOption, cmd)

		sdtMessage.SendResult("/etc/sdt", configData, "", cliMessage, nil,
			statusCode, "venvDelete", "virtualEnv", requestId,
			-1, -1, nil, "", "", "", "", cliInfo.NameOption,
		)
		fmt.Printf("Venv deletion completed: %s \n", cliInfo.NameOption)
	case "delete-app":
		// Check exist app.
		if !sdtGet.CheckExistApp(cliInfo.NameOption) {
			fmt.Printf("App not found: %s\n", cliInfo.NameOption)
			os.Exit(1)
		}

		// Delete app's info and get appId
		appId := sdtDelete.DeleteAppInfo(cliInfo.NameOption)
		//fmt.Println("APPID: [", appId, "]")

		// Stop App
		sdtDelete.DeleteApp(cliInfo.NameOption, appId)

		// If app executed, not send message
		if appId == "" {
			fmt.Printf("App deletion completed: %s\n", cliInfo.NameOption)
			os.Exit(1)
		}
		// Set result
		cliResult = map[string]interface{}{
			"name": cliInfo.NameOption,
			"pid":  -1,
			"size": int64(-1),
		}
		cliMessage = fmt.Sprintf("%s's %s successed.", cliInfo.NameOption, cmd)
		statusCode := 200

		sdtMessage.SendResult("/etc/sdt", configData, cliInfo.NameOption, cliMessage, nil,
			statusCode, "appDelete", "deploy", requestId,
			cliResult["pid"].(int), cliResult["size"].(int64), nil, "", appId, "", "", "",
		)
		fmt.Printf("App deletion completed: %s\n", cliInfo.NameOption)
	case "status":
		sdtGet.GetStatus(configData.AssetCode, configData.Organzation, svcInfo.BwURL)
	case "info":
		sdtGet.GetInfoDevice(configData)
	case "upload":
		fmt.Printf("Upload app in code repository.\n")
		// Get Repo(=templateName)'s OwnerName.
		repoOwnerName, _ := sdtGet.GetTemplateOwner(svcInfo.BwURL, bwcFramework.Stackbase.RepoName, configData)
		// Get Device's ownerName by using your device's sdtcloudID in device's config.json).
		deviceOwnerName := sdtGet.GetRepoOwner(svcInfo.BwURL, configData.AccessToken, configData.SdtcloudId).Username

		if repoOwnerName == "" {
			repoOwnerName = deviceOwnerName
			procLog.Info.Printf("Frist push app in code repository: %s\n")
		} else if repoOwnerName != deviceOwnerName { // repo's ownerName differ device's ownerName.
			// Update password about repoOwnerName.
			fmt.Printf("Device's stackbase user differ app's repo user. Please input password.\n")
			fmt.Printf("Password for %s: ", strings.Split(repoOwnerName, ".")[1])
			pw, _ := terminal.ReadPassword(int(os.Stdin.Fd()))
			configData.SdtcloudId = repoOwnerName
			configData.SdtcloudPw = string(pw)
		}
		sdtDeploy.UploadStackbase(bwcFramework, cliInfo.DirOption, svcInfo.GiteaURL, configData, repoOwnerName)
		fmt.Printf("Successfully upload.")
	case "logs-bwc":
		if cliInfo.TailOption {
			sdtLogs.GetLogsTail(cliInfo.NameOption)
		} else {
			sdtLogs.GetLogs(cliInfo.NameOption, cliInfo.LineOption)
		}
	case "logs-app":
		if !sdtGet.CheckExistApp(cliInfo.NameOption) {
			fmt.Printf("App not found: %s\n", cliInfo.NameOption)
			os.Exit(1)
		}
		appId := sdtGet.GetAppId(cliInfo.NameOption)
		sdtLogs.GetLogsApp(cliInfo.NameOption, appId)
	case "wol":
		sdtDeploy.WolTest(cliInfo.NameOption)
	}

	// fmt.Println(cliResult)

}
