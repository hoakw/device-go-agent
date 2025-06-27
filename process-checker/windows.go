//go:build windows
// +build windows

// This package is the main package of the Process Checker. The Process Checker
// publishes MQTT messages to the cloud about resource usage of apps deployed on devices.
// Supported server architectures for this package include Windows and Linux.
package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	winSvc "golang.org/x/sys/windows/svc"

	sdtProcess "main/src/process"
	sdtType "main/src/processType"
)

// - procLog: This is the Struct that defines the format of the Log.
var (
	procLog sdtType.Logger
)

type winProcessService struct {
	MqttType string
	ArchType string
	RootPath string
	AppPath  string
}

// Struct defining the environment information of the Process-Chekcer agent.
//   - MqttType: MQTT service type used by the agent.
//   - ArchType: Architecture type of the device.
//   - RootPath: Root path of the BWC.
//   - RootPath: Path of the app.
type processService struct {
	MqttType string
	ArchType string
	RootPath string
	AppPath  string
}

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

// This function configures the environment based on the server's architecture information
// and executes core functions accordingly.
//
// Input:
//   - mqtt: Type of MQTT service, which can be aws (AWS IoT Core) or mosq (Mosquitto).
//     -- aws: AWS IoT Core
//     -- mosq: Mosquitto
//   - - exmq: EXMQ
//   - arch: Architecture of the device.
func main() {
	// Set parameter
	var mqttType, archType, rootPath, appPath string
	flag.StringVar(&mqttType, "mqtt", "", "Please input mqtt type(mosq? or aws? or exmq?)")
	flag.StringVar(&archType, "arch", "", "Please input architecture type(amd? or arm?)")
	flag.Parse()

	// Set Config PATH
	if archType == "win" {
		rootPath = "C:/sdt"
		appPath = "C:/sdt/app"
	} else {
		rootPath = "/etc/sdt"
		appPath = "/usr/local/sdt/app"
	}

	// Set Service Variable
	//svcInfo := processService{
	//	MqttType: mqttType,
	//	ArchType: archType,
	//	RootPath: rootPath,
	//	AppPath:  appPath,
	//}

	// Set Log
	logFilePath := fmt.Sprintf("%s/device.logs/process-checker.log", rootPath)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY|os.O_SYNC, 0644)
	if err != nil {
		panic(err)
	}
	defer logFile.Close()
	initError(logFile)

	sdtProcess.Getlog(procLog)

	// Set Service Variable
	winSvcInfo := winProcessService{
		MqttType: mqttType,
		ArchType: archType,
		RootPath: rootPath,
		AppPath:  appPath,
	}
	err = winSvc.Run("ProcessCheckerService", &winSvcInfo)
	if err != nil {
		procLog.Error.Printf("cannot start service: %v\n", err)
	}

	//sdtProcess.RunBody(svcInfo.MqttType, svcInfo.ArchType, svcInfo.RootPath, svcInfo.AppPath)
}

// svc.Handler 인터페이스 구현
func (srv *winProcessService) Execute(args []string, req <-chan winSvc.ChangeRequest, stat chan<- winSvc.Status) (svcSpecificEC bool, exitCode uint32) {
	stat <- winSvc.Status{State: winSvc.StartPending}

	// 실제 서비스 내용
	procLog.Info.Printf("[SVC] Service Content!!!\n")
	stopChan := make(chan bool, 1)
	go sdtProcess.RunBody(srv.MqttType, srv.ArchType, srv.RootPath, srv.AppPath)

	stat <- winSvc.Status{State: winSvc.Running, Accepts: winSvc.AcceptStop | winSvc.AcceptShutdown}

LOOP:
	for {
		// 서비스 변경 요청에 대해 핸들링
		switch r := <-req; r.Cmd {
		case winSvc.Stop, winSvc.Shutdown:
			stopChan <- true
			procLog.Warn.Printf("[SVC] Service Stop!!!\n")
			break LOOP

		case winSvc.Interrogate:
			procLog.Error.Printf("[SVC] Service Interrogate!!!\n")
			stat <- r.CurrentStatus
			time.Sleep(100 * time.Millisecond)
			stat <- r.CurrentStatus

			//case svc.Pause:
			//case svc.Continue:
		}
	}

	stat <- winSvc.Status{State: winSvc.StopPending}
	return
}
