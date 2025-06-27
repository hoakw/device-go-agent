// The process type package defines the structs for processes.
package processType

import (
	"log"
)

// Struct defining the format of logs.
//   - Warn: Warning log type.
//   - Info: General information log type.
//   - Error: Error type information.
type Logger struct {
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

// ConfigInfo struct defines the configuration file used by SDT Cloud on the device.
//   - AssetCode: Device serial number.
//   - DeviceType: Type of device.
//   - MqttUrl: MQTT URL of SDT Cloud.
//   - ProjectCode: ID of the project to which the device belongs.
//   - ServiceCode: Service code of SDT Cloud.
type ConfigInfo struct {
	AssetCode   string `json:"assetcode"`
	DeviceType  string `json:"devicetype"`
	MqttUrl     string `json:"mqtturl"`
	ProjectCode string `json:"projectcode"`
	ServiceCode string `json:"servicecode"`
	ServerIp    string `json:"serverip"`
}

// Struct defining configuration information for managing app metadata on the device.
//   - AppInfoList: List variable of AppInfo Struct.
type AppConfig struct {
	AppInfoList []AppInfo `json: "appInfo"`
}

// AppInfo defines the structure for application metadata information.
//   - AppName: Name of the application.
//   - AppId: ID of the application.
//   - AppVenv: Virtual environment used by the application.
type AppInfo struct {
	AppName string `json:"AppName"`
	AppId   string `json:"AppId"`
	AppVenv string `json:"AppVenv"`
	Managed string `json:"Managed"`
}
