// The control package is the main package that processes control commands
// requested from the cloud. Control supports functionalities such as device
// command control, app management, and device reboot. Control communicates
// messages with the cloud via MQTT. It operates in both Windows and Linux environments.
package control

import (
	"encoding/json"
	"errors"
	"fmt"
	dockerCli "github.com/docker/docker/client"
	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
	sdtDocker "main/src/docker"
	sdtModel "main/src/model"
	"net/http"
	"os/exec"
	"strings"

	sdtConfig "main/src/config"
	sdtType "main/src/controlType"
	sdtDeploy "main/src/deploy"
	//sdtModel "main/src/model"
)

// Global variables used in the control package:
//   - procLog: Struct defining the format of logs.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// Control processes control commands received from the cloud based on their types.
// Supported commands include bash, systemd, docker, and app management (deploy, delete, getConfig, get pid).
// Control executes the commands and sends the processing status to SDT Cloud.
//
// Input:
//   - svcInfo: Information struct for the Control service.
//   - configData: Information struct for BWC Config.
//   - m: Command information struct received from the cloud.
//   - dockerClient: Client variable to control Docker.
//   - archType: Device architecture.
//   - homeUser: Device hostname.
//
// Output:
//   - map[string]interface{}: Message with the processing status of the control command to send back to the cloud.
//   - map[string]interface{}: App's config values to send to the cloud.
func Control(svcInfo sdtType.ControlService,
	configData sdtType.ConfigInfo,
	m sdtType.CmdControl,
	dockerClient *dockerCli.Client,
	archType string,
	homeUser string,
	cli mqttCli.Client) (sdtType.ResultMsg, sdtType.ResultMsg) {
	var result, configResult sdtType.ResultMsg
	var statusCode int

	json_data, _ := json.Marshal(m.CmdInfo)

	switch m.CmdType {
	case "bash":
		var bashData sdtType.CmdBash
		var bashResult string
		var cmdErr error
		var stdout []byte
		procLog.Info.Printf("[BASH] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &bashData)

		if err != nil {
			procLog.Error.Printf("[BASH] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if !CheckBash(bashData) {
			result = FormError(configData.AssetCode, m.RequestId, m.CmdType, bashData.Cmd)
			procLog.Error.Printf("[BASH] Format Error: %s\n", result)
			break
		}

		if bashData.Cmd == "reboot" {
			// TODO: Windows용 재부팅 명령어 추가 필요
			_, _ = sdtConfig.Rebooting("rebooting", m.RequestId)
		}

		if archType == "win" {
			cmd := fmt.Sprintf("%s", bashData.Cmd)
			cmdRun := exec.Command(cmd)
			stdout, cmdErr = cmdRun.CombinedOutput()

			// transform utf16 -> utf8
			newOut, _, _ := transform.String(korean.EUCKR.NewDecoder(), string(stdout))
			newOut = strings.Replace(newOut, "\r\n", "\n", -1)
			bashResult = newOut
		} else {
			cmd := fmt.Sprintf("%s", bashData.Cmd)
			cmdRun := exec.Command("sh", "-c", cmd)
			stdout, cmdErr = cmdRun.CombinedOutput()
			bashResult = string(stdout)
		}

		if cmdErr != nil {
			procLog.Error.Printf("[BASH] Error1: %s\n", bashResult)
			procLog.Error.Printf("[BASH] Error2: %v\n", cmdErr)
			statusCode = http.StatusBadRequest
		} else {
			statusCode = http.StatusOK
		}

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, bashData.Cmd, bashResult)
		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}

	case "systemd":
		// TODO: Windows 장비는 Systmed 사용 불가능 -> 2024.12.02 완료
		var systemdData sdtType.CmdSystemd
		var cmdErr error
		var stdout []byte
		var systemdResult string
		procLog.Info.Printf("[SYSTEMD] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &systemdData)
		if err != nil {
			procLog.Error.Printf("[SYSTEMD] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if !CheckSystemd(systemdData) {
			result = FormError(configData.AssetCode, m.RequestId, m.CmdType, systemdData.Cmd)
			procLog.Error.Printf("[SYSTEMD] Format Error: %s\n", result)
			break
		}

		if archType == "win" {
			cmdErr = errors.New("Cannot execute systmed cmd. OS is windows. So, you cannot execute systemd cmd.")
			statusCode = http.StatusBadRequest
		} else {
			cmd := fmt.Sprintf("systemctl %s %s", systemdData.Cmd, systemdData.Service)
			cmdRun := exec.Command("sh", "-c", cmd)
			stdout, cmdErr = cmdRun.CombinedOutput()
			systemdResult = string(stdout)
			if cmdErr != nil {
				procLog.Error.Printf("[SYSTEMD] Error1: %s\n", systemdResult)
				procLog.Error.Printf("[SYSTEMD] Error2: %v\n", cmdErr)
				statusCode = http.StatusBadRequest
			} else {
				statusCode = http.StatusOK
			}
		}

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, systemdData.Cmd, systemdResult)
		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}

	case "docker":
		var dockerData sdtType.CmdDocker
		var cmdErr error
		var deployMessage string = ""

		procLog.Info.Printf("[DOCKER] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &dockerData)
		if err != nil {
			procLog.Error.Printf("[DOCKER] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if !CheckDocker(m.SubCmdType, dockerData) {
			result = FormError(configData.AssetCode, m.RequestId, m.CmdType, m.SubCmdType)
			procLog.Error.Printf("[DOCKER] Format Error: %s\n", result)
			break
		}

		if m.SubCmdType == "appDeploy" {
			cmdErr, statusCode = sdtDocker.CreateContainer(dockerClient, dockerData.Image, dockerData.AppName, dockerData.AppId, svcInfo.RootPath, nil, nil)

		} else if m.SubCmdType == "appDelete" {
			cmdErr, statusCode = sdtDocker.DeleteContainer(dockerClient, dockerData.AppName, dockerData.AppId, svcInfo.RootPath)
		} else if m.SubCmdType == "appStart" {
			cmdErr, statusCode = sdtDocker.StartContainer(dockerClient, dockerData.AppName, dockerData.AppId)
		} else if m.SubCmdType == "appStop" {
			cmdErr, statusCode = sdtDocker.StopContainer(dockerClient, dockerData.AppName, dockerData.AppId)
		}

		if cmdErr != nil {
			procLog.Error.Printf("[DOCKER] Error: %v\n", cmdErr)
			deployMessage = fmt.Sprintf("%s's %s failed.", dockerData.AppName, m.SubCmdType)
		} else {
			deployMessage = fmt.Sprintf("%s's %s successed.", dockerData.AppName, m.SubCmdType)
		}

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, m.SubCmdType, deployMessage)
		cmdResult.AppId = dockerData.AppId
		cmdResult.AppName = dockerData.AppName

		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}

	case "virtualEnv":
		var venvData sdtType.CmdVenv
		var cmdErr error
		var stdout string

		procLog.Info.Printf("[Venv] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &venvData)
		if err != nil {
			procLog.Error.Printf("[Venv] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if !CheckVenv(m.SubCmdType, venvData) {
			result = FormError(configData.AssetCode, m.RequestId, m.CmdType, m.SubCmdType)
			procLog.Error.Printf("[Venv] Format Error: %s\n", result)
			break
		}

		if m.SubCmdType == "venvCreate" {
			stdout, cmdErr, statusCode = sdtDeploy.CreateVenv(homeUser, venvData, "", svcInfo)
			sdtDeploy.InstallDefaultPkg(venvData.VenvName, configData.DeviceType, configData.ServiceType, svcInfo)

			// if status is fail, delete app's data.
			if statusCode == http.StatusBadRequest {
				sdtDeploy.DeleteVenv(venvData.VenvName, svcInfo)
				procLog.Warn.Printf("[Venv] Venv's creation failed: delete venv's data.\n")
			}
		} else if m.SubCmdType == "venvDelete" {
			stdout, cmdErr, statusCode = sdtDeploy.DeleteVenv(venvData.VenvName, svcInfo)
		} else if m.SubCmdType == "venvUpdate" {
			stdout, cmdErr, statusCode = sdtDeploy.UpdateVenv(venvData, svcInfo)
		}

		if cmdErr != nil {
			procLog.Error.Printf("[Venv] Error1: %s\n", stdout)
			procLog.Error.Printf("[Venv] Error2: %v\n", cmdErr)
			statusCode = http.StatusBadRequest
		} else {
			statusCode = http.StatusOK
		}

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, m.SubCmdType, stdout)
		cmdResult.VenvName = venvData.VenvName
		cmdResult.VenvRequirement = venvData.Requirement

		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}

	case "deploy":
		var deployData sdtType.CmdDeploy
		var appPid int = -1
		var appSize int64 = -1
		var appRepo string = ""
		var deployMessage string = ""

		var cmdErr, configErr error
		var commonResult, jsonResult map[string]interface{}
		var inferenceResult, infJsonResults []map[string]interface{}

		procLog.Info.Printf("[DEPLOY] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &deployData)
		if err != nil {
			procLog.Error.Printf("[DEPLOY] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if !CheckDeploy(m.SubCmdType, deployData) {
			result = FormError(configData.AssetCode, m.RequestId, m.CmdType, m.SubCmdType)
			procLog.Error.Printf("[DEPLOY] Format Error: %s\n", result)
			break
		}

		if m.SubCmdType == "appDeploy" {
			if len(deployData.Apps) > 0 {
				procLog.Info.Printf("[DEPLOY-INF] <INFERENCE APP> It is inference APP!!! \n")
				inferenceResult, cmdErr, statusCode, deployData.VenvName = sdtDeploy.InferenceDeploy(deployData, archType, svcInfo, configData, homeUser, cli)

				// if status is fail, delete app's data.
				if statusCode == http.StatusBadRequest {
					procLog.Error.Printf("[DEPLOY-INF] App's deploy failed[Deploy]: %v.\n", cmdErr)
					appNames, appIds := sdtDeploy.GetAppsFromGroup(deployData.AppGroupId, svcInfo.RootPath)
					sdtDeploy.InferenceDelete(appNames, appIds, archType, deployData, svcInfo)
				} else {
					// 앱이 배포되어 새 Config 파일이 생성됐으므로, 현재 Config 값 확인
					for infIndex, _ := range deployData.Apps {
						jsonResult, configErr, statusCode = sdtConfig.GetConfig(deployData.Apps[infIndex].AppId, deployData.Apps[infIndex].AppName, svcInfo.AppPath, "")
						if statusCode == http.StatusBadRequest {
							sdtDeploy.Delete(deployData.AppName, deployData.AppId, archType, svcInfo)
							procLog.Error.Printf("[DEPLOY-INF] App's deploy failed[GetConfig]: %v.\n", configErr)
							//break
							// 앱 Config를 가져올 수 없으면 nil 로 들어감.
						}
						infJsonResults = append(infJsonResults, jsonResult)
					}
				}

			} else {
				procLog.Info.Printf("[DEPLOY] <COMMON APP> It is common APP!!! \n")
				commonResult, cmdErr, statusCode, deployData.VenvName = sdtDeploy.Deploy(deployData, archType, svcInfo, configData, homeUser, cli)

				// if status is fail, delete app's data.
				if statusCode == http.StatusBadRequest {
					sdtDeploy.Delete(deployData.AppName, deployData.AppId, archType, svcInfo)
					procLog.Error.Printf("[DEPLOY] App's deploy failed[Deploy]: %v.\n", cmdErr)
				} else {

					// 앱이 배포되어 새 Config 파일이 생성됐으므로, 현재 Config 값 확인
					jsonResult, configErr, statusCode = sdtConfig.GetConfig(deployData.AppId, deployData.AppName, svcInfo.AppPath, "")

					if statusCode == http.StatusBadRequest {
						sdtDeploy.Delete(deployData.AppName, deployData.AppId, archType, svcInfo)
						procLog.Error.Printf("[DEPLOY] App's deploy failed[GetConfig]: %v.\n", configErr)
					}

					//configResult = GetConfig(deployData.AppId,
					//	deployData.AppName,
					//	configData.AssetCode,
					//	statusCode,
					//	m.RequestId,
					//	archType,
					//)
				}
				appRepo = commonResult["appRepoPath"].(string)
			}
		} else if m.SubCmdType == "appDelete" {
			if deployData.AppGroupId == "" {
				commonResult, cmdErr, statusCode = sdtDeploy.Delete(deployData.AppName, deployData.AppId, archType, svcInfo)
			} else {
				appNames, appIds := sdtDeploy.GetAppsFromGroup(deployData.AppGroupId, svcInfo.RootPath)
				inferenceResult, cmdErr, statusCode, deployData, infJsonResults = sdtDeploy.InferenceDelete(appNames, appIds, archType, deployData, svcInfo)
			}
		} else if m.SubCmdType == "appStart" {
			commonResult, cmdErr, statusCode = sdtDeploy.Start(deployData.AppName, deployData.AppId, archType, svcInfo)
		} else if m.SubCmdType == "appStop" {
			commonResult, cmdErr, statusCode = sdtDeploy.Stop(deployData.AppName, deployData.AppId, archType, svcInfo)
		}

		if cmdErr != nil {
			procLog.Error.Printf("[DEPLOY] Error1: %s\n", commonResult)
			procLog.Error.Printf("[DEPLOY] Error2: %v\n", cmdErr)
			deployMessage = fmt.Sprintf("%s's %s failed.", deployData.AppName, m.SubCmdType)

			//
		} else {
			deployMessage = fmt.Sprintf("%s's %s successed.", deployData.AppName, m.SubCmdType)
		}

		// 결과 메시지 생성
		var cmdResult sdtType.CmdResult
		var cmdResults []sdtType.CmdResult
		if len(deployData.Apps) > 0 {
			cmdResult = sdtType.NewCmdResult(m.CmdType, m.SubCmdType, deployMessage)
			for resultIndex, _ := range deployData.Apps {
				cmdResult = sdtType.NewCmdResult(m.CmdType, m.SubCmdType, deployMessage)
				// 받은 메시지 기반 값
				cmdResult.AppId = deployData.Apps[resultIndex].AppId
				cmdResult.AppName = deployData.Apps[resultIndex].AppName
				cmdResult.ModelName = deployData.Apps[resultIndex].ModelName
				cmdResult.ModelVersion = deployData.Apps[resultIndex].ModelVersion
				cmdResult.ModelId = deployData.Apps[resultIndex].ModelId

				// 처리 후 결과 값
				if cmdErr != nil {
					cmdResult.VenvName = ""
					cmdResult.Size = int64(-1)
					cmdResult.Pid = int(-1)
					cmdResult.AppRepoPath = ""
					cmdResult.Parameter = nil
				} else {
					cmdResult.VenvName = inferenceResult[resultIndex]["venv"].(string)
					cmdResult.Size = inferenceResult[resultIndex]["size"].(int64)
					cmdResult.Pid = inferenceResult[resultIndex]["pid"].(int)
					cmdResult.AppRepoPath = inferenceResult[resultIndex]["appRepoPath"].(string)
					cmdResult.Parameter = infJsonResults[resultIndex]
				}
				cmdResults = append(cmdResults, cmdResult)

			}

			result = sdtType.ResultMsg{
				AssetCode:  configData.AssetCode,
				Results:    &cmdResults,
				RequestId:  m.RequestId,
				AppGroupId: deployData.AppGroupId,
			}
		} else {
			if cmdErr == nil {
				appPid = commonResult["pid"].(int)
				appSize = commonResult["size"].(int64)
			}

			cmdResult = sdtType.NewCmdResult(m.CmdType, m.SubCmdType, deployMessage)
			cmdResult.VenvName = deployData.VenvName
			cmdResult.AppId = deployData.AppId
			cmdResult.AppName = deployData.AppName
			cmdResult.AppRepoPath = appRepo
			cmdResult.Pid = appPid
			cmdResult.Size = appSize
			cmdResult.Parameter = jsonResult

			result = sdtType.ResultMsg{
				AssetCode: configData.AssetCode,
				Result:    &cmdResult,
				RequestId: m.RequestId,
			}
		}
		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}
		result.Status = cmdStatus

	case "model":
		var modelData sdtType.CmdModel
		var modelMessage string
		var cmdErr error
		var modelResult map[string]interface{}

		procLog.Info.Printf("[MODEL] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &modelData)
		if err != nil {
			procLog.Error.Printf("[MODEL] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if m.SubCmdType == "update" {
			modelResult, cmdErr, statusCode = sdtModel.Update(modelData, svcInfo)
		}

		if cmdErr != nil {
			procLog.Error.Printf("[MODEL] Result: %s\n", modelResult)
			procLog.Error.Printf("[MODEL] Error: %v\n", cmdErr)
			modelMessage = fmt.Sprintf("%s's %s failed.", modelData.AppName, m.SubCmdType)
		} else {
			modelMessage = fmt.Sprintf("%s's %s successed.", modelData.AppName, m.SubCmdType)
		}

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, m.SubCmdType, string(modelMessage))
		cmdResult.AppId = modelData.AppId
		cmdResult.AppName = modelData.AppName
		cmdResult.Parameter = modelResult
		cmdResult.ModelId = modelData.ModelId
		cmdResult.ModelName = modelData.ModelName
		cmdResult.ModelVersion = modelData.ModelVersion

		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}

	case "config":
		var jsonData sdtType.CmdJson
		var stdout string
		var cmdErr, configErr error
		var jsonResult map[string]interface{}

		procLog.Info.Printf("[CONFIG] Control: %s \n", string(json_data))
		err := json.Unmarshal([]byte(string(json_data)), &jsonData)
		if err != nil {
			procLog.Error.Printf("[CONFIG] Unmarshal Error: %v\n", err)
			// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
		}

		if m.SubCmdType == "configFix" {
			stdout, cmdErr, statusCode = sdtConfig.JsonChange(jsonData, svcInfo.AppPath, "")

			// Config 파일의 수정이 발생했으므로, 현재 Config 값 확인
			jsonResult, configErr, statusCode = sdtConfig.GetConfig(jsonData.AppId, jsonData.AppName, svcInfo.AppPath, "")

			if statusCode == http.StatusBadRequest {
				procLog.Error.Printf("[CONFIG] Failed get config: %v.\n", configErr)
			}
		} else if m.SubCmdType == "controlAquarack" {
			stdout, cmdErr, statusCode = sdtConfig.JsonChange(jsonData, "/etc/sdt/aquaApp/aquarack-data-collector", m.SubCmdType)
			// Config 파일의 수정이 발생했으므로, 현재 Config 값 확인

			jsonResult, configErr, statusCode = sdtConfig.GetConfig(jsonData.AppId, jsonData.AppName, "/etc/sdt/aquaApp", m.SubCmdType)

			if statusCode == http.StatusBadRequest {
				procLog.Error.Printf("[AQUARACK] Failed aquarack control: %v.\n", configErr)
			}
		} else if m.SubCmdType == "getConfigAquarack" {

			stdout = "Successfully get confing of aquarack agent."
			jsonResult, configErr, statusCode = sdtConfig.GetConfig(jsonData.AppId, jsonData.AppName, "/etc/sdt/aquaApp", m.SubCmdType)

			if statusCode == http.StatusBadRequest {
				procLog.Error.Printf("[AQUARACK] Failed get aquarack's config: %v.\n", configErr)
			}
		}

		if cmdErr != nil {
			procLog.Error.Printf("[CONFIG] Error1: %s\n", stdout)
			procLog.Error.Printf("[CONFIG] Error2: %v\n", cmdErr)
		}

		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, m.SubCmdType, string(stdout))
		cmdResult.AppId = jsonData.AppId
		cmdResult.AppName = jsonData.AppName
		cmdResult.Parameter = jsonResult

		cmdStatus := sdtType.NewCmdStatus(statusCode)
		if cmdErr == nil {
			cmdStatus.ErrMsg = ""
			cmdStatus.Succeed = 1
		} else {
			cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
			cmdStatus.Succeed = 0
		}

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}

	//case "pid":
	//	var pidData sdtType.CmdPid
	//	procLog.Info.Printf("[PID] Control: %s \n", string(json_data))
	//	err := json.Unmarshal([]byte(string(json_data)), &pidData)
	//	if err != nil {
	//		procLog.Error.Printf("[PID] Unmarshal Error: %v\n", err)
	//		// log.Error(fmt.Sprintf("Unmarshal Error: %v", err))
	//	}
	//	pidResult, cmd_err, statusCode := sdtDeploy.GetPid(pidData.AppName)
	//	if cmd_err != nil {
	//		procLog.Error.Printf("[PID] Error: %v\n", cmd_err)
	//		result = sdtMessage.CheckResult(configData.AssetCode, pidData.AppName, "",
	//			cmd_err, statusCode, m.CmdType,
	//			"app", m.RequestId, -1, -1, nil, "", "", "")
	//	} else {
	//		pidMsg := fmt.Sprintf("%d", pidResult)
	//		result = sdtMessage.CheckResult(configData.AssetCode, pidData.AppName, pidMsg,
	//			cmd_err, statusCode, m.CmdType,
	//			"app", m.RequestId, pidResult, -1, nil, "", "", "")
	//	}
	default:
		statusCode = http.StatusNotFound
		// 결과 메시지 생성
		cmdResult := sdtType.NewCmdResult(m.CmdType, m.SubCmdType, "")

		cmdStatus := sdtType.NewCmdStatus(statusCode)
		cmdStatus.ErrMsg = fmt.Sprintf("%v", errors.New("This command not found."))
		cmdStatus.Succeed = 0

		result = sdtType.ResultMsg{
			AssetCode: configData.AssetCode,
			Result:    &cmdResult,
			Status:    cmdStatus,
			RequestId: m.RequestId,
		}
	}

	return result, configResult
}

