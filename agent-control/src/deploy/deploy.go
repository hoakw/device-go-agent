// The Deploy package handles the deployment of applications on the device.
// Applications are installed and deployed to "/usr/local/sdt/app" directory on the device.
// Applications are deployed using systemd and dockerd. Once deployed, applications can be
// monitored from the SDT Cloud console and accessed from the device terminal using BWC CLI.
package deploy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/minio/minio-go/v7"
	"io"
	"io/ioutil"
	sdtConfig "main/src/config"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	mqttCli "github.com/eclipse/paho.mqtt.golang"
	"github.com/google/uuid"
	"github.com/mholt/archiver"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gopkg.in/yaml.v3"

	sdtType "main/src/controlType"
	sdtMessage "main/src/message"
)

// These are the global variables used in the deploy package.
// - procLog: This is the struct that defines the format of the Log.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// The Start function starts an application deployed on the device.
//
// Input:
//   - appName: The name of the application.
//   - archType: The architecture of the device.
//
// Output:
//   - map[string]interface{}: Information about the application (app name, PID, app size).
//   - error: Error message in case of issues with the start command.
//   - int: Status code of the command execution.
func Start(appName string, appId string, archType string, svcInfo sdtType.ControlService) (map[string]interface{}, error, int) {
	var pid int
	if archType == "win" {
		//taskName := fmt.Sprintf("SDT\\%s", appName)
		//cmd_run := exec.Command("schtasks", "/run", "/tn", taskName)
		//stdout, cmd_err := cmd_run.CombinedOutput()
		//if cmd_err != nil {
		//	procLog.Error.Printf("[DEPLOY] Service Start Error: %s\n", stdout)
		//	return nil, errors.New(string(stdout)), http.StatusBadRequest
		//}
		bwcFramework := GetVenvFromFramework(appName, appId, svcInfo.AppPath)
		svcCmd := fmt.Sprintf("C:/sdt/venv/%s/python.exe C:/sdt/app/%s_%s/main.py start", bwcFramework.Spec.Env.VirtualEnv, appName, appId)
		cmd_run := exec.Command("cmd.exe", "/c", svcCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Start Error: %s\n", stdout)
			return nil, errors.New(string(stdout)), http.StatusBadRequest
		}
		pid = -1
	} else {
		startCmd := fmt.Sprintf("systemctl start %s", appName)
		cmd_run := exec.Command("sh", "-c", startCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Start Error: %s\n", stdout)
			return nil, errors.New(string(stdout)), http.StatusBadRequest
		}

		// get pid
		pid, cmd_err, _ = GetPid(appName)
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Get PID Error: %s\n", cmd_err)
			return nil, cmd_err, http.StatusBadRequest
		}

	}
	stdout := map[string]interface{}{
		"name": appName,
		"pid":  pid,
		"size": int64(-1),
	}

	return stdout, nil, http.StatusOK
}

// The Stop function stops an application deployed on the device.
//
// Input:
//   - appName: The name of the application.
//   - archType: The architecture of the device.
//
// Output:
//   - map[string]interface{}: Information about the application (app name, PID, app size).
//   - error: Error message in case of issues with the stop command.
//   - int: Status code of the command execution.
func Stop(appName string, appId string, archType string, svcInfo sdtType.ControlService) (map[string]interface{}, error, int) {
	if archType == "win" {
		//taskName := fmt.Sprintf("SDT\\%s", appName)
		//cmd_run := exec.Command("schtasks", "/end", "/tn", taskName)
		//stdout, cmd_err := cmd_run.CombinedOutput()
		//if cmd_err != nil {
		//	procLog.Error.Printf("[DEPLOY] Service Stop Error: %s\n", stdout)
		//	return nil, errors.New(string(stdout)), http.StatusBadRequest
		//}
		bwcFramework := GetVenvFromFramework(appName, appId, svcInfo.AppPath)
		svcCmd := fmt.Sprintf("C:/sdt/venv/%s/python.exe C:/sdt/app/%s_%s/main.py stop", bwcFramework.Spec.Env.VirtualEnv, appName, appId)
		cmd_run := exec.Command("cmd.exe", "/c", svcCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Stop Error: %s\n", stdout)
			return nil, errors.New(string(stdout)), http.StatusBadRequest
		}
	} else {
		stopCmd := fmt.Sprintf("systemctl stop %s", appName)
		cmd_run := exec.Command("sh", "-c", stopCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Stop Error: %s\n", stdout)
			return nil, errors.New(string(stdout)), http.StatusBadRequest
		}

	}
	stdout := map[string]interface{}{
		"name": appName,
		"pid":  -1,
		"size": int64(-1),
	}

	return stdout, nil, http.StatusOK
}

// The Delete function deletes an application deployed on the device. Deleting an
// application removes its directory and Systemd (.service) file.
//
// Input:
//   - appId: The ID of the application.
//   - appName: The name of the application.
//   - archType: The architecture of the device.
//
// Output:
//   - map[string]interface{}: Information about the application (app name, PID, app size).
//   - error: Error message in case of issues with the delete command.
//   - int: Status code of the command execution.
func Delete(appName string, appId string, archType string, svcInfo sdtType.ControlService) (map[string]interface{}, error, int) {
	if appName == "" || appName == "*" {
		cmd_err := errors.New("Not collect value.")
		procLog.Error.Printf("[DELETE] Not collect value: %s\n", appName)
		return nil, cmd_err, http.StatusBadRequest
	}

	if archType == "win" {

		//taskName := fmt.Sprintf("SDT\\%s", appName)
		//cmd_run := exec.Command("schtasks", "/end", "/tn", taskName)
		//stdout, cmd_err := cmd_run.CombinedOutput()
		//if cmd_err != nil {
		//	procLog.Error.Printf("[DELETE] Service Stop Error: %s\n", stdout)
		//	return nil, errors.New(string(stdout)), http.StatusBadRequest
		//}
		//
		//cmd_run = exec.Command("schtasks", "/delete", "/tn", taskName, "/f")
		//stdout, cmd_err = cmd_run.CombinedOutput()
		//if cmd_err != nil {
		//	procLog.Error.Printf("[DELETE] Service Delete Error: %s\n", stdout)
		//	return nil, errors.New(string(stdout)), http.StatusBadRequest
		//}
		bwcFramework := GetVenvFromFramework(appName, appId, svcInfo.AppPath)
		svcCmd := fmt.Sprintf("C:/sdt/venv/%s/python.exe C:/sdt/app/%s_%s/main.py stop", bwcFramework.Spec.Env.VirtualEnv, appName, appId)
		cmd_run := exec.Command("cmd.exe", "/c", svcCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Stop Error: %s\n", stdout)
			return nil, errors.New(string(stdout)), http.StatusBadRequest
		}

		svcCmd = fmt.Sprintf("C:/sdt/venv/%s/python.exe C:/sdt/app/%s_%s/main.py remove", bwcFramework.Spec.Env.VirtualEnv, appName, appId)
		cmd_run = exec.Command("cmd.exe", "/c", svcCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DEPLOY] Remove Error: %s\n", stdout)
			return nil, errors.New(string(stdout)), http.StatusBadRequest
		}

		filePath := fmt.Sprintf("C:/sdt/app/%s_%s", appName, appId)
		procLog.Info.Println("[DELETE] APP Remove: ", filePath)
		cmd_err = os.RemoveAll(filePath)
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE] App file Delete Error: %v\n", cmd_err)
			return nil, cmd_err, http.StatusBadRequest
		}

		//startDir := "C:/ProgramData/Microsoft/Windows/Start Menu/Programs/StartUp"
		//batPath := fmt.Sprintf("%s/%s.bat", startDir, appName)
		//procLog.Info.Println("[DELETE] APP Bat Remove: ", filePath)
		//cmd_err = os.RemoveAll(batPath)
		//if cmd_err != nil {
		//	procLog.Error.Printf("[DELETE] App bat Delete Error: %v\n", cmd_err)
		//	return nil, cmd_err, http.StatusBadRequest
		//}
		// TO DO
		// remove app
	} else {
		appRemoveCmd := fmt.Sprintf("rm -rf /usr/local/sdt/app/%s_%s", appName, appId)
		cmd_run := exec.Command("sh", "-c", appRemoveCmd)
		procLog.Warn.Printf("[DELETE] Remove app : %s\n", appName)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE] Remove app Error: %s\n", stdout)
		}

		disableCmd := fmt.Sprintf("systemctl disable %s", appName)
		cmd_run = exec.Command("sh", "-c", disableCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE] Disable Error: %s\n", stdout)
			// return nil, cmd_err, http.StatusBadRequest
		}

		stopCmd := fmt.Sprintf("systemctl stop %s", appName)
		cmd_run = exec.Command("sh", "-c", stopCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE] Stop Error: %s\n", stdout)
			// return nil, cmd_err, http.StatusBadRequest
		}

		removeCmd := fmt.Sprintf("rm /etc/systemd/system/%s.service", appName)
		cmd_run = exec.Command("sh", "-c", removeCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE] Remove service Error: %s\n", stdout)
			// return nil, cmd_err, http.StatusBadRequest
		}
	}

	// delete app json
	DeleteAppInfo(appName, svcInfo.RootPath)

	result := map[string]interface{}{
		"name": appName,
		"pid":  -1,
		"size": int64(-1),
	}

	procLog.Warn.Printf("[DELETE] Finish remove app : %s\n", appName)

	return result, nil, http.StatusOK
}

