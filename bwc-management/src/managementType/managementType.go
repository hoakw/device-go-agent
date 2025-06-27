// The BWC Management package defines the struct for heartbeat.
package managementType

import (
	"log"
)

// ConfigInfo struct defines the configuration file used by SDT Cloud on the device.
//   - AssetCode: Device serial number.
//   - MqttUrl: MQTT URL of SDT Cloud.
//   - ProjectCode: ID of the project to which the device belongs.
//   - ServiceCode: Service code of SDT Cloud.
type ConfigInfo struct {
	AssetCode   string `json:"assetcode"`
	MqttUrl     string `json:"mqtturl"`
	ProjectCode string `json:"projectcode"`
	ServiceCode string `json:"servicecode"`
	ServerIp    string `json:"serverip"`
	DeviceType  string `json:"devicetype"`
}

// Struct defining the format of logs.
//   - Warn: Warning log type.
//   - Info: General information log type.
//   - Error: Error type information.
type Logger struct {
	Warn  *log.Logger
	Info  *log.Logger
	Error *log.Logger
}

// Project struct defines the information related to a project.
//   - ProjectCode: Project ID to which the device belongs.
//   - RootCA: File path of the Root CA for the project.
//   - PrivateKey: File path of the Private Key for the project.
//   - Cert: File path of the Certificate for the project.
type ProjectControl struct {
	ProjectCode string `json: "projectCode"`
	RootCA      string `json: "rootCA"`
	PrivateKey  string `json: "privateKey"`
	Cert        string `json: "cert"`
}

// Struct defining the environment information of the BWC-Management agent.
//   - MqttType: MQTT service type used by the agent.
//   - ArchType: Architecture type of the device.
//   - RootPath: Root path of the BWC.
type ManagementService struct {
	MqttType string
	ArchType string
	RootPath string
}