//// GetConfig retrieves the config value of a deployed app. In SDT Cloud,
//// apps are deployed with a config file for control purposes.
////
//// Input:
////   - appId: ID of the app.
////   - appName: Name of the app.
////   - assetCode: Serial number of the device.
////   - statusCode: Status code of the control command result.
////   - requestId: ID of the control command.
////   - archType: Architecture type of the device.
////
//// Output:
////   - map[string]interface{}: Config value of the app to send to the cloud.
//func GetConfig(appId string,
//	appName string,
//	assetCode string,
//	statusCode int,
//	requestId string,
//	archType string) sdtType.ResultMsg {
//
//	jsonResult, cmdErr, statusCode := sdtConfig.GetConfig(appId, appName, )
//
//	// 결과 메시지 생성
//	cmdResult := sdtType.NewCmdResult("config", "get", "Successfully get config.")
//	cmdResult.AppId = appId
//	cmdResult.Parameter = jsonResult
//
//	cmdStatus := sdtType.NewCmdStatus(statusCode)
//	if cmdErr == nil {
//		cmdStatus.ErrMsg = ""
//		cmdStatus.Succeed = 1
//	} else {
//		cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
//		cmdStatus.Succeed = 0
//	}
//
//	result := sdtType.ResultMsg{
//		AssetCode: assetCode,
//		Result:    &cmdResult,
//		Status:    cmdStatus,
//		RequestId: requestId,
//	}
//	return result
//}