func InferenceDelete(appNames []string,
	appIds []string,
	archType string,
	deployData sdtType.CmdDeploy,
	svcInfo sdtType.ControlService) ([]map[string]interface{}, error, int, sdtType.CmdDeploy, []map[string]interface{}) {
	// TODO: Win과 Linux 분기
	var results, infResult []map[string]interface{}

	if len(appNames) == 0 || len(appIds) == 0 {
		procLog.Error.Printf("[DELETE-INF] Failed get apps info: app's len is zero.\n")
		return nil, errors.New("App's len is zero."), http.StatusBadRequest, deployData, nil
	}

	for index, _ := range appNames {
		appRemoveCmd := fmt.Sprintf("rm -rf /usr/local/sdt/app/%s_%s", appNames[index], appIds[index])
		cmd_run := exec.Command("sh", "-c", appRemoveCmd)
		procLog.Warn.Printf("[DELETE-INF] Remove app [%d / %d] : %s\n", index+1, len(appNames), appNames[index])
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE-INF] Remove app Error: %s\n", stdout)
		}

		disableCmd := fmt.Sprintf("systemctl disable %s", appNames[index])
		cmd_run = exec.Command("sh", "-c", disableCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE-INF] Disable Error: %s\n", stdout)
			// return nil, cmd_err, http.StatusBadRequest
		}

		stopCmd := fmt.Sprintf("systemctl stop %s", appNames[index])
		cmd_run = exec.Command("sh", "-c", stopCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE-INF] Stop Error: %s\n", stdout)
			// return nil, cmd_err, http.StatusBadRequest
		}

		removeCmd := fmt.Sprintf("rm /etc/systemd/system/%s.service", appNames[index])
		cmd_run = exec.Command("sh", "-c", removeCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[DELETE-INF] Remove service Error: %s\n", stdout)
			// return nil, cmd_err, http.StatusBadRequest
		}

		// delete app json
		DeleteAppInfo(appNames[index], svcInfo.RootPath)

		result := map[string]interface{}{
			"name":        appNames[index],
			"pid":         -1,
			"size":        int64(-1),
			"venv":        "",
			"appRepoPath": "",
		}

		deleteApp := sdtType.InferenceDeploy{
			AppId:        appIds[index],
			AppName:      appNames[index],
			ModelName:    "",
			ModelVersion: -1,
			ModelId:      "",
		}

		deployData.Apps = append(deployData.Apps, deleteApp)
		infResult = append(infResult, nil)

		results = append(results, result)
		procLog.Warn.Printf("[DELETE-INF] Finish remove app [%d / %d] : %s\n", index+1, len(appNames), appNames[index])
	}

	return results, nil, http.StatusOK, deployData, infResult
}

