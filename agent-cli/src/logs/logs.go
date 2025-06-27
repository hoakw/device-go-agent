// The Logs package handles logging for the BWC agent on the device and deployed apps.
// The agent's logs are stored in '/etc/sdt/device.log', while app logs are stored in
// the app directory located at '/usr/local/sdt/app'.
package logs

import (
	"bufio"
	"fmt"
	"github.com/hpcloud/tail"
	"io/ioutil"
	sdtType "main/src/cliType"
	"os"

	sdtUtil "main/src/util"
)

// These are the global variables used in the Logs package.
// - procLog: This is the struct that defines the format of the log.
var procLog sdtType.Logger

// Getlog is a function that loads the log format. Log formats are defined as Info,
// Warn, Error, and are output using Printf.
// Here's how you would record the text "Hello World" as an Info log type:
//
//	procLog.Info.Printf("Hello World\n")   ->   [INFO] Hello World
func Getlog(logConfig sdtType.Logger) {
	procLog = logConfig
}

// GetLogsTail function prints the logs of the BWC agent in a tailing manner.
// Tailing means continuously outputting the stored logs.
//
// Input:
//   - targetName: The name of the object whose logs should be printed.
func GetLogsTail(targetName string) {
	procLog.Info.Printf("Get tail logs.\n")
	filePath := fmt.Sprintf("/etc/sdt/device.logs/%s.log", targetName)
	if _, err := os.Stat(filePath); err != nil {
		procLog.Error.Printf("%s not found.\n", targetName)
		return
	}

	// Tail 설정
	config := tail.Config{
		ReOpen:    true,
		MustExist: false,
		Poll:      true,
		Follow:    true,
	}

	// Tail 생성
	t, err := tail.TailFile(filePath, config)
	if err != nil {
		procLog.Error.Printf("%s Not found.\n", targetName)
		return
	}

	// 실시간으로 파일의 변경을 감지하여 출력
	for line := range t.Lines {
		fmt.Println(line.Text)
	}
	procLog.Info.Printf("Successfully get tail logs.\n")
}

// GetLogs function prints the logs of the BWC agent.
//
// Input:
//   - targetName: The name of the object whose logs should be printed.
//   - numLine: The number of lines of logs to print.
func GetLogs(targetName string, numLine int) {
	procLog.Info.Printf("Get logs.\n")
	filePath := fmt.Sprintf("/etc/sdt/device.logs/%s.log", targetName)

	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		procLog.Error.Printf("%s not found.\n", targetName)
		return
	}
	if numLine == 0 {
		fmt.Println(string(content))
	} else {
		lines := sdtUtil.ReadLastNLines(string(content), numLine)

		for _, line := range lines {
			fmt.Println(line)
		}
	}
	procLog.Info.Printf("Successfully get logs.\n")
}

// GetLogsApp function prints the logs of an application.
// The application can be installed either through execution tests or deployment.
// Execution test logs are located in '/etc/sdt/execute'.
// Logs of deployed applications are located in '/usr/local/sdt/app'.
//
// Input:
//   - appName: The name of the application.
//   - appId: The ID of the application.
func GetLogsApp(appName string, appId string) {
	procLog.Info.Printf("Get app logs.\n")
	var fileName string
	// Get appId.
	if appId == "" {
		procLog.Warn.Printf("%s's app not found.\n", appName)
		procLog.Warn.Printf("Check execution app.\n")
		fileName = fmt.Sprintf("/etc/sdt/execute/%s/app-error.log", appName)
	} else {
		fileName = fmt.Sprintf("/usr/local/sdt/app/%s_%s/app-error.log", appName, appId)
	}

	// open logfile.
	file, err := os.Open(fileName)
	if err != nil {
		procLog.Error.Printf("Failed get logs: %s\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Print logfile.
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	// Check Error.
	if err := scanner.Err(); err != nil {
		procLog.Error.Printf("Failed scanner logs: %s\n", err)
		os.Exit(1)
	}
}
