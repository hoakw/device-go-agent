// The config package provides functions to modify and update the configuration
// values of apps deployed on the device. It allows modifying the config values
// of apps from SDT Cloud and updating the current app's config to reflect
// changes made in SDT Cloud.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"

	sdtType "main/src/controlType"
)

// Global variables used in the config package.
// - procLog: Struct defining the format of logs.
// - floatType: Variable storing float type information.
// - stringType: Variable storing string type information.
var (
	procLog    sdtType.Logger
	floatType  = reflect.TypeOf(1.0)
	stringType = reflect.TypeOf("string")
)

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// The ToNumber function converts an Integer or Float type variable to a Json Number type.
//
// Input:
//   - f: An interface variable of type Integer or Float.
//
// Output:
//   - json.Number: A Json Number type variable.
func ToNumber(f interface{}) json.Number {
	var s string
	//if reflect.TypeOf(f) == floatType {
	if reflect.TypeOf(f).Kind() == reflect.Float64 {
		s = fmt.Sprintf("%.1f", f) // 1 decimal if integer
		return json.Number(s)
	} else if reflect.TypeOf(f).Kind() == reflect.String {
		s = f.(string)
	} else {
		s = fmt.Sprintf("%.1f", float64(f.(int)))
	}
	return json.Number(s)
}

func changeInterface(data map[string]interface{}, target map[string]interface{}) (map[string]interface{}, error) {
	var err error = nil

	for key, val := range data {
		if _, exist := target[key]; exist {
			if err != nil {
				return nil, err
			}

			paramType := reflect.TypeOf(target[key])

			if paramType.Kind() == reflect.Map {
				if reflect.TypeOf(val).Kind() == paramType.Kind() {
					target[key], err = changeInterface(val.(map[string]interface{}), target[key].(map[string]interface{}))
					continue
				} else {
					procLog.Error.Printf("Please check key's type.[key=%s]\n", key)
					err = errors.New(fmt.Sprintf("Check [%s] parameter", key))
					return nil, err
				}
			}

			// 수정하려는 값이 Object 타입인 경우
			if reflect.TypeOf(val).Kind() == reflect.Map {
				if paramType.Kind() != reflect.Map {
					procLog.Error.Printf("Please check key's type.[key=%s]\n", key)
					err = errors.New(fmt.Sprintf("Check %s parameter", key))
					return nil, err
				}
			}

			// 수정하려는 값이 문자열인 경우
			if reflect.TypeOf(val).Kind() == reflect.String {
				target[key] = val
				continue
			}

			// 수정하려는 값이 숫자인 경우(int,float 등)

			// 숫자인 경우(int,float 등)
			n, ok := target[key].(json.Number)
			if !ok {
				target[key] = val
			} else if _, err = n.Int64(); err == nil {
				target[key] = val
			} else if _, err = n.Float64(); err == nil {
				target[key] = ToNumber(val)
			}
		}
	}

	return target, err
}

//func AppidUpdate(appId string, fileName string, appName string) {
//	procLog.Info.Printf("[AppID-Update] exec... appid Update...\n")
//	targetFile := fmt.Sprintf("/usr/local/sdt/app/%s_%s/%s", appName, appId, fileName)
//	jsonFile, err := ioutil.ReadFile(targetFile)
//
//	if err != nil {
//		procLog.Error.Printf("[AppID-Update] Not found file Error: %v\n", err)
//	}
//
//	var jsonData map[string]interface{}
//	jsonRecode := json.NewDecoder(strings.NewReader(string(jsonFile)))
//	jsonRecode.UseNumber()
//	err = jsonRecode.Decode(&jsonData)
//	if err != nil {
//		procLog.Error.Printf("[AppID-Update] Unmarshal Error: %v\n", err)
//	}
//
//	jsonData["appId"] = appId
//
//	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
//	err = ioutil.WriteFile(targetFile, saveJson, 0644)
//	if err != nil {
//		procLog.Error.Printf("[AppID-Update] Marshal Error: %v\n", err)
//	}
//}

