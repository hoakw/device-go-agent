package aquarack

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"time"

	"github.com/mholt/archiver"
)

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
func Deploy(zipUrl string, codeRepoIp string, codeRepoPort int, deviceType string) error {

	var installPath = "/etc/sdt/aquaApp"

	app := "aquarack-sensor-collector"
	//fileUrl := "http://cloud-repo.sdt.services/1729752831421.khkim/aquarack-sensor-collector/archive/v1.0.9.zip"
	fileUrl := zipUrl
	appName := "aquarack-data-collector"
	userName := getCurrentUser()

	// 1. 설치 파일 다운로드
	filePath, cmd_err, fileZip := fileDownload(fileUrl, app, appName, installPath)
	if cmd_err != nil {
		fmt.Printf("[INIT] AquaRack Download Error: %v\n", cmd_err)
		return cmd_err
	}

	// 2. 아쿠아랙 수집 앱이 이미 있는지 체크
	// 파일 체크로 존재 확인하기
	//if CheckExistApp(appName, installPath) {
	if false {
		fmt.Printf("[INIT] AquaRack %s's app already exist.\n", appName)
		return errors.New("App already exist.")
	}

	// 3. 아쿠아랙 수집 앱 설치 시작
	fmt.Printf("[INIT] Deploy AquaRack data collector.")

	// 4. 아쿠아랙 필요 패키지 설치
	InstallPkg(codeRepoIp, codeRepoPort, userName, deviceType)
	CreatePythonService(filePath, appName, "main.py", userName)

	// start systemd
	startCmd := fmt.Sprintf("systemctl start %s", appName)
	cmd_run := exec.Command("sh", "-c", startCmd)
	stdout, cmd_err := cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Println("[INIT] Fail deploy: ", cmd_err, "\n", string(stdout))
		cmd_err = errors.New(string(stdout))
		return errors.New(string(stdout))
	}

	// enable systemd
	startCmd = fmt.Sprintf("systemctl enable %s", appName)
	cmd_run = exec.Command("sh", "-c", startCmd)
	stdout, cmd_err = cmd_run.CombinedOutput()
	if cmd_err != nil {
		fmt.Println("[INIT] Fail deploy: ", cmd_err, "\n", string(stdout))
		cmd_err = errors.New(string(stdout))
		return errors.New(string(stdout))
	}

	// save common deploy json

	//SaveAppInfo(appName, appId, venv, "systemd", sdtType.NewInferenceInfo(), "")
	fmt.Println("[INIT] End Deploy..")

	// remove zip file
	fmt.Println("[INIT] Remove ZIP File: ", fileZip)
	removeErr := os.Remove(fileZip)
	if removeErr != nil {
		time.Sleep(1)
		os.Remove(fileZip)
	}
	return cmd_err
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
	app string, //App's name
	appName string, // local app's name
	appDir string,
) (string, error, string) {
	// Build fileName from fullPath
	fileURL, err := url.Parse(fullURLFile)
	if err != nil {
		fmt.Println("[INIT] fileDownload URL parse error: ", err)
	}
	path := fileURL.Path
	segments := strings.Split(path, "/")
	fileName := segments[len(segments)-1]

	if _, err := os.Stat(appDir); os.IsNotExist(err) {
		os.Mkdir(appDir, os.ModePerm)
	}

	fileZip := fmt.Sprintf("%s/%s", appDir, fileName)
	file, err := os.Create(fileZip)
	if err != nil {
		fmt.Println("[INIT] fileDownload file creation error: ", err)
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
		fmt.Println("[INIT] fileDownload URL get file error: ", err)
	}
	defer resp.Body.Close()

	if resp.Status[:3] != "200" {
		return "", errors.New(fmt.Sprintf("Download error: %s", resp.Status)), ""
	}

	_, err = io.Copy(file, resp.Body)

	defer file.Close()

	// unzip!!
	appPath := fmt.Sprintf("%s/%s", appDir, appName)
	zipPath := fmt.Sprintf("%s/%s", appDir, app)
	targetPath := fmt.Sprintf("%s", appDir)

	err = archiver.Unarchive(fileZip, targetPath)
	if err != nil {
		fmt.Println("[INIT] Unzip error: ", err)
	}

	// file rename
	fmt.Println("[INIT] ZIP File Path: ", zipPath)
	fmt.Println("[INIT] APP File Path: ", appPath)
	os.Rename(zipPath, appPath)

	// remove zip file
	fmt.Println("[INIT] Remove ZIP File: ", fileZip)
	removeErr := os.Remove(fileZip)
	if removeErr != nil {
		time.Sleep(1)
		os.Remove(fileZip)
	}

	return appPath, nil, fileZip
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
func InstallPkg(codeRepoIp string, codeRepoPort int, userName string, deviceType string) {
	var pkgCmd, pipPath, pkgLink string
	var pkgList []string
	fmt.Printf("[VENV-Base-PKG] Install base package.\n")
	pipPath = fmt.Sprintf("/home/%s/miniconda3/bin/pip3", userName)

	// Set Pip Path
	pkgLink = fmt.Sprintf("%s install --trusted-host %s --index-url http://%s:%d/api/packages/app.manager/pypi/simple/", pipPath, codeRepoIp, codeRepoIp, codeRepoPort)

	if deviceType == "aquarack" {
		pkgList = []string{"sdtcloudonprem", "sdtclouds3", "sdtcloud", "sdtcloudwin"} //, "boto3", "AWSIoTPythonSDK", "awsiotsdk"}
	} else {
		pkgList = []string{"sdtcloudpubsub", "sdtclouds3", "sdtcloud", "sdtcloudwin"} //, "boto3", "AWSIoTPythonSDK", "awsiotsdk"}
	}

	// Install SDT Cloud's PKG
	for _, pkgName := range pkgList {
		fmt.Printf("[INIT] Default pkg install... [%s]\n", pkgName)
		pkgCmd = fmt.Sprintf("%s %s", pkgLink, pkgName)
		cmd_run := exec.Command("sh", "-c", pkgCmd)
		fmt.Println(cmd_run)
		fmt.Println(pkgCmd)
		sdtout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			fmt.Printf("[INIT] Default pkg installed, Error: [%v] %s \n", cmd_err, sdtout)
		}
	}
	fmt.Printf("[INIT] Complate SDTCloud package installed.\n")

	// Base PKG for SDTCloud
	// Set Pip Path
	pkgLink = fmt.Sprintf("%s install ", pipPath)

	pkgList = []string{"boto3", "AWSIoTPythonSDK", "awsiotsdk"}

	// Install base PKG for SDTCloud
	for _, pkgName := range pkgList {
		fmt.Printf("[INIT] Default pkg install... [%s]\n", pkgName)
		pkgCmd = fmt.Sprintf("%s %s", pkgLink, pkgName)
		cmd_run := exec.Command("sh", "-c", pkgCmd)
		fmt.Println(cmd_run)
		fmt.Println(pkgCmd)
		sdtout, cmd_err := cmd_run.CombinedOutput()
		if cmd_err != nil {
			fmt.Printf("[INIT] Default pkg installed, Error: [%v] %s \n", cmd_err, sdtout)
		}
	}
	fmt.Printf("[INIT] Complate Base package installed.\n")

	// Install collector's pkg.
	appPath := "/etc/sdt/aquaApp/aquarack-data-collector"
	envPath := fmt.Sprintf("/home/%s/miniconda3", userName)
	pkgFileName := fmt.Sprintf("%s/requirements.txt", appPath)
	fmt.Printf("[INIT] Install app's pkg.\n")

	// install pkg
	var outBuffer, errBuffer bytes.Buffer
	pkgCmd = fmt.Sprintf("%s/bin/pip install -r %s", envPath, pkgFileName)

	cmd_run := exec.Command("sh", "-c", pkgCmd)
	cmd_run.Stdout = &outBuffer
	cmd_run.Stderr = &errBuffer

	cmd_err := cmd_run.Run()

	if cmd_err != nil {
		errContent := errors.New(errBuffer.String())
		fmt.Printf("[INIT] Install package Error: %v, %v\n", cmd_err, errContent)
		return
	}
	fmt.Printf("[INIT] Installed V-ENV's package.\n")
}

// CreatePythonService function creates a systemd file (.service) for a Python application.
//
// Input:
//   - appDir: The directory on the device where the app will be installed.
//   - appName: The name of the application.
//   - appVenv: The virtual environment name for the application.
//   - runCmd: The command to execute the application.
func CreatePythonService(appDir string, appName string, runCmd string, userName string) {
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
	execBin = fmt.Sprintf("/home/%s/miniconda3/bin/python3", userName)

	// Write content to the file
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
		fmt.Println("Error writing to the file:", err)
		return
	}

	svcFile := fmt.Sprintf("/etc/systemd/system/%s.service", appName)
	err = CopyFile(filePath, svcFile)

	if err != nil {
		fmt.Println("Error svc file copy to systemd:", err)
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

func getCurrentUser() string {
	// sudo를 사용한 경우 원래 사용자의 이름 가져오기
	if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
		return sudoUser
	}

	// 일반적인 경우 현재 사용자 가져오기
	currentUser, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return currentUser.Username
}
