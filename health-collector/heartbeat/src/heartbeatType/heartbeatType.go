// Package heartbeat defines the struct for heartbeat.
package heartbeatType

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
//   - MqttUrl: MQTT URL of SDT Cloud.
//   - ProjectCode: ID of the project to which the device belongs.
//   - ServiceCode: Service code of SDT Cloud.
type ConfigInfo struct {
	AssetCode   string `json:"assetcode"`
	MqttUrl     string `json:"mqtturl"`
	ProjectCode string `json:"projectcode"`
	ServiceCode string `json:"servicecode"`
	ServerIp    string `json:"serverip"`
}

// Struct definition for Heartbeat agent's environment information.
//   - MqttType: MQTT service type.
//   - ArchType: Device architecture.
//   - RootPath: Root path of BWC.
type HeartbeatService struct {
	MqttType string
	ArchType string
	RootPath string
}