// The Deploy function deploys an application onto the device. Deploying an
// application creates its directory and Systemd (.service) file.
//
// Input:
//   - deployData: Struct containing deployment command information.
//   - archType: The architecture of the device.
//
// Output:
//   - map[string]interface{}: Information about the application (app name, PID, app size).
//   - error: Error message in case of issues with the deploy command.
//   - int: Status code of the command execution.
func Deploy(deployData sdtType.CmdDeploy,
	archType string,
	svcInfo sdtType.ControlService,
	configData sdtType.ConfigInfo,
	homeUser string,
	cli mqttCli.Client) (map[string]interface{}, error, int, string) {

	var deployResult map[string]interface{}
	var bwcFramework sdtType.Framework
	var runTime, pkgInfo string
	appId := deployData.AppId
	app := deployData.App
	venv := deployData.VenvName
	fileUrl := deployData.FileUrl
	appName := deployData.AppName

	// 결과값 초기화
	deployResult = map[string]interface{}{
		"name":        "",
		"pid":         -1,
		"size":        -1,
		"appRepoPath": "",
	}

	filePath, fileSize, cmd_err, appRepoPath, fileZip := fileDownload(fileUrl, appId, app, appName, archType)
	if cmd_err != nil {
		procLog.Error.Printf("[DEPLOY] Download Error: %v\n", cmd_err)
		return deployResult, cmd_err, http.StatusBadRequest, venv
	} else {
		// TODO: Linux에서 한거처럼 앱 배포 순서 추가해야함
		if archType == "win" {

			// change appname in framework.yaml
			SaveFramework(appName, appId, svcInfo.AppPath)

			bwcFramework = GetVenvFromFramework(appName, appId, svcInfo.AppPath)

			// Check exit about app
			if CheckExistApp(appName, svcInfo.RootPath) {
				procLog.Error.Printf("[DEPLOY] %s's app already exist.\n", appName)
				return deployResult, errors.New("App already exist."), http.StatusBadRequest, venv
			}

			// Get venv from framework
			if venv == "app-store" || venv == "" {
				procLog.Info.Printf("[DEPLOY] Deploy app as app-store.")
				venv = bwcFramework.Spec.Env.VirtualEnv
				runTime = bwcFramework.Spec.Env.RunTime
			} else {
				procLog.Info.Printf("[DEPLOY] Deploy app as console.")
				runTime = bwcFramework.Spec.Env.RunTime
			}

			// Check exist env.
			procLog.Info.Printf("[DEPLOY] The runtime is %s.\n", runTime)
			if strings.Contains(runTime, "python") {
				procLog.Info.Printf("[DEPLOY] Checking if %s venv exists.\n", venv)
				envList := GetVenvList(svcInfo.VenvPath)
				if !Contains(envList, venv) {
					// Create Virutal Env
					procLog.Warn.Printf("%s's venv not found.\n", venv)
					procLog.Warn.Printf("%s's venv install in device.\n", venv)
					venvData := sdtType.CmdVenv{
						VenvName:    venv,
						Requirement: bwcFramework.Spec.Env.Package,
						BinFile:     bwcFramework.Spec.Env.Bin,
						RunTime:     bwcFramework.Spec.Env.RunTime,
					}
					stdout, cmdErr, statusCode := CreateVenv(homeUser, venvData, filePath, svcInfo)
					InstallDefaultPkg(venvData.VenvName, configData.DeviceType, configData.ServiceType, svcInfo)

					if cmdErr != nil {
						procLog.Error.Printf("[DEPLOY] Failed download python pkg.\n")
						cmd_err = errors.New(string(stdout))
						return deployResult, cmd_err, statusCode, venv
					}

					newUUID := uuid.New()
					requestId := newUUID.String()

					// 결과 메시지 생성
					cmdResult := sdtType.NewCmdResult("virtualEnv", "venvCreate", stdout)
					cmdResult.VenvName = venv
					cmdResult.VenvRequirement = bwcFramework.Spec.Env.Package

					cmdStatus := sdtType.NewCmdStatus(statusCode)
					if cmdErr == nil {
						cmdStatus.ErrMsg = ""
						cmdStatus.Succeed = 1
					} else {
						cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
						cmdStatus.Succeed = 0
					}

					result := sdtType.ResultMsg{
						AssetCode: configData.AssetCode,
						Result:    &cmdResult,
						Status:    cmdStatus,
						RequestId: requestId,
					}

					topic := fmt.Sprintf("%s/%s/%s/bwc/control/self-deploy", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
					sdtMessage.SendDataEdgeMqtt(result, topic, cli)

				}

			}

			// APP Info 저장
			// save deploy json
			SaveAppInfo(appName, appId, venv, "systemd", sdtType.NewInferenceInfo(), "", svcInfo.RootPath)

			// 서비스 등록
			svcCmd := fmt.Sprintf("C:/sdt/venv/%s/python.exe C:/sdt/app/%s_%s/main.py install", venv, appName, appId)
			cmd_run := exec.Command("cmd.exe", "/c", svcCmd)
			cmdResult, cmd_err := cmd_run.CombinedOutput()
			if cmd_err != nil {
				procLog.Error.Printf("[DEPLOY] Service creation Error: %s\n", cmdResult)
				return deployResult, errors.New(string(cmdResult)), http.StatusBadRequest, venv
			}
			procLog.Info.Printf("[DEPLOY] Service creation Result: %s\n", cmdResult)

			// Restart 설정
			svcName := fmt.Sprintf("%s_%s", appName, appId)
			cmd_run = exec.Command("sc", "failure", svcName, "reset= 0", "actions= restart/10000")
			cmdResult, cmd_err = cmd_run.CombinedOutput()
			if cmd_err != nil {
				procLog.Error.Printf("[DEPLOY] Service set restart Error: %s\n", cmdResult)
				return deployResult, errors.New(string(cmdResult)), http.StatusBadRequest, venv
			}
			procLog.Info.Printf("[DEPLOY] Service set restart: %s\n", cmdResult)

			// Service 시작
			svcCmd = fmt.Sprintf("C:/sdt/venv/%s/python.exe C:/sdt/app/%s_%s/main.py start", venv, appName, appId)
			cmd_run = exec.Command("cmd.exe", "/c", svcCmd)
			cmdResult, cmd_err = cmd_run.CombinedOutput()
			if cmd_err != nil {
				procLog.Error.Printf("[DEPLOY] Service start Error: %s\n", cmdResult)
				return deployResult, errors.New(string(cmdResult)), http.StatusBadRequest, venv
			}
			procLog.Info.Printf("[DEPLOY] Service start Result: %s\n", cmdResult)

			deployResult["name"] = appName
			deployResult["pid"] = -1 // TODO: Windows의 PID를 어떻게 가져오지?
			deployResult["size"] = fileSize
			deployResult["appRepoPath"] = appRepoPath
			// return stdout2, cmd_err, http.StatusOK, appRepoPath

			// return stdout2, cmd_err, http.StatusBadRequest, appRepoPath
		} else {
			// Linux arch...
			// new deploy

			// save deploy json
			//SaveAppInfo(appName, appId, venv, "systemd", sdtType.NewInferenceInfo(), "")

			// change appname in framework.yaml
			SaveFramework(appName, appId, svcInfo.AppPath)

			// Check new or old
			installFile := fmt.Sprintf("%s/install.sh", filePath)

			// Check if the file exists
			if _, err := os.Stat(installFile); err == nil { // -> Old versiopn
				// old version
				cmd := fmt.Sprintf("%s/install.sh", filePath)
				procLog.Info.Println("[DEPLOY] Start Deploy: ", cmd)
				cmd_run := exec.Command("bash", cmd, appName, appId, appId)
				stdout, cmd_err := cmd_run.CombinedOutput()

				if cmd_err != nil {
					procLog.Error.Println("[DEPLOY] Fail deploy: ", cmd_err, "\n", string(stdout))
					cmd_err = errors.New(string(stdout))
					return deployResult, errors.New(string(stdout)), http.StatusBadRequest, venv
				}
			} else if os.IsNotExist(err) { // -> New version
				// new version
				bwcFramework = GetVenvFromFramework(appName, appId, svcInfo.AppPath)

				// Check exist about app
				if CheckExistApp(appName, svcInfo.RootPath) {
					procLog.Error.Printf("[DEPLOY] %s's app already exist.\n", appName)
					return deployResult, errors.New("App already exist."), http.StatusBadRequest, venv
				}

				// Get venv from framework
				if venv == "app-store" || venv == "" {
					procLog.Info.Printf("[DEPLOY] Deploy app as app-store.")
					venv = bwcFramework.Spec.Env.VirtualEnv
					runTime = bwcFramework.Spec.Env.RunTime
					// Get Pkg Info from requirement.txt
					pkgPath := fmt.Sprintf("%s/%s_%s/%s", svcInfo.AppPath, deployData.AppName, deployData.AppId, bwcFramework.Spec.Env.Package)
					content, err := os.ReadFile(pkgPath)
					if err != nil {
						procLog.Warn.Printf("[DEPLOY] Failed read requirement.txt: %v\n", err)
						pkgInfo = ""
					}
					pkgInfo = string(content)

				} else {
					// 콘솔에서 앱 배포 기능 삭제 됨
					procLog.Info.Printf("[DEPLOY] Deploy app as console.")
					pkgInfo = ""
					runTime = bwcFramework.Spec.Env.RunTime
				}

				// Check exist env.
				procLog.Info.Printf("[DEPLOY] The runtime is %s.\n", runTime)
				if strings.Contains(runTime, "python") {
					procLog.Info.Printf("[DEPLOY] Checking if %s venv exists.\n", venv)
					envList := GetVenvList(svcInfo.VenvPath)
					if !Contains(envList, venv) {
						// Create Virutal Env
						procLog.Warn.Printf("%s's venv not found.\n", venv)
						procLog.Warn.Printf("%s's venv install in device.\n", venv)
						venvData := sdtType.CmdVenv{
							VenvName:    venv,
							Requirement: bwcFramework.Spec.Env.Package,
							BinFile:     bwcFramework.Spec.Env.Bin,
							RunTime:     bwcFramework.Spec.Env.RunTime,
						}
						// requirements.txt 으로 패키지 설치할 때, 에러가 발생할 경우
						//  - 에러 메시지를 보내줘야 함
						stdout, cmdErr, statusCode := CreateVenv(homeUser, venvData, filePath, svcInfo)
						InstallDefaultPkg(venvData.VenvName, configData.DeviceType, configData.ServiceType, svcInfo)

						if cmdErr != nil {
							procLog.Error.Printf("[DEPLOY] Failed download python pkg.\n")
							//cmd_err = errors.New(string(stdout))
							return deployResult, cmdErr, statusCode, venv
						}

						newUUID := uuid.New()
						requestId := newUUID.String()

						// 결과 메시지 생성
						cmdResult := sdtType.NewCmdResult("virtualEnv", "venvCreate", stdout)
						cmdResult.VenvName = venv
						cmdResult.VenvRequirement = pkgInfo

						cmdStatus := sdtType.NewCmdStatus(statusCode)
						if cmdErr == nil {
							cmdStatus.ErrMsg = ""
							cmdStatus.Succeed = 1
						} else {
							cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
							cmdStatus.Succeed = 0
						}

						result := sdtType.ResultMsg{
							AssetCode: configData.AssetCode,
							Result:    &cmdResult,
							Status:    cmdStatus,
							RequestId: requestId,
						}

						//result := sdtMessage.CheckResult(configData.AssetCode, "", stdout,
						//	cmd_err, statusCode, "venvCreate",
						//	"virtualEnv", requestId, -1, -1, nil, "", "", venvData.VenvName)
						topic := fmt.Sprintf("%s/%s/%s/bwc/control/self-deploy", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
						sdtMessage.SendDataEdgeMqtt(result, topic, cli)

					}

					CreatePythonService(filePath, appName, venv, "main.py", svcInfo.VenvPath)
				} else if strings.Contains(runTime, "go") {
					CreateGoService(filePath, appName, "main.py")

				}

				// APP Info 저장
				// save deploy json
				SaveAppInfo(appName, appId, venv, "systemd", sdtType.NewInferenceInfo(), "", svcInfo.RootPath)

				//// Inference Check...
				//// TODO
				////  - Change appType(BwcFramework -> mqtt message(appType)
				////  - Add deployment of request app.
				//appType := bwcFramework.Spec.AppType
				//if appType == "inference" {
				//	err := CreateInferenceDir(filePath)
				//	if err != nil {
				//		procLog.Error.Printf("Failed creating inference directory.\n")
				//		return deployResult, err, http.StatusBadRequest, venv
				//	}
				//	procLog.Info.Printf("[DEPLOY] App is inference. So, device download weight file.(=%s)\n", appType)
				//	err = DownloadWeight(bwcFramework, filePath, svcInfo.MinioURL)
				//	if err != nil {
				//		procLog.Error.Printf("[DEPLOY] Failed download inference model.\n")
				//		return deployResult, err, http.StatusBadRequest, venv
				//	}
				//}

				// start systemd
				startCmd := fmt.Sprintf("systemctl start %s", appName)
				cmd_run := exec.Command("sh", "-c", startCmd)
				stdout, cmd_err := cmd_run.CombinedOutput()
				if cmd_err != nil {
					procLog.Error.Println("[DEPLOY] Fail deploy: ", cmd_err, "\n", string(stdout))
					cmd_err = errors.New(string(stdout))
					return deployResult, errors.New(string(stdout)), http.StatusBadRequest, venv
				}

				// enable systemd
				startCmd = fmt.Sprintf("systemctl enable %s", appName)
				cmd_run = exec.Command("sh", "-c", startCmd)
				stdout, cmd_err = cmd_run.CombinedOutput()
				if cmd_err != nil {
					procLog.Error.Println("[DEPLOY] Fail deploy: ", cmd_err, "\n", string(stdout))
					cmd_err = errors.New(string(stdout))
					return deployResult, errors.New(string(stdout)), http.StatusBadRequest, venv
				}
			}

			// save common deploy json

			//SaveAppInfo(appName, appId, venv, "systemd", sdtType.NewInferenceInfo(), "")
			procLog.Info.Println("[DEPLOY] End Deploy..")

			// get pid
			pid, cmd_err, _ := GetPid(appName)
			if cmd_err != nil {
				// get error log
				logResult := GetLogsApp(svcInfo.AppPath, appName, appId)
				//cmd_log := exec.Command("journalctl", "-u", appName, "-n", "30")
				//stdout, _ := cmd_log.CombinedOutput()
				//cmd_err = errors.New(string(stdout))
				//stdout2 := map[string]interface{}{
				//	"name":        appName,
				//	"pid":         pid,
				//	"size":        fileSize,
				//	"appRepoPath": appRepoPath,
				//}
				deployResult["name"] = appName
				deployResult["pid"] = pid
				deployResult["size"] = fileSize
				deployResult["appRepoPath"] = appRepoPath

				return deployResult, errors.New(logResult), http.StatusBadRequest, venv
			}
			//deployResult = map[string]interface{}{
			//	"name":        appName,
			//	"pid":         pid,
			//	"size":        fileSize,
			//	"appRepoPath": appRepoPath,
			//}
			deployResult["name"] = appName
			deployResult["pid"] = pid
			deployResult["size"] = fileSize
			deployResult["appRepoPath"] = appRepoPath
			// return stdout2, cmd_err, http.StatusOK, appRepoPath
		}
	}

	// remove zip file
	procLog.Info.Println("[DEPLOY] Remove ZIP File: ", fileZip)
	removeErr := os.Remove(fileZip)
	if removeErr != nil {
		time.Sleep(1)
		os.Remove(fileZip)
	}

	return deployResult, cmd_err, http.StatusOK, venv
}

// The InferenceDeploy function deploys inference onto the device. Deploying an
// application creates its directory and Systemd (.service) file.
//
// Input:
//   - deployData: Struct containing deployment command information.
//   - archType: The architecture of the device.
//
// Output:
//   - map[string]interface{}: Information about the application (app name, PID, app size).
//   - error: Error message in case of issues with the deploy command.
//   - int: Status code of the command execution.
func InferenceDeploy(deployData sdtType.CmdDeploy,
	archType string,
	svcInfo sdtType.ControlService,
	configData sdtType.ConfigInfo,
	homeUser string,
	cli mqttCli.Client) ([]map[string]interface{}, error, int, string) {

	// TODO:
	//  - 배포 시, Config값 함께 수정되서 배포가 되는 로직 추가 (완료)
	//  - Weight 파일 정의 필요
	//  - Config 파일 명칭 고정할 건지와 내용 정리(Key 값) (완료)
	var deployResult map[string]interface{}
	var inferenceResult []map[string]interface{}
	var bwcFramework sdtType.Framework
	var runTime, venv, appId, appName, modelFileName string
	var cmdErr error
	var pid int

	// 다수의 앱을 배포한다.
	for appIndex, appItem := range deployData.Apps {
		venv = appItem.VenvName
		appId = appItem.AppId
		appName = appItem.AppName

		// 결과 변수 초기화
		deployResult = map[string]interface{}{
			"name":        "",
			"pid":         -1,
			"size":        -1,
			"appRepoPath": "",
			"venv":        "",
		}

		procLog.Info.Printf("[DEPLOY-INF] [%d / %d] %s App deploy. \n", appIndex+1, len(deployData.Apps), appName)

		// app download
		filePath, fileSize, cmdErr, appRepoPath, fileZip := fileDownload(appItem.FileUrl, appId, appItem.App, appName, archType)

		if cmdErr != nil {
			procLog.Error.Printf("[DEPLOY-INF] Download Error: %v\n", cmdErr)
			return inferenceResult, cmdErr, http.StatusBadRequest, venv
		}

		// change appname in framework.yaml
		SaveFramework(appName, appId, svcInfo.AppPath)

		// new version
		bwcFramework = GetVenvFromFramework(appName, appId, svcInfo.AppPath)

		// Check exist about app
		if CheckExistApp(appName, svcInfo.RootPath) {
			procLog.Error.Printf("[DEPLOY-INF] %s's app already exist.\n", appName)
			return inferenceResult, errors.New("App already exist."), http.StatusBadRequest, venv
		}

		// Get venv from framework
		if venv == "" || venv == "app-store" { // appstore
			procLog.Info.Printf("[DEPLOY-INF] Deploy app as app-store.")
			venv = bwcFramework.Spec.Env.VirtualEnv
			runTime = bwcFramework.Spec.Env.RunTime
		} else {
			procLog.Info.Printf("[DEPLOY-INF] Deploy app as console.")
			runTime = bwcFramework.Spec.Env.RunTime
		}
		procLog.Info.Printf("[DEPLOY-INF] The runtime is %s.\n", runTime)

		// Only Python
		// Check exist python Virtual Env
		procLog.Info.Printf("[DEPLOY-INF] [%d / %d] Checking if %s venv exists.\n", appIndex+1, len(deployData.Apps), venv)
		envList := GetVenvList(svcInfo.VenvPath)
		if !Contains(envList, venv) {
			// Create Virutal Env
			procLog.Warn.Printf("[DEPLOY-INF] %s's venv not found.\n", venv)
			procLog.Warn.Printf("[DEPLOY-INF] %s's venv install in device.\n", venv)
			venvData := sdtType.CmdVenv{
				VenvName:    venv,
				Requirement: bwcFramework.Spec.Env.Package,
				BinFile:     bwcFramework.Spec.Env.Bin,
				RunTime:     bwcFramework.Spec.Env.RunTime,
			}
			stdout, cmdErr, statusCode := CreateVenv(homeUser, venvData, filePath, svcInfo)
			InstallDefaultPkg(venvData.VenvName, configData.DeviceType, configData.ServiceType, svcInfo)

			if cmdErr != nil {
				procLog.Error.Printf("[DEPLOY-INF] Failed download python pkg.\n")
				cmdErr = errors.New(string(stdout))
				return inferenceResult, cmdErr, statusCode, venv
			}

			newUUID := uuid.New()
			requestId := newUUID.String()

			// 결과 메시지 생성
			cmdResult := sdtType.NewCmdResult("virtualEnv", "venvCreate", stdout)
			cmdResult.VenvName = venvData.VenvName

			cmdStatus := sdtType.NewCmdStatus(statusCode)
			if cmdErr == nil {
				cmdStatus.ErrMsg = ""
				cmdStatus.Succeed = 1
			} else {
				cmdStatus.ErrMsg = fmt.Sprintf("%v", cmdErr)
				cmdStatus.Succeed = 0
			}

			result := sdtType.ResultMsg{
				AssetCode: configData.AssetCode,
				Result:    &cmdResult,
				Status:    cmdStatus,
				RequestId: requestId,
			}

			topic := fmt.Sprintf("%s/%s/%s/bwc/control/self-deploy", configData.ServiceCode, configData.ProjectCode, configData.AssetCode)
			sdtMessage.SendDataEdgeMqtt(result, topic, cli)

		}

		// APP info 저장
		// save Inference deploy json
		SaveAppInfo(appName, appId, venv, "systemd", deployData.Apps[appIndex], deployData.AppGroupId, svcInfo.RootPath)

		CreatePythonService(filePath, appName, venv, "main.py", svcInfo.VenvPath)

		// Inference와 Request APP 구분
		if appItem.AppType == "INFERENCE" {
			cmdErr = CreateInferenceDir(filePath)
			if cmdErr != nil {
				procLog.Error.Printf("[DEPLOY-INF] Failed creating inference directory.\n")
				return inferenceResult, cmdErr, http.StatusBadRequest, venv
			}

			procLog.Info.Printf("[DEPLOY-INF] [%d / %d] App is inference. So, device download weight file.\n", appIndex+1, len(deployData.Apps))
			//modelFileName = appItem.Parameter[appItem.ModelFileKey].(string)
			keys := strings.Split(appItem.ModelFileKey, ".")
			fileDict := appItem.Parameter
			for n := 0; n < len(keys)-1; n++ {
				fileDict = fileDict[keys[n]].(map[string]interface{})
			}

			modelFileName = fileDict[keys[len(keys)-1]].(string)
			cmdErr = DownloadWeight_new(appItem.ModelUrl, appName, appId, modelFileName, svcInfo.AppPath)
			if cmdErr != nil {
				procLog.Error.Printf("[DEPLOY-INF] Failed download inference model.\n")
				return inferenceResult, cmdErr, http.StatusBadRequest, venv
			}

			// Apply config
			// Common Parameter
			parameter := sdtType.CmdJson{
				AppId:     appId,
				AppName:   appName,
				FileName:  "config.json", // BW와 약속한 config 파일 이름
				Parameter: appItem.Parameter,
			}

			_, configErr, _ := sdtConfig.JsonChange(parameter, svcInfo.AppPath, "")
			if configErr != nil {
				procLog.Error.Printf("[DEPLOY-INF] Failed fixing parameter.\n")
				return inferenceResult, configErr, http.StatusBadRequest, venv
			}
		} else if appItem.AppType == "REQUEST" {
			procLog.Info.Printf("[DEPLOY-INF] [%d / %d] App is request.\n", appIndex+1, len(deployData.Apps))
		} else {
			procLog.Error.Printf("[DEPLOY-INF] %s: Invalid app type. \n", appItem.AppType)
			return inferenceResult, errors.New(fmt.Sprintf("%s: Invalid app type. \n", appItem.AppType)), http.StatusBadRequest, venv
		}

		// start systemd
		startCmd := fmt.Sprintf("systemctl start %s", appName)
		cmd_run := exec.Command("sh", "-c", startCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Println("[DEPLOY-INF] Fail deploy: ", cmd_err, "\n", string(stdout))
			cmd_err = errors.New(string(stdout))
			return inferenceResult, errors.New(string(stdout)), http.StatusBadRequest, venv
		}

		// enable systemd
		startCmd = fmt.Sprintf("systemctl enable %s", appName)
		cmd_run = exec.Command("sh", "-c", startCmd)
		stdout, cmd_err = cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Println("[DEPLOY] Fail deploy: ", cmd_err, "\n", string(stdout))
			cmd_err = errors.New(string(stdout))
			return inferenceResult, errors.New(string(stdout)), http.StatusBadRequest, venv
		}
		procLog.Info.Println("[DEPLOY-INF] End Deploy..")

		// get pid
		pid, cmd_err, _ = GetPid(appName)
		if cmd_err != nil {
			// get error log
			logResult := GetLogsApp(svcInfo.AppPath, appName, appId)
			//cmd_log := exec.Command("journalctl", "-u", appName, "-n", "30")
			//stdout, _ = cmd_log.CombinedOutput()
			return inferenceResult, errors.New(logResult), http.StatusBadRequest, venv
		}

		procLog.Info.Printf("[DEPLOY-INF] Remove ZIP File: %s\n", fileZip)
		removeErr := os.Remove(fileZip)
		if removeErr != nil {
			time.Sleep(1)
			os.Remove(fileZip)
		}
		procLog.Warn.Printf("[DEPLOY-INF] REMOVE: %s\n", removeErr)

		//deployResult = map[string]interface{}{
		//	"name":        appName,
		//	"pid":         pid,
		//	"size":        fileSize,
		//	"appRepoPath": appRepoPath,
		//	"venv":        venv,
		//}
		deployResult["name"] = appName
		deployResult["pid"] = pid
		deployResult["size"] = fileSize
		deployResult["appRepoPath"] = appRepoPath
		deployResult["venv"] = venv

		inferenceResult = append(inferenceResult, deployResult)

	}

	return inferenceResult, cmdErr, http.StatusOK, venv
}

// The GetPid function retrieves the PID of an application deployed on the device.
//
// Input:
//   - appName: Name of the application.
//
// Output:
//   - int: PID of the application.
//   - error: Error message in case of issues with the getPid command.
//   - int: Status code of the command execution.
func GetPid(appName string) (int, error, int) {
	// TO DO
	//   - 다수의 앱에 대한 PID 값을 주는 방법
	getPid := fmt.Sprintf("systemctl show --property MainPID %s", appName)
	cmd_run := exec.Command("sh", "-c", getPid)
	stdout, err := cmd_run.CombinedOutput()
	if err != nil {
		procLog.Error.Printf("[DEPLOY] Get pid error: %s\n", stdout)
		return -1, errors.New(string(stdout)), http.StatusBadRequest
	}
	strOut := string(stdout)
	pidStr := strings.Split(strOut[:len(strOut)-1], "=")
	if pidStr[len(pidStr)-1] == "" {
		procLog.Error.Println("[DEPLOY] Not found process: ")
		err = errors.New("Not found process")
		return -1, err, http.StatusBadRequest
	}
	pid, err := strconv.Atoi(pidStr[len(pidStr)-1])
	if err != nil {
		procLog.Error.Println("[DEPLOY] Convert pid (string -> int) error: ", err)
		return -1, err, http.StatusBadRequest
	} else if pid == 0 {
		procLog.Error.Println("[DEPLOY] Not found process: ")
		err = errors.New("Not found process")
		return -1, err, http.StatusBadRequest
	}
	return pid, err, http.StatusOK
}

// The DeleteVenv function deletes a Python virtual environment installed on the device.
// Virtual environments are used for managing dependencies for python applications.
//
// Input:
//   - envName: Name of the virtual environment.
//
// Output:
//   - string: Result message of the operation.
//   - error: Error message in case of issues with the deleteVenv command.
//   - int: Status code of the command execution.
func DeleteVenv(envName string, svcInfo sdtType.ControlService) (string, error, int) {
	if envName == "" || envName == "*" {
		cmd_err := errors.New("Not collect value.")
		procLog.Error.Printf("[DELETE-ENV] Not collect value: %s\n", envName)
		return "", cmd_err, http.StatusBadRequest
	}

	targetEnv := fmt.Sprintf("%s/%s", svcInfo.VenvPath, envName)
	cmd_err := os.RemoveAll(targetEnv)
	if cmd_err != nil {
		procLog.Error.Printf("[DELETE-ENV] V-ENV delete failed: %v\n", cmd_err)
		return "", cmd_err, http.StatusBadRequest
	}

	venvResult := "Deleted V-Env."
	return venvResult, cmd_err, http.StatusOK
}

// The CreateBaseVenv function creates a base (default) virtual environment.
// The base virtual environment is a pre-defined Python environment provided by default.
// If a specific virtual environment is not selected, the app will be deployed in the base virtual environment.
//
// Input:
//   - homeUser: Hostname of the device.
func CreateBaseVenv(homeUser string, svcInfo sdtType.ControlService) {
	envName := "base"
	envHome := svcInfo.VenvPath

	if _, err := os.Stat(envHome); os.IsNotExist(err) {
		os.Mkdir(envHome, os.ModePerm)
	}

	// Check Base Venv
	envDir, _ := ioutil.ReadDir(envHome)
	for _, f := range envDir {
		if f.IsDir() {
			if f.Name() == "base" {
				procLog.Info.Printf("[CREATE-ENV-BASE] Already exist base Venv.\n")
				return
			}
		}
	}

	envPath := fmt.Sprintf("%s/%s", envHome, envName)

	// create base miniconda3's venv
	var createEnvCmd string
	if svcInfo.ArchType == "win" {
		createEnvCmd = fmt.Sprintf("%s/../python -m venv %s", svcInfo.MinicondaPath, envPath)
	} else {
		if homeUser == "root" {
			createEnvCmd = fmt.Sprintf("/root/miniconda3/bin/python -m venv %s", envPath)
		} else {
			createEnvCmd = fmt.Sprintf("%s/python -m venv %s", svcInfo.MinicondaPath, envPath)
		}
	}

	cmd_run := exec.Command(svcInfo.BaseCmd[0], svcInfo.BaseCmd[1], createEnvCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Warn.Printf("[INIT-ENV] Create Base VENV(Miniconda3) Error: %s\n", stdout)
	} else {
		procLog.Info.Printf("[INIT-ENV] Create Base VENV(Miniconda3).\n")
		return
	}

	// Check bin file
	procLog.Warn.Printf("[INIT-ENV] Create base VENV(base-python3). Please install miniconda3.\n")
	var binFile string
	binFile = "/usr/bin/python"
	if _, err := os.Stat(binFile); os.IsNotExist(err) {
		binFile = "/usr/bin/python3"
	}

	createEnvCmd = fmt.Sprintf("%s -m venv %s", binFile, envPath)

	cmd_run = exec.Command("sh", "-c", createEnvCmd)
	stdout, cmd_err = cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Warn.Printf("[INIT-ENV] Create Base VENV Error: %s\n", stdout)
		return
	}
	procLog.Info.Printf("[INIT-ENV] Create Base VENV\n")
}