// CheckBash validates the request parameters for bash type control commands.
// The bash command must specify the actual command to be executed.
//
// Input:
//   - checkData: Struct containing bash command information.
//
// Output:
//   - bool: Validation result (true: valid, false: issue detected)
func CheckBash(checkData sdtType.CmdBash) bool {
	if checkData.Cmd == "" {
		return false
	}
	return true
}

// CheckSystemd validates the request parameters for systemd type control commands.
// The systemd command must specify the actual command to be executed.
//
// Input:
//   - checkData: Struct containing systemd command information.
//
// Output:
//   - bool: Validation result (true: valid, false: issue detected)
func CheckSystemd(checkData sdtType.CmdSystemd) bool {
	if checkData.Cmd == "" {
		return false
	} else if checkData.Service == "" {
		return false
	}
	return true
}

// CheckDocker validates the request parameters for docker type control commands.
// The docker command must specify the actual command to be executed.
//
// Input:
//   - checkData: Struct containing docker command information.
//
// Output:
//   - bool: Validation result (true: valid, false: issue detected)
func CheckDocker(subCmd string, checkData sdtType.CmdDocker) bool {
	if subCmd == "appDeploy" {
		if checkData.Image == "" {
			return false
		}
	} else if checkData.AppName == "" {
		return false
	}

	return true
}