// The Rebooting function modifies the BWC config when rebooting the device.
// During reboot, it records in the BWC config that the device has been rebooted.
// Based on this record, it sends a boot completion message to the cloud after reboot.
//
// Input:
//   - rebootVal: Booting processing value.
//   - requestid: RequestID of the control command.
//
// Output:
//   - error: Cause of control processing failure.
//   - int: HTTP status code.
func Rebooting(rebootVal string, requestid string) (error, int) {
	procLog.Info.Printf("[REBOOT] exec... reboot\n")
	targetFile := "/etc/sdt/device.config/config.json"
	jsonFile, err := ioutil.ReadFile(targetFile)

	if err != nil {
		procLog.Error.Printf("[REBOOT] Not found file Error: %v\n", err)
		return err, http.StatusBadRequest
	}

	var jsonData map[string]interface{}
	jsonRecode := json.NewDecoder(strings.NewReader(string(jsonFile)))
	jsonRecode.UseNumber()
	err = jsonRecode.Decode(&jsonData)
	if err != nil {
		procLog.Error.Printf("[REBOOT] Unmarshal Error: %v\n", err)
		return err, http.StatusBadRequest
	}

	jsonData["reboot"] = rebootVal
	jsonData["requestid"] = requestid

	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
	err = ioutil.WriteFile(targetFile, saveJson, 0644)
	if err != nil {
		return err, http.StatusBadRequest
	}

	return nil, http.StatusOK
}

// The JsonChange function modifies the config information of an app deployed on the device.
// Apps deployed from SDT Cloud are managed alongside Json-formatted config files.
//
// Input:
//   - configCmd: Struct containing config modification command information.
//   - archType: Architecture of the device.
//
// Output:
//   - string: Requested config value (returned as a string after conversion).
//   - error: Cause of control processing failure.
//   - int: HTTP status code.
func JsonChange(configCmd sdtType.CmdJson, appPath string, cmdType string) (string, error, int) {
	// Set parameter
	appId := configCmd.AppId
	fileName := configCmd.FileName
	appName := configCmd.AppName
	paramData := configCmd.Parameter

	if configCmd.Parameter == nil {
		procLog.Info.Printf("[CONFIG] Not change. \n")
		return "", nil, http.StatusOK
	}

	if configCmd.FileName == "" {
		procLog.Info.Printf("[DEBUG] Completed. \n")
		return "", nil, http.StatusOK
	}

	procLog.Info.Printf("[CONFIG] ParmData: %s\n", paramData)

	// json read
	var targetFile string
	if cmdType == "controlAquarack" {
		targetFile = fmt.Sprintf("%s/%s", appPath, fileName)
	} else {
		targetFile = fmt.Sprintf("%s/%s_%s/%s", appPath, appName, appId, fileName)
	}
	//if archType == "win" {
	//	targetFile = fmt.Sprintf("C:/sdt/app/%s_%s/%s", appName, appId, fileName)
	//} else {
	//	targetFile = fmt.Sprintf("/usr/local/sdt/app/%s_%s/%s", appName, appId, fileName)
	//}

	jsonFile, err := ioutil.ReadFile(targetFile)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Not found file Error: %v\n", err)
		return "", err, http.StatusBadRequest
	}
	var jsonData map[string]interface{}
	jsonRecode := json.NewDecoder(strings.NewReader(string(jsonFile)))
	jsonRecode.UseNumber()
	err = jsonRecode.Decode(&jsonData)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Unmarshal Error: %v\n", err)
		return "", err, http.StatusBadRequest
	}

	// Requirement
	// - Float 1 값을 1.0으로 표현해야 합니다.
	jsonData, err = changeInterface(paramData, jsonData)
	if err != nil {
		fmt.Printf("Found error. -> %v \n", err)
		return "", err, http.StatusBadRequest
	}

	//for key, val := range paramData {
	//	if _, exist := jsonData[key]; exist {
	//		n, ok := jsonData[key].(json.Number)
	//		if !ok {
	//			jsonData[key] = val
	//			continue
	//		} else if _, err = n.Int64(); err == nil {
	//			jsonData[key] = val
	//			continue
	//		} else if _, err = n.Float64(); err == nil {
	//			jsonData[key] = ToNumber(val)
	//			continue
	//		}
	//	}
	//}

	saveJson, _ := json.MarshalIndent(&jsonData, "", "\t")
	err = ioutil.WriteFile(targetFile, saveJson, 0644)
	if err != nil {
		return "", err, http.StatusBadRequest
	}

	jsonBytes, _ := json.Marshal(paramData) // JSON ENCODING
	jsonString := string(jsonBytes)
	return jsonString, nil, http.StatusOK
}