// The CreateVenv function creates a python virtual environment on the device.
// Virtual environments are isolated spaces used for python applications to manage package dependencies.
// They are specific to python applications.
//
// Input:
//   - homeUser: Hostname of the device.
//   - venvData: Struct containing virtual environment control command information.
//
// Output:
//   - string: Processing result message.
//   - error: Error message of the createVenv command.
//   - int: Command processing status.
func CreateVenv(homeUser string, venvData sdtType.CmdVenv, appDir string, svcInfo sdtType.ControlService) (string, error, int) {
	envName := venvData.VenvName
	pkgInfo := venvData.Requirement
	envHome := svcInfo.VenvPath
	minicondaPath := svcInfo.MinicondaPath

	if _, err := os.Stat(envHome); os.IsNotExist(err) {
		os.Mkdir(envHome, os.ModePerm)
	}

	envPath := fmt.Sprintf("%s/%s", envHome, envName)
	var createEnvCmd string

	runTimeVersion := strings.Replace(venvData.RunTime, "python", "", -1)
	procLog.Info.Printf("[CREATE-ENV] Create venv(=%s).\n", runTimeVersion)
	if homeUser == "root" {
		createEnvCmd = fmt.Sprintf("root/miniconda3/bin/conda create -p %s python=%s -y", envPath, runTimeVersion)
	} else {
		createEnvCmd = fmt.Sprintf("%s/conda create -p %s python=%s -y", minicondaPath, envPath, runTimeVersion)
	}

	//cmd_run := exec.Command("sh", "-c", createEnvCmd)
	cmd_run := exec.Command(svcInfo.BaseCmd[0], svcInfo.BaseCmd[1], createEnvCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("[CREATE-ENV] Create V-ENV Error: %s\n", stdout)
		return "", errors.New(string(stdout)), http.StatusBadRequest
	}
	procLog.Info.Printf("[CREATE-ENV] Create V-ENV.\n")

	// create requirement.txt
	pkgFileName := fmt.Sprintf("%s/requirements.txt", envPath)
	if appDir != "" { // Deploy from app-store.
		CopyFile(fmt.Sprintf("%s/%s", appDir, pkgInfo), pkgFileName)
	} else {
		pkgFile, err := os.Create(pkgFileName)
		if err != nil {
			procLog.Error.Printf("[CREATE-ENV] Create requirement file Error: %v\n", cmd_err)
			return "", cmd_err, http.StatusBadRequest
		}
		defer pkgFile.Close()

		// Split Package name...
		//  - " " 스페이스바, # 주석 라인 전처리 작업(삭제)
		pkgString := strings.Split(pkgInfo, "\n")
		for _, vals := range pkgString {
			checkString := strings.ReplaceAll(vals, " ", "")
			if len(vals) == 0 || string(checkString[0]) == "#" {
				continue
			}
			content := fmt.Sprintf("%s\n", vals)
			_, err = pkgFile.WriteString(content)
			if err != nil {
				procLog.Error.Printf("[CREATE-ENV] Write requirement file Error: %v\n", cmd_err)
				return "", cmd_err, http.StatusBadRequest
			}
		}
	}

	procLog.Info.Printf("[CREATE-ENV] Install app's pkg.\n")

	// install pkg
	var outBuffer, errBuffer bytes.Buffer
	var pkgCmd string
	if svcInfo.ArchType == "win" {
		pkgCmd = fmt.Sprintf("%s/Scripts/pip install -r %s", envPath, pkgFileName)
	} else {
		pkgCmd = fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	}
	//cmd_run = exec.Command("sh", "-c", pkgCmd)
	cmd_run = exec.Command(svcInfo.BaseCmd[0], svcInfo.BaseCmd[1], pkgCmd)
	cmd_run.Stdout = &outBuffer
	cmd_run.Stderr = &errBuffer

	cmd_err = cmd_run.Run()

	if cmd_err != nil {
		errContent := errors.New(errBuffer.String())
		procLog.Error.Printf("[CREATE-ENV] Install package Error: %v, %v\n", cmd_err, errContent)
		return "", errContent, http.StatusBadRequest
	}
	procLog.Info.Printf("[CREATE-ENV] Installed V-ENV's package.\n")
	venvResult := "Created V-Env."
	return venvResult, cmd_err, http.StatusOK
}