// CheckVenv validates the request parameters for virutal environment type control commands.
// The virutal environment command must specify the actual command to be executed.
//
// Input:
//   - checkData: Struct containing virutal environment command information.
//
// Output:
//   - bool: Validation result (true: valid, false: issue detected)
func CheckVenv(subCmd string, checkData sdtType.CmdVenv) bool {
	if subCmd == "venvCreate" {
		if checkData.VenvName == "" {
			return false
		} else if checkData.BinFile == "" {
			return false
		}
	}
	return true
}

// CheckDeploy validates the request parameters for deploy type control commands.
// The deploy command must specify the actual command to be executed.
//
// Input:
//   - checkData: Struct containing deploy command information.
//
// Output:
//   - bool: Validation result (true: valid, false: issue detected)
func CheckDeploy(subCmd string, checkData sdtType.CmdDeploy) bool {
	if subCmd == "appDeploy" {
		if checkData.FileUrl == "" && checkData.Image == "" && checkData.Apps == nil {
			return false
		}
		// } else if checkData.Exec == "" {
		// 	return false
		// }
	} else {
		if checkData.AppName == "" && checkData.AppGroupId == "" {
			return false
		}
	}
	return true
}

// FormError creates an error message to return when a control command fails.
// The format of the error message is as follows:
//
//	msg = {
//		"assetCode": "SerialNumber",
//		"status": {
//			"succeed": 0 or 1,
//			"statusCode": int
//			"errMsg": string
//		},
//		"result": {
//			"name": string,
//			"pid": int,
//			"size": int,
//			"message": string,
//			"releasedAt": int64,
//			"updatedAt": int64
//		},
//		"requestId": string
//	}
//
// Input:
//   - assetcode: Serial number of the device.
//   - requestId: ID of the control command.
//
// Output:
//   - map[string]interface{}: Error message to be sent to the cloud.
func FormError(
	assetcode string, // Edge's name
	requestId string,
	cmd string,
	subCmd string,
) sdtType.ResultMsg {

	//result = map[string]interface{}{
	//	"name":       "",
	//	"pid":        -1,
	//	"size":       -1,
	//	"message":    "This is not the correct form.",
	//	"releasedAt": int64(time.Now().UTC().Unix() * 1000),
	//	"updatedAt":  int64(time.Now().UTC().Unix() * 1000),
	//}
	//
	//statusBody = map[string]interface{}{
	//	"succeed":    0,
	//	"statusCode": http.StatusBadRequest,
	//	"errMsg":     "This is not the correct form.",
	//}
	//
	//cmdMsg = map[string]interface{}{
	//	"assetCode": assetcode,
	//	"result":    result,
	//	"status":    statusBody,
	//	"requestId": requestId,
	//}

	// 결과 메시지 생성
	cmdResult := sdtType.NewCmdResult(cmd, subCmd, "This is not the correct form.")

	cmdStatus := sdtType.NewCmdStatus(http.StatusBadRequest)
	cmdStatus.ErrMsg = "This is not the correct form."
	cmdStatus.Succeed = 0

	formErrResult := sdtType.ResultMsg{
		AssetCode: assetcode,
		Result:    &cmdResult,
		Status:    cmdStatus,
		RequestId: requestId,
	}

	return formErrResult
}