// The GetConfig function retrieves the config information of an app deployed on the device.
//
// Input:
//   - appId: ID of the app.
//   - appName: Name of the app.
//   - archType: Architecture of the device.
//
// Output:
//   - map[string]interface{}: Config values of the app.
//   - error: Cause of control processing failure.
//   - int: HTTP status code.
func GetConfig(appId string, appName string, appPath string, cmdType string) (map[string]interface{}, error, int) {
	procLog.Info.Printf("[CONFIG] Get config: %s\n", appName)

	// json read
	var targetDir string
	if cmdType == "controlAquarack" || cmdType == "getConfigAquarack" {
		targetDir = fmt.Sprintf("%s/%s", appPath, appName)
	} else {
		targetDir = fmt.Sprintf("%s/%s_%s", appPath, appName, appId)
	}

	fileList, err := ioutil.ReadDir(targetDir)
	if err != nil {
		procLog.Warn.Printf("[CONFIG] Not found App Error: %v\n", err)
		return nil, err, http.StatusOK
	}

	var allConfig = make(map[string]interface{})
	for _, file := range fileList {
		fileName := fmt.Sprintf("%s", file.Name())
		fileType := strings.Split(fileName, ".")

		if fileType[len(fileType)-1] == "json" {
			result := GetJson(fileName, targetDir)
			allConfig[fileName] = result
		}

	}
	// Not found config file.
	if len(allConfig) == 0 {
		return allConfig, nil, http.StatusNoContent
	}

	return allConfig, nil, http.StatusOK
}

// The GetJson function reads a JSON file.
//
// Input:
//   - appId: ID of the app.
//   - appName: Name of the app.
//   - fileName: Name of the JSON file to read.
//   - targetDir: Directory where the JSON file is located (app location).
//
// Output:
//   - map[string]interface{}: Config values of the app.
func GetJson(fileName string, targetDir string) map[string]interface{} {
	targetFile := fmt.Sprintf("%s/%s", targetDir, fileName)
	jsonFile, err := ioutil.ReadFile(targetFile)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Not found file Error: %v\n", err)
	}
	var jsonData map[string]interface{}
	err = json.Unmarshal(jsonFile, &jsonData)
	if err != nil {
		procLog.Error.Printf("[CONFIG] Unmarshal Error: %v\n", err)
	}

	return jsonData
}

// The JsonAdd function adds values to a JSON file.
//
// Input:
//   - fileName: Name of the JSON file to modify.
//   - appName: Name of the app.
//   - paramData: JSON data to add.
//
// Output:
//   - string: Modified Config value (returned as a string).
//   - error: Error encountered during the operation.
//   - int: HTTP status code.
func JsonAdd(fileName string, appName string, paramData map[string]interface{}) (string, error, int) {
	// json read
	targetFile := fmt.Sprintf("/usr/local/sdt/app/%s/%s", appName, fileName)
	if _, err := os.Stat(targetFile); os.IsExist(err) {
		procLog.Error.Printf("[CONFIG] Found file Error: %v\n", err)
		return "", err, http.StatusBadRequest
	}

	saveJson, _ := json.MarshalIndent(&paramData, "", " ")
	err := ioutil.WriteFile(targetFile, saveJson, 0644)
	if err != nil {
		return "", err, http.StatusBadRequest
	}

	jsonBytes, _ := json.Marshal(paramData) // JSON ENCODING
	jsonString := string(jsonBytes)
	return jsonString, nil, http.StatusOK
}