// New version(only miniconda)
func CreateVenv_newVersion(homeUser string, venvData sdtType.CmdVenv, configData sdtType.ConfigInfo) (string, error, int) {
	// Set Variable
	var homePath, runTimeVersion, createEnvCmd string
	envName := venvData.VenvName
	pkgInfo := venvData.Requirement
	envHome := "/etc/sdt/venv"
	envPath := fmt.Sprintf("%s/%s", envHome, envName)
	if homeUser == "root" {
		homePath = "/root"
	} else {
		homePath = fmt.Sprintf("/home/%s", homeUser)
	}

	// Set Runtime Version
	runTimeVersion = strings.Replace(venvData.BinFile, "python", "", -1)

	if _, err := os.Stat(envHome); os.IsNotExist(err) {
		os.Mkdir(envHome, os.ModePerm)
	}

	if configData.DeviceType == "nodeq" {
		setErr := errors.New("[Cannot create virtual env in nodeQ.")
		procLog.Error.Printf("[CREATE-ENV] %v\n", setErr)
		return "", setErr, http.StatusBadRequest
	} else if envName == "base" {
		setErr := errors.New("Already create 'base' venv.")
		procLog.Error.Printf("[CREATE-ENV] %v\n", setErr)
		return "", setErr, http.StatusBadRequest
	} else {
		createEnvCmd = fmt.Sprintf("%s/miniconda3/bin/conda create -p %s python=%s -y", homePath, envPath, runTimeVersion)
		cmd_run := exec.Command("sh", "-c", createEnvCmd)
		stdout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[CREATE-ENV] [%v] %s\n", cmd_err, stdout)
			return "", errors.New(string(stdout)), http.StatusBadRequest
		}
		fmt.Printf("Create V-ENV.\n")
	}

	cmd_run := exec.Command("sh", "-c", createEnvCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("[CREATE-ENV] Create V-ENV Error: [%v] %s\n", cmd_err, stdout)
		return "", errors.New(string(stdout)), http.StatusBadRequest
	}
	procLog.Info.Printf("[CREATE-ENV] Create V-ENV\n")

	// create requirement.txt
	pkgFileName := fmt.Sprintf("%s/requirements.txt", envPath)
	pkgFile, err := os.Create(pkgFileName)
	if err != nil {
		procLog.Error.Printf("[CREATE-ENV] Create requirement file Error: %v\n", err)
		return "", err, http.StatusBadRequest
	}
	defer pkgFile.Close()

	// Split Package name...
	pkgString := strings.Split(pkgInfo, "\n")
	for _, vals := range pkgString {
		checkString := strings.ReplaceAll(vals, " ", "")
		if len(vals) == 0 || string(checkString[0]) == "#" {
			continue
		}
		content := fmt.Sprintf("%s\n", vals)
		_, err = pkgFile.WriteString(content)
		if err != nil {
			procLog.Error.Printf("[CREATE-ENV] Write requirement file Error: %v\n", err)
			return "", err, http.StatusBadRequest
		}
	}
	procLog.Info.Printf("[CREATE-ENV] Create pkg file.\n")

	// install pkg
	var outBuffer, errBuffer bytes.Buffer
	pkgCmd := fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	cmd_run = exec.Command("sh", "-c", pkgCmd)
	cmd_run.Stdout = &outBuffer
	cmd_run.Stderr = &errBuffer

	cmd_err = cmd_run.Run()

	if cmd_err != nil {
		errContent := errors.New(errBuffer.String())
		procLog.Error.Printf("[CREATE-ENV] Install package Error: %v, %v\n", cmd_err, errContent)
		return "", errContent, http.StatusBadRequest
	}
	procLog.Info.Printf("[CREATE-ENV] Installed V-ENV's package.\n")
	venvResult := "Created V-Env."
	return venvResult, cmd_err, http.StatusOK
}

