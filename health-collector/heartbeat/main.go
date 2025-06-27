//go:build linux
// +build linux

// This package is the main package for heartbeat. Heartbeat updates the device's
// heartbeat status by sending its state to the cloud via MQTT messages.
// This package supports server architectures including Windows and Linux.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	sdtHeartbeat "main/src/heartbeat"
	sdtType "main/src/heartbeatType"
)

// - procLog: This is the Struct that defines the format of the Log.
var (
	procLog sdtType.Logger
)

// initError defines and initializes the log format. The log formats are defined as Info,
// Warn, and Error, and the output is done using Printf. If you call the function to define and
// initialize the log formats, you can use it to output logs as follows:
// The way to log the text "Hello World" as an Info log type is shown below.
// procLog.Info.Printf("Hello World\n")
// Output: [INFO] Hello World
func initError(logFile io.Writer) {
	procLog.Info = log.New(logFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Warn = log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Error = log.New(logFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// This function receives server architecture information, configures the environment
// accordingly, and executes core functions.
//
// Input:
//   - mqtt: Type of MQTT service, which can be aws (AWS IoT Core) or mosq (Mosquitto).
//     -- aws: AWS IoT Core
//     -- mosq: Mosquitto
//   - - exmq: EXMQ
//   - arch: Architecture of the device.
func main() {
	// Set parameter
	var mqttType, archType, rootPath string
	flag.StringVar(&mqttType, "mqtt", "", "Please input mqtt type(mosq? or aws? or exmq?)")
	flag.StringVar(&archType, "arch", "", "Please input architecture type(amd? or arm?)")
	flag.Parse()

	// Set Config PATH
	if archType == "win" {
		rootPath = "C:/sdt"
	} else {
		rootPath = "/etc/sdt"
	}

	// Set Service Variable
	svcInfo := sdtType.HeartbeatService{
		MqttType: mqttType,
		ArchType: archType,
		RootPath: rootPath,
	}

	// Set logger
	logFilePath := fmt.Sprintf("%s/device.logs/device-heartbeat.log", rootPath)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	initError(logFile)

	sdtHeartbeat.Getlog(procLog)
	sdtHeartbeat.RunBody(svcInfo.MqttType, svcInfo.ArchType, svcInfo.RootPath)
}
