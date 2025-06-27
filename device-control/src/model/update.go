// The Model package handles model of applications on the device.
// Model will install to "/usr/local/sdt/app/{appName_appId}/weights" directory on the device.
package update

import (
	"errors"
	sdtConfig "main/src/config"
	sdtType "main/src/controlType"
	sdtDeploy "main/src/deploy"
	"net/http"
	"strings"
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
func Update(modelData sdtType.CmdModel, svcInfo sdtType.ControlService) (map[string]interface{}, error, int) {
	var cmdErr error
	var statusCode int
	var fileName string = ""
	var jsonResult, fileDict map[string]interface{}

	// 1. 앱 종료
	procLog.Warn.Printf("[MODEL] App stop: %s\n", modelData.AppName)
	_, cmdErr, statusCode = sdtDeploy.Stop(modelData.AppName, modelData.AppId, svcInfo.ArchType, svcInfo)
	if cmdErr != nil {
		procLog.Error.Printf("[MODEL] Failed app stop: %s\n", cmdErr)
		return nil, cmdErr, statusCode
	}

	// 2. 모델 다운로드
	procLog.Warn.Printf("[MODEL] Download app's weight file s: %s\n", modelData.AppName)
	// 모델 파일 이름 가져오기
	if modelData.ModelFileKey == "" {
		procLog.Error.Printf("[MODEL] Cannot found 'modelFileKey' Key.\n")
		return nil, errors.New("Cannot found 'modelFileKey' Key."), http.StatusBadRequest
	} else {
		keys := strings.Split(modelData.ModelFileKey, ".")
		fileDict = modelData.Parameter
		for n := 0; n < len(keys)-1; n++ {
			fileDict = fileDict[keys[n]].(map[string]interface{})
		}

		fileName = fileDict[keys[len(keys)-1]].(string)

	}

	// TODO: 기존 모델 삭제?? -> 체크 필요
	cmdErr = sdtDeploy.DownloadWeight_new(modelData.ModelUrl, modelData.AppName, modelData.AppId, fileName, svcInfo.AppPath)
	if cmdErr != nil {
		procLog.Error.Printf("[MODEL] Failed download: %s\n", cmdErr)
		return nil, cmdErr, http.StatusBadRequest
	}

	// 3. Config 수정
	procLog.Warn.Printf("[MODEL] Fix app's config: %s\n", modelData.AppName)
	// Parameter
	parameter := sdtType.CmdJson{
		AppId:     modelData.AppId,
		AppName:   modelData.AppName,
		FileName:  "config.json", // BW와 약속한 config 파일 이름
		Parameter: modelData.Parameter,
	}

	_, configErr, _ := sdtConfig.JsonChange(parameter, svcInfo.AppPath, "")
	if configErr != nil {
		procLog.Error.Printf("[MODEL] Failed fixing parameter.\n")
		return nil, configErr, http.StatusBadRequest
	}
	// 현재 Cofing 값 가져오기
	jsonResult, configErr, _ = sdtConfig.GetConfig(modelData.AppId, modelData.AppName, svcInfo.AppPath, "")
	if configErr != nil {
		procLog.Error.Printf("[MODEL] Failed get parameter.\n")
		return nil, configErr, http.StatusBadRequest
	}

	// 4. 앱 실행
	procLog.Warn.Printf("[MODEL] App start: %s\n", modelData.AppName)
	_, cmdErr, statusCode = sdtDeploy.Start(modelData.AppName, modelData.AppId, svcInfo.ArchType, svcInfo)
	if cmdErr != nil {
		procLog.Error.Printf("[MODEL] Failed app stop: %s\n", cmdErr)
		return nil, cmdErr, statusCode
	}

	return jsonResult, nil, http.StatusOK
}