// The UpdateVenv function updates a python virtual environment on the device.
// Updating a virtual environment involves installing or uninstalling packages within the environment.
//
// Input:
//   - venvData: Struct containing virtual environment control command information.
//
// Output:
//   - string: Processing result message.
//   - error: Error message of the updateVenv command.
//   - int: Command processing status.
func UpdateVenv(venvData sdtType.CmdVenv, svcInfo sdtType.ControlService) (string, error, int) {
	envPath := fmt.Sprintf("%s/%s", svcInfo.VenvPath, venvData.VenvName)

	// Install pkg
	pkgFileName := fmt.Sprintf("%s/requirements.txt", envPath)
	pkgFile, err := os.Create(pkgFileName)
	//pkgFile, err := os.OpenFile(pkgFileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)

	if err != nil {
		procLog.Error.Printf("[UPDATE-ENV] Create requirement file Error: %v\n", err)
		return "", err, http.StatusBadRequest
	}
	defer pkgFile.Close()

	// Split Package name...
	pkgString := strings.Split(venvData.Requirement, "\n")
	for _, vals := range pkgString {
		checkString := strings.ReplaceAll(vals, " ", "")
		if len(vals) == 0 || string(checkString[0]) == "#" {
			continue
		} else if string(checkString[0]) == "-" {
			replaceVal := strings.ReplaceAll(vals, "-", "")
			UninstallPkg(envPath, replaceVal)
		} else {
			content := fmt.Sprintf("%s\n", vals)
			_, err = pkgFile.WriteString(content)
			if err != nil {
				procLog.Error.Printf("[UPDATE-ENV] Write requirement file Error: %v\n", err)
				return "", err, http.StatusBadRequest
			}
		}
	}
	procLog.Info.Printf("[UPDATE-ENV] Create pkg file.\n")

	// install pkg
	procLog.Info.Printf("[UPDATE-ENV] Install package\n")
	pkgCmd := fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)
	cmd_run := exec.Command("sh", "-c", pkgCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()

	if cmd_err != nil {
		errContent := errors.New(string(stdout))
		procLog.Error.Printf("[UPDATE-ENV] Update package Error: %v, %v\n", cmd_err, errContent)
		return "", errContent, http.StatusBadRequest
	}
	procLog.Info.Printf("[UPDATE-ENV] Updated V-ENV's package.\n")

	updateResult := "Updated V-Env's packages."
	return updateResult, cmd_err, http.StatusOK
}

// The fileDownload function downloads an application file from a code repository.
// The application is installed in the "/usr/local/sdt/app" directory.
//
// Input:
//   - fullURLFile: URI of the application file to download.
//   - appId: Application ID.
//   - app: Application name stored in the code repository.
//   - appName: Name of the application to deploy.
//   - archType: Device architecture.
//
// Output:
//   - string: Path of the installed application on the device.
//   - int64: Size of the application.
//   - error: Error message in string format.
//   - string: Path of the application in the code repository.
//   - string: Path of the application's zip file.(Byte)
func fileDownload(
	fullURLFile string,
	appId string, //App's ID
	app string, //App's name
	appName string, // local app's name
	archType string, // arch -> linux or window
) (string, int64, error, string, string) {
	// Build fileName from fullPath
	fileURL, err := url.Parse(fullURLFile)
	if err != nil {
		procLog.Error.Println("[DEPLOY] fileDownload URL parse error: ", err)
	}
	path := fileURL.Path
	segments := strings.Split(path, "/")
	fileName := segments[len(segments)-1]
	appRepoPath := fmt.Sprintf("%s://%s/%s/%s:%s\n", fileURL.Scheme,
		fileURL.Host,
		segments[1],
		segments[2],
		strings.Split(fileName, ".zip")[0])
	// Create blank file
	var appDir string
	if archType == "win" {
		appDir = "C:/sdt/app"
	} else {
		appDir = "/usr/local/sdt/app"
	}
	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		os.Mkdir(appDir, os.ModePerm)
	}

	fileZip := fmt.Sprintf("%s/%s", appDir, fileName)
	file, err := os.Create(fileZip)
	if err != nil {
		procLog.Error.Println("[DEPLOY] fileDownload file creation error: ", err)
	}
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}
	// Put content on file
	resp, err := client.Get(fullURLFile)
	if err != nil {
		procLog.Error.Println("[DEPLOY] fileDownload URL get file error: ", err)
	}
	defer resp.Body.Close()

	if resp.Status[:3] != "200" {
		return fileZip, 0, errors.New(fmt.Sprintf("Download error: %s", resp.Status)), appRepoPath, fileZip
	}

	_, err = io.Copy(file, resp.Body)

	defer file.Close()

	// unzip!!
	// appPath -> usr/local/sdt/app/{app's Name}
	// app's Name -> {appName}_{appId}
	appPath := fmt.Sprintf("%s/%s_%s", appDir, appName, appId)
	zipPath := fmt.Sprintf("%s/%s", appDir, app)
	targetPath := fmt.Sprintf("%s", appDir)

	err = archiver.Unarchive(fileZip, targetPath)
	if err != nil {
		procLog.Error.Println("[DEPLOY] Unzip error: ", err)
	}

	// file rename
	procLog.Info.Println("[DEPLOY] ZIP File Path: ", zipPath)
	procLog.Info.Println("[DEPLOY] APP File Path: ", appPath)
	os.Rename(zipPath, appPath)

	// get file size
	fileInfo, _ := os.Stat(fileZip)
	fileSize := fileInfo.Size() // Byte

	// for {
	// 	if _, err = os.Stat(appPath); os.IsExist(err) {
	// 		break
	// 	}
	// 	procLog.Warn.Printf("[DEPLOY] APP File not created: %s / %v", appPath, err)
	// 	time.Sleep(1)
	// }

	// remove zip file
	procLog.Info.Println("[DEPLOY] Remove ZIP File: ", fileZip)
	removeErr := os.Remove(fileZip)
	if removeErr != nil {
		time.Sleep(1)
		os.Remove(fileZip)
	}

	return appPath, fileSize, nil, appRepoPath, fileZip
}

// CreateGoService function creates a Systemd file (.service) for a Golang application.
//
// Input:
//   - appDir: The directory on the device where the app will be installed.
//   - appName: The name of the application.
//   - runCmd: The command to execute the application.
func CreateGoService(appDir string, appName string, runCmd string) {
	// Specify the file name and path
	filePath := fmt.Sprintf("%s/%s.service", appDir, appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Errorf("error creating service file : %v", err)
		return
	}
	defer file.Close()

	// Write content to the file
	// Remove: Environment=PATH=/etc/sdt/venv/%s/bin:$PATH
	content := fmt.Sprintf(`[Unit]
Description=%s

[Service]
WorkingDirectory=%s
ExecStart=%s/%s
Restart=always
RestartSec=10
StandardOutput=file:/%s/app.log
StandardError=file:/%s/app-error.log

[Install]
WantedBy=multi-user.target
	`, appName, appDir, appDir, runCmd, appDir, appDir)
	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Println("Error writing to the file:", err)
		return
	}

	svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	err = CopyFile(filePath, svcFile)

	if err != nil {
		procLog.Error.Println("Error svc file copy to systemd:", err)
		return
	}
}

