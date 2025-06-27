package main

import (
	"os"
	"io"
	"fmt"
	"log"
	"flag"
	"time"

	winSvc "golang.org/x/sys/windows/svc"

	sdtHeartbeat "main/src/heartbeat"
	sdtType "main/src/heartbeatType"
)

var (
	procLog	sdtType.Logger
)

type heartbeatService struct {
	MqttType	string
	ArchType	string
	RootPath	string
}


func initError(logFile io.Writer) {
	procLog.Info = log.New(logFile, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Warn = log.New(logFile, "[WARNING] ", log.Ldate|log.Ltime|log.Lshortfile)
	procLog.Error = log.New(logFile, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)
}

// svc.Handler 인터페이스 구현
func (srv *heartbeatService) Execute(args []string, req <-chan winSvc.ChangeRequest, stat chan<- winSvc.Status) (svcSpecificEC bool, exitCode uint32) {
    stat <- winSvc.Status{State: winSvc.StartPending}
 
    // 실제 서비스 내용
	procLog.Info.Printf("[SVC] Service Content!!!\n")
    stopChan := make(chan bool, 1)
    go sdtHeartbeat.RunBody(srv.MqttType, srv.ArchType, srv.RootPath)
 
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

func main() {
	// Set parameter
	var mqttType, archType, rootPath string
	flag.StringVar(&mqttType, "mqtt", "", "Please input mqtt type(mosq? or aws? or exmq?)")
	flag.StringVar(&archType, "arch", "", "Please input architecture type(amd? or arm?)")
	flag.Parse()

	// Set Config PATH
	if archType == "win"{
		rootPath = "C:/sdt"
	} else {
		rootPath = "/etc/sdt"
	}

	// Set Service Variable 
	svcInfo := heartbeatService{
		MqttType:	mqttType,
		ArchType:	archType,
		RootPath:	rootPath,
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

	err = winSvc.Run("DeviceHeartbeatService", &svcInfo)
	if err != nil {
		panic(err)
	}
}