// CreatePythonService function creates a systemd file (.service) for a Python application.
//
// Input:
//   - appDir: The directory on the device where the app will be installed.
//   - appName: The name of the application.
//   - appVenv: The virtual environment name for the application.
//   - runCmd: The command to execute the application.
func CreatePythonService(appDir string, appName string, appVenv string, runCmd string, venvPath string) {
	// Specify the file name and path
	filePath := fmt.Sprintf("%s/%s.service", appDir, appName)

	// Create a new file or truncate an existing file
	file, err := os.Create(filePath)
	if err != nil {
		fmt.Errorf("error creating service file : %v", err)
		return
	}
	defer file.Close()

	// Set Python Exec bin
	var execBin string
	if appVenv == "base" {
		//execBin = "/usr/bin/python3"
		execBin = fmt.Sprintf("%s/base/bin/python", venvPath)
	} else {
		execBin = fmt.Sprintf("%s/%s/bin/python", venvPath, appVenv)
	}

	// Write content to the file
	// Remove: Environment=PATH=/etc/sdt/venv/%s/bin:$PATH
	content := fmt.Sprintf(`[Unit]
Description=%s

[Service]
WorkingDirectory=%s
ExecStart=%s %s
Restart=always
RestartSec=10
StandardOutput=file:/%s/app.log
StandardError=file:/%s/app-error.log

[Install]
WantedBy=multi-user.target
	`, appName, appDir, execBin, runCmd, appDir, appDir)
	_, err = file.WriteString(content)
	if err != nil {
		procLog.Error.Println("Error writing to the file:", err)
		return
	}

	svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	err = CopyFile(filePath, svcFile)

	if err != nil {
		procLog.Error.Println("Error svc file copy to systemd:", err)
		return
	}
}

// CopyFile function copies a directory or file.
//
// Input:
//   - srcPath: The path to the source file or directory to be copied.
//   - destPath: The destination path where the file or directory will be copied.
func CopyFile(srcPath, destPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("error opening source file: %v", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating destination file: %v", err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("error copying file content: %v", err)
	}

	return nil
}

// SaveAppInfo function saves metadata of the deployed app on the device.
// BWC manages metadata of deployed apps as a Json file on the device.
//
// Input:
//   - appName: The name of the app.
//   - appId: The ID of the app.
//   - appVenv: The virtual environment used by the app.
func SaveAppInfo(appName string,
	appId string,
	appVenv string,
	appManaged string,
	inferenceInfo sdtType.InferenceDeploy,
	groupId string,
	rootPath string,
) {

	appInfoFile := fmt.Sprintf("%s/device.config/app.json", rootPath)
	// check app's info file
	if _, err := os.Stat(appInfoFile); os.IsNotExist(err) {
		appInfo := []sdtType.AppInfo{
			{
				AppName: appName,
				AppId:   appId,
				AppVenv: appVenv,
				Managed: appManaged,
				AppInference: &sdtType.AppInferenceInfo{
					ModelId:      inferenceInfo.ModelId,
					ModelName:    inferenceInfo.ModelName,
					ModelVersion: inferenceInfo.ModelVersion,
				},
				AppGroupId: groupId,
			},
		}
		jsonData := sdtType.AppConfig{
			AppInfoList: appInfo,
		}
		saveJson, err := json.MarshalIndent(jsonData, "", "\t")
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed save app's Marshal: %v\n", err)
		}

		err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed save app's info: %v\n", err)
		}
	} else {
		jsonFile, err := ioutil.ReadFile(appInfoFile)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed load app's file: %v\n", err)
		}
		var jsonData sdtType.AppConfig
		err = json.Unmarshal(jsonFile, &jsonData)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed save app's Unmarshal: %v\n", err)
		}

		newApp := sdtType.AppInfo{
			AppName: appName,
			AppId:   appId,
			AppVenv: appVenv,
			Managed: appManaged,
			AppInference: &sdtType.AppInferenceInfo{
				ModelId:      inferenceInfo.ModelId,
				ModelName:    inferenceInfo.ModelName,
				ModelVersion: inferenceInfo.ModelVersion,
			},
			AppGroupId: groupId,
		}

		jsonData.AppInfoList = append(jsonData.AppInfoList, newApp)
		saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
		err = ioutil.WriteFile(appInfoFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("[DEPLOY] failed save app's file: %v\n", err)
		}
	}
}

// SaveFramework function updates the content of the framework file that manages app metadata.
// The framework file contains information such as app name, runtime, and code repository needed
// when deploying the app. When deploying an app from the SDT Cloud console, it's possible to deploy
// with a different app name, which can cause the app name in the Framework file to be out of sync.
// This function updates the app name to synchronize it.
//
// Input:
//   - appName: The name of the app.
//   - appId: The ID of the app.
func SaveFramework(appName string, appId string, appPath string) {
	// Change appName in Framework.yaml
	var bwcFramework sdtType.Framework
	frameworkFile := fmt.Sprintf("%s/%s_%s/framework.yaml", appPath, appName, appId)
	_, err := os.Stat(frameworkFile)

	if os.IsNotExist(err) {
		procLog.Warn.Printf("[DEPLAY] Find framework.json.\n")
		frameworkFile = fmt.Sprintf("%s/%s_%s/framework.json", appPath, appName, appId)
		jsonFile, err := ioutil.ReadFile(frameworkFile)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed load framework's file: %v\n", err)
			return
		}

		err = json.Unmarshal(jsonFile, &bwcFramework)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed save framework's Unmarshal: %v\n", err)
			return
		}

		bwcFramework.Spec.AppName = appName
		saveJson, _ := json.MarshalIndent(&bwcFramework, "", "\t")
		err = ioutil.WriteFile(frameworkFile, saveJson, 0644)
		if err != nil {
			procLog.Error.Printf("[DEPLOY] failed save framework's file: %v\n", err)
		}
		return
	}

	// check app's info file
	procLog.Warn.Printf("[DEPLAY] Find framework.yaml.\n")
	yamlFile, err := ioutil.ReadFile(frameworkFile)
	if err != nil {
		procLog.Error.Printf("[DEPLAY] Failed load framework's file: %v\n", err)
		return
	}

	err = yaml.Unmarshal(yamlFile, &bwcFramework)
	if err != nil {
		procLog.Error.Printf("[DEPLAY] Failed save framework's Unmarshal: %v\n", err)
		return
	}

	bwcFramework.Spec.AppName = appName
	saveYaml, _ := yaml.Marshal(&bwcFramework)
	err = ioutil.WriteFile(frameworkFile, saveYaml, 0644)
	if err != nil {
		procLog.Error.Printf("[DEPLOY] failed save framework's file: %v\n", err)
		return
	}

}

// GetVenvFromFramework function reads the Framework file to retrieve information about
// the virtual environment (venv) and runtime.
//
// Input:
//   - appName: The name of the app.
//   - appId: The ID of the app.
//
// Output:
//   - string: The name of the virtual environment (venv).
//   - string: The runtime value.
func GetVenvFromFramework(appName string, appId string, appPath string) sdtType.Framework {
	var bwcFramework sdtType.Framework

	// Change appName in Framework.yaml
	procLog.Info.Printf("[DEPLAY] Find framework.yaml.\n")
	frameworkFile := fmt.Sprintf("%s/%s_%s/framework.yaml", appPath, appName, appId)
	_, err := os.Stat(frameworkFile)

	// check app's info json(New version)
	if os.IsNotExist(err) {
		procLog.Info.Printf("[DEPLAY] Failed read framework.yaml.\n")
		procLog.Warn.Printf("[DEPLAY] Find framework.json.\n")
		frameworkFile = fmt.Sprintf("%s/%s_%s/framework.json", appPath, appName, appId)
		jsonFile, err := ioutil.ReadFile(frameworkFile)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed load framework's file: %v\n", err)
			return bwcFramework
		}

		err = json.Unmarshal(jsonFile, &bwcFramework)
		if err != nil {
			procLog.Error.Printf("[DEPLAY] Failed save framework's Unmarshal: %v\n", err)
			return bwcFramework
		}

		return bwcFramework
	}

	// check app's info file(yaml)
	procLog.Warn.Printf("[DEPLAY] Find framework.yaml.\n")
	yamlFile, err := ioutil.ReadFile(frameworkFile)
	if err != nil {
		procLog.Error.Printf("[DEPLAY] Failed load framework's file: %v\n", err)
		return bwcFramework
	}
	err = yaml.Unmarshal(yamlFile, &bwcFramework)
	if err != nil {
		procLog.Error.Printf("[DEPLAY] Failed save framework's Unmarshal: %v\n", err)
		return bwcFramework
	}

	return bwcFramework

}

// DeleteAppInfo function deletes the app's metadata from the device when the app is being deleted.
//
// Input:
//   - appName: The name of the app to be deleted.
func DeleteAppInfo(appName string, rootPath string) {
	var appId string = ""
	appInfoFile := fmt.Sprintf("%s/device.config/app.json", rootPath)

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("[DELETE] Failed load app's info: %v\n", err)
		return
	}
	var jsonData, saveData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("[DELETE] Failed delete app's Unmarshal: %v\n", err)
		return
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
		procLog.Error.Printf("[DELETE] failed delete app's info: %v\n", err)
		return
	}
	procLog.Warn.Printf("[DELETE] delete app's info: %s\n", appId)
}

// GetAppsFromGroup function deletes the app's metadata from the device when the app is being deleted.
//
// Input:
//   - appName: The name of the app to be deleted.
func GetAppsFromGroup(appGroupId string, rootPath string) ([]string, []string) {
	var appNames, appIds []string
	appInfoFile := fmt.Sprintf("%s/device.config/app.json", rootPath)

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("[GET-APPS] Failed load app's info: %v\n", err)
		return appNames, appIds
	}
	var jsonData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("[GET-APPS] Failed get app's Unmarshal: %v\n", err)
		return appNames, appIds
	}

	for _, val := range jsonData.AppInfoList {
		if val.AppGroupId == appGroupId {
			appNames = append(appNames, val.AppName)
			appIds = append(appIds, val.AppId)
			continue
		}
	}
	procLog.Warn.Printf("[GET-APPS] app's info: %s / %s\n", appNames, appIds)
	return appNames, appIds
}

// GetVenvList function retrieves the list of virtual environments installed on the device.
//
// Output:
//   - []string: List of virtual environments.
func GetVenvList(venvPath string) []string {
	var envList []string
	envDir, _ := ioutil.ReadDir(venvPath)

	for _, f := range envDir {
		if f.IsDir() {
			envList = append(envList, f.Name())
		}
	}
	return envList
}

// Contains function checks if a specific string exists in a list of strings.
//
// Input:
//   - elems: List of strings.
//   - v: String to check for.
//
// Output:
//   - bool: True if the string exists in the list, false otherwise.
func Contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}

// UninstallPkg function uninstalls a package from a virtual environment.
//
// Input:
//   - envPath: The path to the virtual environment.
//   - pkgName: The name of the package to uninstall.
//
// Output:
//   - error: An error message if package uninstallation fails.
func UninstallPkg(envPath string, pkgName string) error {
	pkgCmd := fmt.Sprintf("%s/bin/pip uninstall -y %s", envPath, pkgName)
	procLog.Info.Printf("[Uninstall-PKG] Uninstall package: %s \n", pkgName)
	cmd_run := exec.Command("sh", "-c", pkgCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		procLog.Error.Printf("[UPDATE-ENV] Uninstall package Error: [%v] %s\n", cmd_err, stdout)
		return errors.New(string(stdout))
	}
	procLog.Info.Printf("[UPDATE-ENV] Uninstalled package\n")
	return cmd_err
}

// InstallDefaultPkg function installs default packages provided by SDT Cloud into a virtual environment.
// Default packages provided by SDT Cloud include MQTT, S3, and MQTTforNodeQ.
// These packages facilitate communication with various services.
//
// Input:
//   - venvName: The name of the virtual environment.
//   - sdtCloudIP: The IP address of SDT Cloud.
//   - giteaPort: The port number of the code repository.
//   - deviceType: The type of the device (ECN, NodeQ).
func InstallDefaultPkg(venvName string,
	//sdtCloudIP string,
	//giteaPort int,
	deviceType string,
	serviceType string,
	svcInfo sdtType.ControlService) {
	var pkgCmd, pipPath, pkgLink string
	var pkgList []string
	procLog.Info.Printf("[VENV-Base-PKG] Install base package.\n")
	if svcInfo.ArchType == "win" {
		pipPath = fmt.Sprintf("%s/%s/Scripts/pip3", svcInfo.VenvPath, venvName)
	} else {
		pipPath = fmt.Sprintf("%s/%s/bin/pip3", svcInfo.VenvPath, venvName)
	}

	// Set Pip Path
	pkgLink = fmt.Sprintf("%s install --trusted-host %s --index-url http://%s:%d/api/packages/app.manager/pypi/simple/", pipPath, svcInfo.SdtcloudIP, svcInfo.SdtcloudIP, svcInfo.GiteaPort)

	if deviceType == "nodeq" {
		pkgList = []string{"sdtcloudnodeqmqtt", "sdtcloud"}
	} else if deviceType == "ecn" {
		pkgList = []string{"sdtcloudpubsub", "sdtclouds3", "sdtcloud", "sdtcloudwin"}
	} else if deviceType == "aquarack" {
		pkgList = []string{"sdtcloudpubsub", "sdtclouds3", "sdtcloud", "sdtcloudwin"}
	} else {
		procLog.Error.Printf("[VENV-Base-PKG] Not found type.[%s]\n", deviceType)
	}

	if serviceType == "onprem" {
		procLog.Info.Printf("This service type is %s, so download onprem's pkg..\n", serviceType)
		pkgList = []string{"sdtcloudonprem"}
	}

	for _, pkgName := range pkgList {
		procLog.Info.Printf("[VENV-Base-PKG] Default pkg install... [%s]\n", pkgName)
		pkgCmd = fmt.Sprintf("%s %s", pkgLink, pkgName)
		cmd_run := exec.Command(svcInfo.BaseCmd[0], svcInfo.BaseCmd[1], pkgCmd)
		sdtout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			procLog.Error.Printf("[VENV-Base-PKG] Default pkg installed, Error: [%v] %s \n", cmd_err, sdtout)
		}
	}
	procLog.Info.Printf("[VENV-Base-PKG] Complate package installed.\n")
}

// DownloadWeight function download model weight file from storage. Weight file used in inference app.
// Weight file store objectstorage. So, this function need accesskey and secretkey about objectstorage.
//
// Input:
//   - bwcFramework: The name of the virtual environment.
//   - appDir: The IP address of SDT Cloud.
//   - minioURL: The port number of the code repository.
//
// Output:
//   - error: An error message if weight file download fail.
func DownloadWeight(bwcFramework sdtType.Framework, appDir string, minioURL string) error {
	endpoint := minioURL
	minioKey := bwcFramework.Inference.AccessKey
	minioSecret := bwcFramework.Inference.SecretKey
	procLog.Warn.Printf("KEY: %s / SCRECT: %s \n", minioKey, minioSecret)
	useSSL := false

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioKey, minioSecret, ""),
		Secure: useSSL,
	})
	if err != nil {
		procLog.Error.Printf("Failed access minio: %v\n", err)
		return err
	}

	// Set minio bucket and file.
	bucketName := bwcFramework.Inference.Bucket
	objectName := fmt.Sprintf("%s/%s", bwcFramework.Inference.Path, bwcFramework.Inference.WeightFile)
	filePath := fmt.Sprintf("%s/weights/%s", appDir, bwcFramework.Inference.WeightFile)

	// Download model file.
	procLog.Info.Printf("Download model file from object storage.\n")
	err = minioClient.FGetObject(context.Background(), bucketName, objectName, filePath, minio.GetObjectOptions{})
	if err != nil {
		procLog.Error.Printf("Failed download model: %v\n", err)
		return err
	}

	procLog.Info.Printf("Successfully downloaded %s to %s\n", objectName, filePath)
	return nil
}

func DownloadWeight_new(modelUrl string, appName string, appId string, fileName string, appPath string) error {
	if fileName == "" {
		procLog.Error.Println("Filename is null: ")
		return errors.New("Filename is null.")
	}
	weighFile := fmt.Sprintf("%s/%s_%s/%s", appPath, appName, appId, fileName)
	procLog.Info.Printf("Weight File: %s.\n", weighFile)
	file, err := os.Create(weighFile)
	if err != nil {
		procLog.Error.Println("FileDownload file creation error: ", err)
		return err
	}
	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			procLog.Error.Println("Cannot connect http error: ", err)
			return nil
		},
	}

	resp, err := client.Get(modelUrl)
	if err != nil {
		procLog.Error.Println("FileDownload URL get file error: ", err)
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(file, resp.Body)

	defer file.Close()

	procLog.Info.Printf("Successfully downloaded %s to %s\n", fileName, appName)
	return nil
}

// CreateInferenceDir function create directory that inference needed. Created directory is weights, result, logs, data.
// Weights directory store weight file. Result directory store result of inference. Logs directory store log of app.
// Data directory store input data for inference.
//
// Input:
//   - appDir: The IP address of SDT Cloud.
//
// Output:
//   - error: An error message if creation directory fail.
func CreateInferenceDir(appDir string) error {
	procLog.Info.Printf("[Deploy] Make inference directory(weights, result, logs, data).\n")
	weightFile := fmt.Sprintf("%s/weights", appDir)
	err := os.MkdirAll(weightFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}
	resultFile := fmt.Sprintf("%s/result", appDir)
	err = os.MkdirAll(resultFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}
	logsFile := fmt.Sprintf("%s/logs", appDir)
	err = os.MkdirAll(logsFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}
	dataFile := fmt.Sprintf("%s/data", appDir)
	err = os.MkdirAll(dataFile, os.ModePerm)
	if err != nil {
		procLog.Error.Printf("Error creating destination directory: %v\n", err)
		return err
	}

	return nil
}

func CheckExistApp(targetApp string, rootPath string) bool {
	procLog.Info.Printf("Check app exist.\n")
	appInfoFile := fmt.Sprintf("%s/device.config/app.json", rootPath)

	if _, err := os.Stat(appInfoFile); os.IsNotExist(err) {
		return false
	}

	jsonFile, err := ioutil.ReadFile(appInfoFile)
	if err != nil {
		procLog.Error.Printf("Failed load app's file: %v\n", err)
		os.Exit(1)
	}

	var jsonData sdtType.AppConfig
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("Failed delete app's Unmarshal: %v\n", err)
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

func GetLogsApp(appPath string, appName string, appId string) string {
	procLog.Info.Printf("Get app logs.\n")
	var fileName, logContent string
	// Get appId.
	fileName = fmt.Sprintf("%s/%s_%s/app-error.log", appPath, appName, appId)

	// open logfile.
	file, err := os.Open(fileName)
	if err != nil {
		procLog.Error.Printf("Failed get logs: %s\n", err)
		return ""
	}
	defer file.Close()

	// Print logfile.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		logContent += scanner.Text() + "\n"
	}

	// Check Error.
	if err := scanner.Err(); err != nil {
		procLog.Error.Printf("Failed scanner logs: %s\n", err)
		return ""
	}
	return logContent
}
